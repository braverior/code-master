package handler

import (
	"github.com/codeMaster/backend/internal/middleware"
	"github.com/codeMaster/backend/internal/model"
	"github.com/codeMaster/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type ReviewHandler struct {
	reviewService *service.ReviewService
}

func NewReviewHandler(reviewService *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{reviewService: reviewService}
}

// POST /codegen/:id/review
func (h *ReviewHandler) TriggerAIReview(c *gin.Context) {
	taskID := parseID(c.Param("id"))

	var req struct {
		ReviewerIDs []uint `json:"reviewer_ids"`
	}
	_ = c.ShouldBindJSON(&req)

	rev, err := h.reviewService.TriggerAIReview(c.Request.Context(), taskID, req.ReviewerIDs, middleware.GetCurrentUserID(c))
	if err != nil {
		code, msg := parseErrorCode(err)
		BadRequest(c, code, msg)
		return
	}
	Success(c, gin.H{
		"review_id": rev.ID,
		"ai_status": rev.AIStatus,
		"message":   "AI Review 已启动",
	})
}

// GET /reviews/:id
func (h *ReviewHandler) GetReviewByID(c *gin.Context) {
	reviewID := parseID(c.Param("id"))
	rev, err := h.reviewService.GetReviewByID(reviewID)
	if err != nil {
		NotFound(c, 40406, "Review 记录不存在")
		return
	}

	data := gin.H{
		"id":              rev.ID,
		"codegen_task_id": rev.CodegenTaskID,
		"ai_score":        rev.AIScore,
		"ai_status":       rev.AIStatus,
		"human_status":    rev.HumanStatus,
		"merge_status":    rev.MergeStatus,
		"created_at":      rev.CreatedAt,
		"updated_at":      rev.UpdatedAt,
	}

	if rev.AIReviewResult.Data != nil {
		data["ai_review"] = gin.H{
			"score":      rev.AIScore,
			"summary":    rev.AIReviewResult.Data.Summary,
			"issues":     rev.AIReviewResult.Data.Issues,
			"categories": rev.AIReviewResult.Data.Categories,
		}
	}

	if rev.HumanReviewer != nil {
		data["human_reviewer"] = rev.HumanReviewer.Brief()
	}
	if rev.HumanComment != "" {
		data["human_comment"] = rev.HumanComment
	}
	if rev.MergeRequestURL != "" {
		data["merge_request_url"] = rev.MergeRequestURL
	}

	if rev.CodegenTask != nil {
		if rev.CodegenTask.DiffStat.Data != nil {
			data["diff_stat"] = gin.H{
				"files_changed": rev.CodegenTask.DiffStat.Data.FilesChanged,
				"additions":     rev.CodegenTask.DiffStat.Data.Additions,
				"deletions":     rev.CodegenTask.DiffStat.Data.Deletions,
			}
		}
		data["source_branch"] = rev.CodegenTask.SourceBranch
		data["target_branch"] = rev.CodegenTask.TargetBranch
		if rev.CodegenTask.Requirement != nil {
			data["requirement"] = gin.H{
				"id":    rev.CodegenTask.Requirement.ID,
				"title": rev.CodegenTask.Requirement.Title,
			}
		}
		if rev.CodegenTask.Repository != nil {
			data["repository"] = gin.H{
				"id":   rev.CodegenTask.Repository.ID,
				"name": rev.CodegenTask.Repository.Name,
			}
			data["git_url"] = rev.CodegenTask.Repository.GitURL
			data["platform"] = rev.CodegenTask.Repository.Platform
		}
	}

	if len(rev.Reviewers) > 0 {
		reviewers := make([]gin.H, 0, len(rev.Reviewers))
		for _, u := range rev.Reviewers {
			reviewers = append(reviewers, gin.H{"id": u.ID, "name": u.Name, "avatar": u.Avatar})
		}
		data["reviewers"] = reviewers
	}

	Success(c, data)
}

// GET /codegen/:id/review
func (h *ReviewHandler) GetReview(c *gin.Context) {
	taskID := parseID(c.Param("id"))
	rev, err := h.reviewService.GetReview(taskID)
	if err != nil {
		NotFound(c, 40406, "Review 记录不存在")
		return
	}

	data := gin.H{
		"id":              rev.ID,
		"codegen_task_id": rev.CodegenTaskID,
		"ai_score":        rev.AIScore,
		"ai_status":       rev.AIStatus,
		"human_status":    rev.HumanStatus,
		"merge_status":    rev.MergeStatus,
		"created_at":      rev.CreatedAt,
		"updated_at":      rev.UpdatedAt,
	}

	if rev.AIReviewResult.Data != nil {
		data["ai_review"] = gin.H{
			"score":      rev.AIScore,
			"summary":    rev.AIReviewResult.Data.Summary,
			"issues":     rev.AIReviewResult.Data.Issues,
			"categories": rev.AIReviewResult.Data.Categories,
		}
	}

	if rev.HumanReviewer != nil {
		data["human_reviewer"] = rev.HumanReviewer.Brief()
	}
	if rev.HumanComment != "" {
		data["human_comment"] = rev.HumanComment
	}
	if rev.MergeRequestURL != "" {
		data["merge_request_url"] = rev.MergeRequestURL
	}

	Success(c, data)
}

// GET /reviews/pending
func (h *ReviewHandler) ListPending(c *gin.Context) {
	page, pageSize := parsePage(c)
	userID := middleware.GetCurrentUserID(c)

	var projectID *uint
	if s := c.Query("project_id"); s != "" {
		v := parseID(s)
		projectID = &v
	}

	reviews, total, err := h.reviewService.ListPendingReviews(userID, projectID, page, pageSize)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	list := h.buildReviewList(reviews)
	SuccessPaged(c, list, total, page, pageSize)
}

// GET /reviews
func (h *ReviewHandler) ListReviews(c *gin.Context) {
	page, pageSize := parsePage(c)
	userID := middleware.GetCurrentUserID(c)
	humanStatus := c.Query("human_status")

	var projectID *uint
	if s := c.Query("project_id"); s != "" {
		v := parseID(s)
		projectID = &v
	}

	reviews, total, err := h.reviewService.ListReviews(userID, humanStatus, projectID, page, pageSize)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	list := h.buildReviewList(reviews)
	SuccessPaged(c, list, total, page, pageSize)
}

func (h *ReviewHandler) buildReviewList(reviews []model.CodeReview) []gin.H {
	list := make([]gin.H, 0, len(reviews))
	for _, rev := range reviews {
		item := gin.H{
			"id":              rev.ID,
			"codegen_task_id": rev.CodegenTaskID,
			"ai_score":        rev.AIScore,
			"ai_status":       rev.AIStatus,
			"human_status":    rev.HumanStatus,
			"merge_status":    rev.MergeStatus,
			"created_at":      rev.CreatedAt,
		}
		if rev.AIReviewResult.Data != nil {
			item["ai_summary"] = rev.AIReviewResult.Data.Summary
		}
		if rev.HumanReviewer != nil {
			item["human_reviewer"] = rev.HumanReviewer.Brief()
		}
		if rev.CodegenTask != nil {
			item["target_branch"] = rev.CodegenTask.TargetBranch
			if rev.CodegenTask.DiffStat.Data != nil {
				item["diff_stat"] = gin.H{
					"files_changed": rev.CodegenTask.DiffStat.Data.FilesChanged,
					"additions":     rev.CodegenTask.DiffStat.Data.Additions,
					"deletions":     rev.CodegenTask.DiffStat.Data.Deletions,
				}
			}
			if rev.CodegenTask.Requirement != nil {
				item["requirement"] = gin.H{
					"id":    rev.CodegenTask.Requirement.ID,
					"title": rev.CodegenTask.Requirement.Title,
				}
				if rev.CodegenTask.Requirement.Creator != nil {
					item["creator"] = rev.CodegenTask.Requirement.Creator.Brief()
				}
				if rev.CodegenTask.Requirement.Project != nil {
					item["project"] = gin.H{
						"id":   rev.CodegenTask.Requirement.Project.ID,
						"name": rev.CodegenTask.Requirement.Project.Name,
					}
				}
			}
			if rev.CodegenTask.Repository != nil {
				item["repository"] = gin.H{
					"id":   rev.CodegenTask.Repository.ID,
					"name": rev.CodegenTask.Repository.Name,
				}
			}
		}
		list = append(list, item)
	}
	return list
}

// PUT /reviews/:id/human
func (h *ReviewHandler) SubmitHumanReview(c *gin.Context) {
	reviewID := parseID(c.Param("id"))
	userID := middleware.GetCurrentUserID(c)

	var req struct {
		Comment string `json:"comment"`
		Status  string `json:"status" binding:"required,oneof=approved rejected needs_revision"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, 40001, "参数校验失败: "+err.Error())
		return
	}

	if (req.Status == "rejected" || req.Status == "needs_revision") && req.Comment == "" {
		BadRequest(c, 40001, "拒绝时必须填写审查意见")
		return
	}

	rev, err := h.reviewService.SubmitHumanReview(reviewID, userID, req.Comment, req.Status)
	if err != nil {
		code, msg := parseErrorCode(err)
		BadRequest(c, code, msg)
		return
	}

	data := gin.H{
		"id":           rev.ID,
		"human_comment": rev.HumanComment,
		"human_status": rev.HumanStatus,
		"updated_at":   rev.UpdatedAt,
	}
	if rev.HumanReviewer != nil {
		data["human_reviewer"] = rev.HumanReviewer.Brief()
	}
	Success(c, data)
}

// POST /reviews/:id/merge-request
func (h *ReviewHandler) CreateMergeRequest(c *gin.Context) {
	reviewID := parseID(c.Param("id"))
	userID := middleware.GetCurrentUserID(c)
	rev, err := h.reviewService.CreateMergeRequest(reviewID, userID)
	if err != nil {
		code, msg := parseErrorCode(err)
		BadRequest(c, code, msg)
		return
	}
	Success(c, gin.H{
		"review_id":         rev.ID,
		"merge_request_id":  rev.MergeRequestID,
		"merge_request_url": rev.MergeRequestURL,
		"merge_status":      rev.MergeStatus,
	})
}

// GET /reviews/:id/merge-request
func (h *ReviewHandler) GetMergeRequestStatus(c *gin.Context) {
	reviewID := parseID(c.Param("id"))
	userID := middleware.GetCurrentUserID(c)
	data, err := h.reviewService.GetMergeRequestStatus(reviewID, userID)
	if err != nil {
		NotFound(c, 40406, "Review 记录不存在")
		return
	}
	Success(c, data)
}
