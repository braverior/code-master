package handler

import (
	"time"

	"github.com/codeMaster/backend/internal/middleware"
	"github.com/codeMaster/backend/internal/model"
	"github.com/codeMaster/backend/internal/notify"
	"github.com/codeMaster/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type RequirementHandler struct {
	reqService     *service.RequirementService
	projectService *service.ProjectService
	notifier       notify.Notifier
}

func NewRequirementHandler(reqService *service.RequirementService, projectService *service.ProjectService, notifier notify.Notifier) *RequirementHandler {
	return &RequirementHandler{reqService: reqService, projectService: projectService, notifier: notifier}
}

// POST /projects/:id/requirements
func (h *RequirementHandler) Create(c *gin.Context) {
	projectID := parseID(c.Param("id"))
	userID := middleware.GetCurrentUserID(c)

	var req struct {
		Title        string         `json:"title" binding:"required,max=256"`
		Description  string         `json:"description" binding:"required"`
		DocLinks     model.DocLinks `json:"doc_links"`
		Priority     string         `json:"priority"`
		Deadline     *time.Time     `json:"deadline"`
		AssigneeID   *uint          `json:"assignee_id"`
		RepositoryID *uint          `json:"repository_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, 40001, "参数校验失败: "+err.Error())
		return
	}

	priority := req.Priority
	if priority == "" {
		priority = "p1"
	}

	if req.AssigneeID != nil {
		if err := h.reqService.ValidateAssignee(projectID, *req.AssigneeID); err != nil {
			code, msg := parseErrorCode(err)
			BadRequest(c, code, msg)
			return
		}
	}
	if req.RepositoryID != nil {
		if err := h.reqService.ValidateRepository(projectID, *req.RepositoryID); err != nil {
			code, msg := parseErrorCode(err)
			BadRequest(c, code, msg)
			return
		}
	}

	requirement := &model.Requirement{
		ProjectID:    projectID,
		Title:        req.Title,
		Description:  req.Description,
		DocLinks:     req.DocLinks,
		Priority:     priority,
		Deadline:     req.Deadline,
		Status:       "draft",
		CreatorID:    userID,
		AssigneeID:   req.AssigneeID,
		RepositoryID: req.RepositoryID,
	}

	if err := h.reqService.Create(requirement); err != nil {
		InternalError(c, err.Error())
		return
	}

	// Reload with relations
	requirement, _ = h.reqService.GetByID(requirement.ID)

	// Notify assignee about new requirement
	if h.notifier != nil && requirement.Assignee != nil && requirement.Assignee.FeishuUID != "" {
		creatorName := ""
		if requirement.Creator != nil {
			creatorName = requirement.Creator.Name
		}
		projectName := ""
		if requirement.Project != nil {
			projectName = requirement.Project.Name
		}
		go h.notifier.NotifyRequirementCreated(c.Request.Context(), notify.RequirementCreatedEvent{
			RequirementID:  requirement.ID,
			Title:          requirement.Title,
			ProjectName:    projectName,
			CreatorName:    creatorName,
			AssigneeOpenID: requirement.Assignee.FeishuUID,
			Priority:       requirement.Priority,
		})
	}

	data := gin.H{
		"id":          requirement.ID,
		"title":       requirement.Title,
		"description": requirement.Description,
		"doc_links":   requirement.DocLinks,
		"priority":    requirement.Priority,
		"deadline":    requirement.Deadline,
		"status":      requirement.Status,
		"created_at":  requirement.CreatedAt,
	}
	if requirement.Creator != nil {
		data["creator"] = requirement.Creator.Brief()
	}
	if requirement.Assignee != nil {
		data["assignee"] = requirement.Assignee.Brief()
	}
	if requirement.Repository != nil {
		data["repository"] = gin.H{"id": requirement.Repository.ID, "name": requirement.Repository.Name}
	}

	Success(c, data)
}

// GET /projects/:id/requirements
func (h *RequirementHandler) List(c *gin.Context) {
	projectID := parseID(c.Param("id"))
	page, pageSize := parsePage(c)
	status := c.Query("status")
	priority := c.Query("priority")
	keyword := c.Query("keyword")
	sortBy := c.DefaultQuery("sort_by", "created_at")
	order := c.DefaultQuery("order", "desc")

	var assigneeID, creatorID *uint
	if s := c.Query("assignee_id"); s != "" {
		v := parseID(s)
		assigneeID = &v
	}
	if s := c.Query("creator_id"); s != "" {
		v := parseID(s)
		creatorID = &v
	}

	reqs, total, err := h.reqService.List(projectID, status, priority, keyword, assigneeID, creatorID, page, pageSize, sortBy, order)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	list := make([]gin.H, 0, len(reqs))
	for _, r := range reqs {
		item := gin.H{
			"id":         r.ID,
			"title":      r.Title,
			"priority":   r.Priority,
			"status":     r.Status,
			"doc_links":  r.DocLinks,
			"deadline":   r.Deadline,
			"created_at": r.CreatedAt,
			"updated_at": r.UpdatedAt,
		}
		if r.Creator != nil {
			item["creator"] = r.Creator.Brief()
		}
		if r.Assignee != nil {
			item["assignee"] = r.Assignee.Brief()
		}
		if r.Repository != nil {
			item["repository"] = gin.H{"id": r.Repository.ID, "name": r.Repository.Name}
		}

		latestTask := h.reqService.GetLatestCodegenTask(r.ID)
		if latestTask != nil {
			item["latest_codegen"] = gin.H{
				"id":         latestTask.ID,
				"status":     latestTask.Status,
				"created_at": latestTask.CreatedAt,
			}
		}
		latestReview := h.reqService.GetLatestReview(r.ID)
		if latestReview != nil {
			item["latest_review"] = gin.H{
				"id":           latestReview.ID,
				"ai_score":     latestReview.AIScore,
				"human_status": latestReview.HumanStatus,
			}
		}
		list = append(list, item)
	}

	SuccessPaged(c, list, total, page, pageSize)
}

// GET /requirements
func (h *RequirementHandler) ListAll(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	page, pageSize := parsePage(c)
	scope := c.DefaultQuery("scope", "all")
	status := c.Query("status")
	keyword := c.Query("keyword")

	reqs, total, err := h.reqService.ListAccessible(userID, scope, status, keyword, page, pageSize)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	list := make([]gin.H, 0, len(reqs))
	for _, r := range reqs {
		item := gin.H{
			"id":         r.ID,
			"title":      r.Title,
			"priority":   r.Priority,
			"status":     r.Status,
			"doc_links":  r.DocLinks,
			"deadline":   r.Deadline,
			"created_at": r.CreatedAt,
			"updated_at": r.UpdatedAt,
		}
		if r.Project != nil {
			item["project"] = gin.H{"id": r.Project.ID, "name": r.Project.Name}
		}
		if r.Creator != nil {
			item["creator"] = r.Creator.Brief()
		}
		if r.Assignee != nil {
			item["assignee"] = r.Assignee.Brief()
		}
		if r.Repository != nil {
			item["repository"] = gin.H{"id": r.Repository.ID, "name": r.Repository.Name}
		}
		list = append(list, item)
	}

	SuccessPaged(c, list, total, page, pageSize)
}

// GET /requirements/:id
func (h *RequirementHandler) GetDetail(c *gin.Context) {
	id := parseID(c.Param("id"))
	req, err := h.reqService.GetByID(id)
	if err != nil {
		NotFound(c, 40404, "需求不存在")
		return
	}

	data := gin.H{
		"id":          req.ID,
		"title":       req.Title,
		"description": req.Description,
		"doc_links":   req.DocLinks,
		"priority":    req.Priority,
		"deadline":    req.Deadline,
		"status":      req.Status,
		"created_at":  req.CreatedAt,
		"updated_at":  req.UpdatedAt,
	}
	if req.Project != nil {
		data["project"] = gin.H{"id": req.Project.ID, "name": req.Project.Name}
	}
	if req.Creator != nil {
		data["creator"] = req.Creator.Brief()
	}
	if req.Assignee != nil {
		data["assignee"] = req.Assignee.Brief()
	}
	if req.Repository != nil {
		data["repository"] = gin.H{"id": req.Repository.ID, "name": req.Repository.Name, "platform": req.Repository.Platform, "git_url": req.Repository.GitURL}
	}

	// Codegen tasks
	var tasks []model.CodegenTask
	h.reqService.DB().Where("requirement_id = ?", id).Order("created_at desc").Find(&tasks)
	taskList := make([]gin.H, 0, len(tasks))
	for _, t := range tasks {
		item := gin.H{
			"id":            t.ID,
			"status":        t.Status,
			"source_branch": t.SourceBranch,
			"target_branch": t.TargetBranch,
			"created_at":    t.CreatedAt,
		}
		if t.DiffStat.Data != nil {
			item["diff_stat"] = gin.H{
				"files_changed": t.DiffStat.Data.FilesChanged,
				"additions":     t.DiffStat.Data.Additions,
				"deletions":     t.DiffStat.Data.Deletions,
			}
		}
		if t.ErrorMessage != "" {
			item["error_message"] = t.ErrorMessage
		}
		if t.StartedAt != nil {
			item["started_at"] = t.StartedAt
		}
		if t.CompletedAt != nil {
			item["completed_at"] = t.CompletedAt
		}
		taskList = append(taskList, item)
	}
	data["codegen_tasks"] = taskList

	latestReview := h.reqService.GetLatestReview(id)
	if latestReview != nil {
		data["latest_review"] = gin.H{
			"id":           latestReview.ID,
			"ai_score":     latestReview.AIScore,
			"ai_status":    latestReview.AIStatus,
			"human_status": latestReview.HumanStatus,
		}
	}

	Success(c, data)
}

// PUT /requirements/:id
func (h *RequirementHandler) Update(c *gin.Context) {
	id := parseID(c.Param("id"))
	userID := middleware.GetCurrentUserID(c)

	req, err := h.reqService.GetByID(id)
	if err != nil {
		NotFound(c, 40404, "需求不存在")
		return
	}

	if !middleware.GetCurrentUserIsAdmin(c) && req.CreatorID != userID {
		Forbidden(c, 40303, "非需求创建者，无权编辑")
		return
	}

	if req.Status != "draft" && req.Status != "rejected" {
		BadRequest(c, 40003, "需求当前状态为 "+req.Status+"，不可编辑")
		return
	}

	var body struct {
		Title        *string         `json:"title"`
		Description  *string         `json:"description"`
		DocLinks     *model.DocLinks `json:"doc_links"`
		Priority     *string         `json:"priority"`
		Deadline     *time.Time      `json:"deadline"`
		AssigneeID   *uint           `json:"assignee_id"`
		RepositoryID *uint           `json:"repository_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, 40001, "参数校验失败: "+err.Error())
		return
	}

	updates := make(map[string]interface{})
	if body.Title != nil {
		updates["title"] = *body.Title
	}
	if body.Description != nil {
		updates["description"] = *body.Description
	}
	if body.DocLinks != nil {
		updates["doc_links"] = *body.DocLinks
	}
	if body.Priority != nil {
		updates["priority"] = *body.Priority
	}
	if body.Deadline != nil {
		updates["deadline"] = *body.Deadline
	}
	if body.AssigneeID != nil {
		updates["assignee_id"] = *body.AssigneeID
	}
	if body.RepositoryID != nil {
		updates["repository_id"] = *body.RepositoryID
	}

	// Reset status to draft if was rejected
	if req.Status == "rejected" {
		updates["status"] = "draft"
	}

	updated, err := h.reqService.Update(id, updates)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	// Notify new assignee if assignee changed
	if h.notifier != nil && body.AssigneeID != nil &&
		(req.AssigneeID == nil || *body.AssigneeID != *req.AssigneeID) {
		// Reload with relations to get new assignee info
		reloaded, reloadErr := h.reqService.GetByID(id)
		if reloadErr == nil && reloaded.Assignee != nil && reloaded.Assignee.FeishuUID != "" {
			assignerName := ""
			if currentUser := middleware.GetCurrentUser(c); currentUser != nil {
				assignerName = currentUser.Name
			}
			projectName := ""
			if reloaded.Project != nil {
				projectName = reloaded.Project.Name
			}
			go h.notifier.NotifyRequirementAssigned(c.Request.Context(), notify.RequirementAssignedEvent{
				RequirementID:  reloaded.ID,
				Title:          reloaded.Title,
				ProjectName:    projectName,
				AssignerName:   assignerName,
				AssigneeOpenID: reloaded.Assignee.FeishuUID,
				Priority:       reloaded.Priority,
			})
		}
	}

	Success(c, gin.H{
		"id":         updated.ID,
		"title":      updated.Title,
		"priority":   updated.Priority,
		"status":     updated.Status,
		"updated_at": updated.UpdatedAt,
	})
}

// DELETE /requirements/:id
func (h *RequirementHandler) Delete(c *gin.Context) {
	id := parseID(c.Param("id"))
	userID := middleware.GetCurrentUserID(c)

	req, err := h.reqService.GetByID(id)
	if err != nil {
		NotFound(c, 40404, "需求不存在")
		return
	}

	if !middleware.GetCurrentUserIsAdmin(c) && req.CreatorID != userID {
		Forbidden(c, 40303, "非需求创建者，无权删除")
		return
	}

	if req.Status == "generating" {
		BadRequest(c, 40003, "需求当前状态为 generating，不可删除")
		return
	}

	var body struct {
		Force bool `json:"force"`
	}
	c.ShouldBindJSON(&body)

	if req.Status != "draft" && !body.Force {
		BadRequest(c, 40003, "需求非 draft 状态，需要 force=true 确认删除")
		return
	}

	if err := h.reqService.Delete(id); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "需求已删除"})
}
