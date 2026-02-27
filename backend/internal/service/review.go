package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/codeMaster/backend/internal/gitops"
	"github.com/codeMaster/backend/internal/model"
	"github.com/codeMaster/backend/internal/notify"
	"github.com/codeMaster/backend/internal/review"
	"github.com/codeMaster/backend/pkg/encrypt"
	"gorm.io/gorm"
)

type ReviewService struct {
	db         *gorm.DB
	aiReviewer *review.AIReviewer
	aesKey     string
	workDir    string
	notifier   notify.Notifier
}

func NewReviewService(db *gorm.DB, aesKey, workDir string) *ReviewService {
	return &ReviewService{
		db:         db,
		aiReviewer: review.NewAIReviewer(db),
		aesKey:     aesKey,
		workDir:    workDir,
	}
}

// SetNotifier sets the notifier for sending notifications after review events.
func (s *ReviewService) SetNotifier(n notify.Notifier) {
	s.notifier = n
}

func (s *ReviewService) TriggerAIReview(ctx context.Context, taskID uint, reviewerIDs []uint, userID uint) (*model.CodeReview, error) {
	var task model.CodegenTask
	if err := s.db.Preload("Repository").First(&task, taskID).Error; err != nil {
		return nil, err
	}
	if task.Status != "completed" {
		return nil, fmt.Errorf("40003:生成任务尚未完成，无法 Review")
	}

	// Check if already running
	var existing model.CodeReview
	err := s.db.Where("codegen_task_id = ? AND ai_status = ?", taskID, "running").First(&existing).Error
	if err == nil {
		return nil, fmt.Errorf("40003:AI Review 正在进行中")
	}

	rev := &model.CodeReview{
		CodegenTaskID: taskID,
		ReviewerIDs:   model.JSONUintArray(reviewerIDs),
		AIStatus:      "pending",
		HumanStatus:   "pending",
		MergeStatus:   "none",
	}
	if err := s.db.Create(rev).Error; err != nil {
		return nil, err
	}

	go s.runAIReview(rev, &task, s.getUserGitToken(userID))
	return rev, nil
}

func (s *ReviewService) runAIReview(rev *model.CodeReview, task *model.CodegenTask, gitToken string) {
	ctx := context.Background()
	workDir := filepath.Join(s.workDir, "review", strconv.FormatUint(uint64(task.ID), 10))
	defer os.RemoveAll(workDir)

	// Resolve token: prefer user's personal token, fall back to repo's stored token
	token := gitToken
	if token == "" {
		var err error
		token, err = encrypt.AESDecrypt(s.aesKey, task.Repository.AccessToken)
		if err != nil {
			s.db.Model(rev).Update("ai_status", "failed")
			return
		}
	}

	if err := gitops.Clone(ctx, task.Repository.GitURL, token, task.TargetBranch, workDir); err != nil {
		s.db.Model(rev).Update("ai_status", "failed")
		return
	}

	diffContent, _ := gitops.GetDiffContent(ctx, workDir, task.SourceBranch, task.TargetBranch, "")
	s.aiReviewer.RunReview(ctx, rev, workDir, diffContent)

	// Notify after AI review completes
	if s.notifier != nil {
		var updatedRev model.CodeReview
		if s.db.First(&updatedRev, rev.ID).Error == nil && updatedRev.AIStatus != "pending" && updatedRev.AIStatus != "running" {
			var req model.Requirement
			if s.db.Preload("Creator").Preload("Assignee").Preload("Project").First(&req, task.RequirementID).Error == nil {
				creatorOpenID := ""
				if req.Creator != nil {
					creatorOpenID = req.Creator.FeishuUID
				}
				assigneeOpenID := ""
				if req.Assignee != nil {
					assigneeOpenID = req.Assignee.FeishuUID
				}
				projectName := ""
				if req.Project != nil {
					projectName = req.Project.Name
				}
				go s.notifier.NotifyAIReviewCompleted(context.Background(), notify.AIReviewCompletedEvent{
					RequirementID:  req.ID,
					Title:          req.Title,
					ProjectName:    projectName,
					ReviewID:       rev.ID,
					CreatorOpenID:  creatorOpenID,
					AssigneeOpenID: assigneeOpenID,
					AIScore:        updatedRev.AIScore,
					AIStatus:       updatedRev.AIStatus,
				})
			}
		}
	}
}

func (s *ReviewService) GetReview(codegenTaskID uint) (*model.CodeReview, error) {
	var rev model.CodeReview
	if err := s.db.Where("codegen_task_id = ?", codegenTaskID).
		Preload("HumanReviewer").
		Order("created_at desc").First(&rev).Error; err != nil {
		return nil, err
	}
	return &rev, nil
}

func (s *ReviewService) GetReviewByID(id uint) (*model.CodeReview, error) {
	var rev model.CodeReview
	if err := s.db.Preload("HumanReviewer").Preload("CodegenTask.Requirement").Preload("CodegenTask.Repository").First(&rev, id).Error; err != nil {
		return nil, err
	}
	// Fill reviewers from ReviewerIDs
	if len(rev.ReviewerIDs) > 0 {
		var users []*model.User
		s.db.Where("id IN ?", []uint(rev.ReviewerIDs)).Find(&users)
		rev.Reviewers = users
	}
	return &rev, nil
}

func (s *ReviewService) ListPendingReviews(userID uint, projectID *uint, page, pageSize int) ([]model.CodeReview, int64, error) {
	return s.ListReviews(userID, "pending", projectID, page, pageSize)
}

func (s *ReviewService) ListReviews(userID uint, humanStatus string, projectID *uint, page, pageSize int) ([]model.CodeReview, int64, error) {
	query := s.db.Model(&model.CodeReview{}).
		Joins("JOIN codegen_tasks ON code_reviews.codegen_task_id = codegen_tasks.id").
		Joins("JOIN requirements ON codegen_tasks.requirement_id = requirements.id").
		Where("code_reviews.ai_status IN ?", []string{"passed", "warning", "failed"})

	switch humanStatus {
	case "pending":
		query = query.Where("code_reviews.human_status IN ?", []string{"pending", "needs_revision"})
	case "approved":
		query = query.Where("code_reviews.human_status = ?", "approved")
	case "rejected":
		query = query.Where("code_reviews.human_status = ?", "rejected")
	// default: no human_status filter — return all
	}

	if projectID != nil {
		query = query.Where("requirements.project_id = ?", *projectID)
	}

	// Only show reviews where user is a member of the project
	query = query.Where("requirements.project_id IN (SELECT project_id FROM project_members WHERE user_id = ?)", userID)

	var total int64
	query.Count(&total)

	var reviews []model.CodeReview
	if err := query.Preload("CodegenTask.Requirement.Project").Preload("CodegenTask.Requirement.Creator").Preload("CodegenTask.Requirement").Preload("CodegenTask.Repository").Preload("HumanReviewer").
		Order("code_reviews.created_at desc").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&reviews).Error; err != nil {
		return nil, 0, err
	}
	return reviews, total, nil
}

func (s *ReviewService) SubmitHumanReview(reviewID, reviewerID uint, comment, status string) (*model.CodeReview, error) {
	var rev model.CodeReview
	if err := s.db.First(&rev, reviewID).Error; err != nil {
		return nil, err
	}
	if rev.AIStatus == "running" || rev.AIStatus == "pending" {
		return nil, fmt.Errorf("40003:AI Review 尚未完成，请等待")
	}

	updates := map[string]interface{}{
		"human_reviewer_id": reviewerID,
		"human_comment":     comment,
		"human_status":      status,
	}
	if err := s.db.Model(&rev).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Update requirement status based on review result
	var task model.CodegenTask
	s.db.First(&task, rev.CodegenTaskID)

	if status == "approved" {
		s.db.Model(&model.Requirement{}).Where("id = ?", task.RequirementID).Update("status", "approved")
	} else if status == "rejected" {
		s.db.Model(&model.Requirement{}).Where("id = ?", task.RequirementID).Update("status", "rejected")
	}

	// Notify human review submitted
	if s.notifier != nil {
		var req model.Requirement
		if s.db.Preload("Creator").Preload("Assignee").Preload("Project").First(&req, task.RequirementID).Error == nil {
			creatorOpenID := ""
			if req.Creator != nil {
				creatorOpenID = req.Creator.FeishuUID
			}
			assigneeOpenID := ""
			if req.Assignee != nil {
				assigneeOpenID = req.Assignee.FeishuUID
			}
			projectName := ""
			if req.Project != nil {
				projectName = req.Project.Name
			}
			var reviewer model.User
			reviewerName := ""
			if s.db.First(&reviewer, reviewerID).Error == nil {
				reviewerName = reviewer.Name
			}
			go s.notifier.NotifyHumanReviewSubmitted(context.Background(), notify.HumanReviewSubmittedEvent{
				RequirementID:  req.ID,
				Title:          req.Title,
				ProjectName:    projectName,
				ReviewID:       reviewID,
				CreatorOpenID:  creatorOpenID,
				AssigneeOpenID: assigneeOpenID,
				ReviewerName:   reviewerName,
				Status:         status,
				Comment:        comment,
			})
		}
	}

	return s.GetReviewByID(reviewID)
}

func (s *ReviewService) CreateMergeRequest(reviewID uint, userID uint) (*model.CodeReview, error) {
	rev, err := s.GetReviewByID(reviewID)
	if err != nil {
		return nil, err
	}
	if rev.HumanStatus != "approved" {
		return nil, fmt.Errorf("40004:人工审查尚未通过，无法创建合并请求")
	}
	if rev.MergeStatus != "none" {
		return nil, fmt.Errorf("40005:合并请求已创建，请勿重复操作")
	}

	task := rev.CodegenTask
	repo := task.Repository

	// Resolve token: prefer user's personal token, fall back to repo's stored token
	token := s.getUserGitToken(userID)
	if token == "" {
		token, err = encrypt.AESDecrypt(s.aesKey, repo.AccessToken)
		if err != nil {
			return nil, fmt.Errorf("无可用的 Git Token，请在个人设置中配置")
		}
	}

	title := fmt.Sprintf("feat(req-%d): %s", task.RequirementID, task.Requirement.Title)
	description := s.buildMRDescription(task, rev)

	mrResult, err := gitops.CreateMergeRequest(gitops.MergeRequestInput{
		Platform:          repo.Platform,
		PlatformProjectID: repo.PlatformProjectID,
		AccessToken:       token,
		SourceBranch:      task.TargetBranch,
		TargetBranch:      task.SourceBranch,
		Title:             title,
		Description:       description,
		GitURL:            repo.GitURL,
	})
	if err != nil {
		return nil, fmt.Errorf("50101:创建合并请求失败: %s", err.Error())
	}

	s.db.Model(rev).Updates(map[string]interface{}{
		"merge_request_id":  mrResult.ID,
		"merge_request_url": mrResult.URL,
		"merge_status":      "created",
	})

	// Update requirement status
	s.db.Model(&model.Requirement{}).Where("id = ?", task.RequirementID).Update("status", "merged")

	return s.GetReviewByID(reviewID)
}

func (s *ReviewService) GetMergeRequestStatus(reviewID uint, userID uint) (map[string]interface{}, error) {
	rev, err := s.GetReviewByID(reviewID)
	if err != nil {
		return nil, err
	}
	if rev.MergeStatus == "none" {
		return map[string]interface{}{
			"merge_status":      "none",
			"merge_request_url": nil,
		}, nil
	}

	// Optionally refresh from platform
	if rev.MergeStatus == "created" && rev.CodegenTask != nil && rev.CodegenTask.Repository != nil {
		repo := rev.CodegenTask.Repository
		token := s.getUserGitToken(userID)
		if token == "" {
			token, _ = encrypt.AESDecrypt(s.aesKey, repo.AccessToken)
		}
		newStatus, _ := gitops.GetMergeRequestStatus(repo.Platform, repo.PlatformProjectID, rev.MergeRequestID, token, repo.GitURL)
		if newStatus != "" && newStatus != rev.MergeStatus {
			s.db.Model(rev).Update("merge_status", newStatus)
			rev.MergeStatus = newStatus
		}
	}

	return map[string]interface{}{
		"merge_request_id":  rev.MergeRequestID,
		"merge_request_url": rev.MergeRequestURL,
		"merge_status":      rev.MergeStatus,
	}, nil
}

// getUserGitToken retrieves user's personal git token from UserSetting.
func (s *ReviewService) getUserGitToken(userID uint) string {
	if userID == 0 {
		return ""
	}
	var setting model.UserSetting
	if err := s.db.Where("user_id = ?", userID).First(&setting).Error; err != nil {
		return ""
	}
	return setting.GitlabToken
}

func (s *ReviewService) buildMRDescription(task *model.CodegenTask, rev *model.CodeReview) string {
	desc := "## 需求\n"
	if task.Requirement != nil {
		desc += fmt.Sprintf("%s\n\n", task.Requirement.Title)
	}
	desc += "## AI Review\n"
	if rev.AIScore != nil {
		desc += fmt.Sprintf("- 评分: %d/100\n", *rev.AIScore)
	}
	desc += fmt.Sprintf("- 状态: %s\n", rev.AIStatus)
	if rev.HumanComment != "" {
		desc += fmt.Sprintf("\n## 人工 Review\n- 意见: %s\n", rev.HumanComment)
	}
	desc += "\n---\n*由 CodeMaster 自动生成*\n"

	now := time.Now()
	_ = now
	return desc
}
