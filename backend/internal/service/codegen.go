package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/codeMaster/backend/internal/codegen"
	"github.com/codeMaster/backend/internal/gitops"
	"github.com/codeMaster/backend/internal/model"
	"github.com/codeMaster/backend/internal/notify"
	"github.com/codeMaster/backend/internal/sse"
	"github.com/codeMaster/backend/pkg/encrypt"
	"github.com/codeMaster/backend/pkg/feishu"
	"gorm.io/gorm"
)

type CodegenService struct {
	db     *gorm.DB
	pool   *codegen.Pool
	hub    *sse.Hub
	aesKey string
	maxTurns    int
	timeoutMin  int
	workDir     string
	useLocalGit bool

	notifier  notify.Notifier
	docClient     *feishu.DocClient

	mu        sync.Mutex
	executors map[uint]*codegen.Executor
}

func NewCodegenService(db *gorm.DB, pool *codegen.Pool, hub *sse.Hub, aesKey string, maxTurns, timeoutMin int, workDir string, useLocalGit bool) *CodegenService {
	return &CodegenService{
		db:          db,
		pool:        pool,
		hub:         hub,
		aesKey:      aesKey,
		maxTurns:    maxTurns,
		timeoutMin:  timeoutMin,
		workDir:     workDir,
		useLocalGit: useLocalGit,
		executors:   make(map[uint]*codegen.Executor),
	}
}

// SetNotifier sets the notifier for sending notifications after codegen events.
func (s *CodegenService) SetNotifier(n notify.Notifier) {
	s.notifier = n
}

// SetDocClient sets the Feishu doc client for fetching document content during codegen.
func (s *CodegenService) SetDocClient(dc *feishu.DocClient) {
	s.docClient = dc
}

func (s *CodegenService) TriggerGeneration(requirement *model.Requirement, repo *model.Repository, extraContext, sourceBranch string, userID uint) (*model.CodegenTask, int, error) {
	if sourceBranch == "" {
		sourceBranch = repo.DefaultBranch
	}
	targetBranch := fmt.Sprintf("code-master/req-%d", requirement.ID)

	task := &model.CodegenTask{
		RequirementID: requirement.ID,
		RepositoryID:  repo.ID,
		SourceBranch:  sourceBranch,
		TargetBranch:  targetBranch,
		ExtraContext:  extraContext,
		Status:        "pending",
	}
	if err := s.db.Create(task).Error; err != nil {
		return nil, 0, err
	}

	s.db.Model(requirement).Update("status", "generating")

	// Query user's LLM settings and git token
	var apiKey, baseURL, modelName, gitToken string
	if userID > 0 {
		var setting model.UserSetting
		if err := s.db.Where("user_id = ?", userID).First(&setting).Error; err == nil {
			apiKey = setting.APIKey
			baseURL = setting.BaseURL
			modelName = setting.Model
			gitToken = setting.GitlabToken
		}
	}

	executor := codegen.NewExecutor(codegen.ExecutorConfig{
		DB:           s.db,
		Hub:          s.hub,
		AESKey:       s.aesKey,
		MaxTurns:     s.maxTurns,
		TimeoutMin:   s.timeoutMin,
		WorkDir:      s.workDir,
		UseLocalGit:  s.useLocalGit,
		Task:         task,
		Requirement:  requirement,
		Repo:         repo,
		ExtraContext: extraContext,
		DocClient:    s.docClient,
		APIKey:       apiKey,
		BaseURL:      baseURL,
		ModelName:    modelName,
		GitToken:     gitToken,
	})

	s.mu.Lock()
	s.executors[task.ID] = executor
	s.mu.Unlock()

	queuePos := s.pool.Submit(func() {
		defer func() {
			s.mu.Lock()
			delete(s.executors, task.ID)
			s.mu.Unlock()
		}()
		err := executor.Run(context.Background())

		// Send notifications
		if s.notifier != nil {
			var req model.Requirement
			if dbErr := s.db.Preload("Creator").Preload("Assignee").Preload("Project").First(&req, requirement.ID).Error; dbErr == nil {
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

				var latestTask model.CodegenTask
				if s.db.First(&latestTask, task.ID).Error == nil {
					if err == nil && latestTask.Status == "completed" {
						filesChanged, additions, deletions := 0, 0, 0
						if latestTask.DiffStat.Data != nil {
							filesChanged = latestTask.DiffStat.Data.FilesChanged
							additions = latestTask.DiffStat.Data.Additions
							deletions = latestTask.DiffStat.Data.Deletions
						}
						go s.notifier.NotifyCodegenCompleted(context.Background(), notify.CodegenCompletedEvent{
							RequirementID: req.ID,
							Title:         req.Title,
							ProjectName:   projectName,
							TaskID:        task.ID,
							CreatorOpenID: creatorOpenID,
							AssigneeOpenID: assigneeOpenID,
							FilesChanged:  filesChanged,
							Additions:     additions,
							Deletions:     deletions,
						})
					} else if latestTask.Status == "failed" {
						go s.notifier.NotifyCodegenFailed(context.Background(), notify.CodegenFailedEvent{
							RequirementID:  req.ID,
							Title:          req.Title,
							ProjectName:    projectName,
							TaskID:         task.ID,
							CreatorOpenID:  creatorOpenID,
							AssigneeOpenID: assigneeOpenID,
							ErrorMessage:   latestTask.ErrorMessage,
						})
					}
				}
			}
		}

	})

	return task, queuePos, nil
}

func (s *CodegenService) ManualSubmit(requirement *model.Requirement, repo *model.Repository, sourceBranch, commitMessage, commitURL string, userID uint) (*model.CodegenTask, error) {
	if sourceBranch == "" {
		sourceBranch = repo.DefaultBranch
	}
	targetBranch := fmt.Sprintf("code-master/req-%d", requirement.ID)

	// Extract commit SHA from commit URL
	commitSHA := extractCommitSHA(commitURL)

	now := time.Now()
	task := &model.CodegenTask{
		RequirementID: requirement.ID,
		RepositoryID:  repo.ID,
		SourceBranch:  sourceBranch,
		TargetBranch:  targetBranch,
		Status:        "completed",
		Prompt:        "手动提交",
		ExtraContext:  commitMessage,
		CommitSHA:     commitSHA,
		StartedAt:     &now,
		CompletedAt:   &now,
	}
	if err := s.db.Create(task).Error; err != nil {
		return nil, err
	}

	// Clone repo and compute diff in background-like fashion (synchronous but lightweight)
	// Get user's personal git token for diff computation
	var gitToken string
	if userID > 0 {
		var setting model.UserSetting
		if err := s.db.Where("user_id = ?", userID).First(&setting).Error; err == nil {
			gitToken = setting.GitlabToken
		}
	}
	go s.computeManualDiff(task, repo, sourceBranch, targetBranch, gitToken)

	s.db.Model(requirement).Update("status", "generated")

	return task, nil
}

// extractCommitSHA extracts the commit SHA from a commit URL.
// Supports GitHub (…/commit/{sha}) and GitLab (…/-/commit/{sha}).
func extractCommitSHA(commitURL string) string {
	commitURL = strings.TrimSpace(commitURL)
	if commitURL == "" {
		return ""
	}
	idx := strings.LastIndex(commitURL, "/commit/")
	if idx == -1 {
		return commitURL // not a recognized URL pattern, store as-is
	}
	sha := strings.TrimRight(commitURL[idx+len("/commit/"):], "/")
	return sha
}

func (s *CodegenService) computeManualDiff(task *model.CodegenTask, repo *model.Repository, sourceBranch, targetBranch, gitToken string) {
	ctx := context.Background()
	workDir := filepath.Join(s.workDir, "manual", strconv.FormatUint(uint64(task.ID), 10))
	defer os.RemoveAll(workDir)

	// Resolve token: prefer user's personal token, fall back to repo's encrypted token
	token := gitToken
	if token == "" {
		var err error
		token, err = encrypt.AESDecrypt(s.aesKey, repo.AccessToken)
		if err != nil {
			return
		}
	}

	// Clone from source branch
	if err := gitops.Clone(ctx, repo.GitURL, token, sourceBranch, workDir); err != nil {
		return
	}

	// Fetch the target branch
	if err := gitops.FetchAndCheckout(ctx, workDir, repo.GitURL, token, targetBranch); err != nil {
		// Target branch doesn't exist, no diff to compute
		return
	}

	// Compute diff between source and target
	diffStat, err := gitops.GetDiffStat(ctx, workDir, sourceBranch, targetBranch)
	if err != nil {
		return
	}

	diffFiles, err := gitops.GetDiffFiles(ctx, workDir, sourceBranch, targetBranch)
	if err == nil {
		for i := range diffFiles {
			if content, err := gitops.GetDiffContent(ctx, workDir, sourceBranch, targetBranch, diffFiles[i].Path); err == nil {
				diffFiles[i].Diff = content
			}
		}
		diffStat.Files = diffFiles
	}

	s.db.Model(task).Update("diff_stat", model.JSONDiffStat{Data: diffStat})
}

func (s *CodegenService) GetTask(id uint) (*model.CodegenTask, error) {
	var task model.CodegenTask
	if err := s.db.Preload("Requirement").Preload("Repository").First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *CodegenService) ListTasksByRequirement(requirementID uint, page, pageSize int) ([]model.CodegenTask, int64, error) {
	query := s.db.Model(&model.CodegenTask{}).Where("requirement_id = ?", requirementID)

	var total int64
	query.Count(&total)

	var tasks []model.CodegenTask
	if err := query.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

func (s *CodegenService) CancelTask(taskID uint) error {
	s.mu.Lock()
	executor, ok := s.executors[taskID]
	s.mu.Unlock()

	if !ok {
		var task model.CodegenTask
		if err := s.db.First(&task, taskID).Error; err != nil {
			return err
		}
		if task.Status == "pending" {
			s.db.Model(&task).Update("status", "cancelled")
			return nil
		}
		return fmt.Errorf("40003:任务已完成，无法取消")
	}
	return executor.Cancel()
}

func (s *CodegenService) GetHub() *sse.Hub {
	return s.hub
}
