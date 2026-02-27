package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/codeMaster/backend/internal/middleware"
	"github.com/codeMaster/backend/internal/service"
	"github.com/codeMaster/backend/internal/sse"
	"github.com/gin-gonic/gin"
)

type CodegenHandler struct {
	codegenService *service.CodegenService
	reqService     *service.RequirementService
	repoService    *service.RepositoryService
	reviewService  *service.ReviewService
}

func NewCodegenHandler(
	codegenService *service.CodegenService,
	reqService *service.RequirementService,
	repoService *service.RepositoryService,
	reviewService *service.ReviewService,
) *CodegenHandler {
	return &CodegenHandler{
		codegenService: codegenService,
		reqService:     reqService,
		repoService:    repoService,
		reviewService:  reviewService,
	}
}

// POST /requirements/:id/generate
func (h *CodegenHandler) Generate(c *gin.Context) {
	reqID := parseID(c.Param("id"))

	requirement, err := h.reqService.GetByID(reqID)
	if err != nil {
		NotFound(c, 40404, "需求不存在")
		return
	}

	if requirement.Status == "generating" {
		BadRequest(c, 40003, "需求正在生成中，请等待完成")
		return
	}
	if requirement.RepositoryID == nil {
		BadRequest(c, 40004, "需求未关联代码仓库，请先关联仓库")
		return
	}
	if requirement.AssigneeID == nil {
		BadRequest(c, 40004, "需求未指派 RD，请先指派开发人员")
		return
	}
	if h.reqService.HasRunningTask(reqID) {
		BadRequest(c, 40003, "该需求已有生成任务正在运行中")
		return
	}

	repo, err := h.repoService.GetByID(*requirement.RepositoryID)
	if err != nil {
		InternalError(c, "仓库不存在")
		return
	}

	var body struct {
		ExtraContext  string `json:"extra_context"`
		SourceBranch string `json:"source_branch"`
	}
	c.ShouldBindJSON(&body)

	task, queuePos, err := h.codegenService.TriggerGeneration(requirement, repo, body.ExtraContext, body.SourceBranch, middleware.GetCurrentUserID(c))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"task_id":        task.ID,
		"status":         task.Status,
		"source_branch":  task.SourceBranch,
		"target_branch":  task.TargetBranch,
		"queue_position": queuePos,
	})
}

// GET /codegen/:id/stream
func (h *CodegenHandler) Stream(c *gin.Context) {
	taskID := parseID(c.Param("id"))

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		InternalError(c, "streaming not supported")
		return
	}

	lastEventID := sse.ParseLastEventID(c.GetHeader("Last-Event-ID"))

	hub := h.codegenService.GetHub()

	// Replay history
	history, _ := hub.ReplayFrom(int64(taskID), lastEventID)
	eventID := lastEventID
	for _, ev := range history {
		data, _ := json.Marshal(ev.Data)
		evType := ev.Type
		// Backward compatibility: old events stored as "error" must be renamed to "task_error"
		// to avoid triggering EventSource's native onerror handler on the client side.
		if evType == "error" {
			evType = "task_error"
		}
		fmt.Fprintf(c.Writer, "id: %d\nevent: %s\ndata: %s\n\n", eventID, evType, string(data))
		eventID++
		flusher.Flush()
	}

	// Check if task is done
	task, err := h.codegenService.GetTask(taskID)
	if err != nil {
		fmt.Fprintf(c.Writer, "event: task_error\ndata: {\"message\":\"任务不存在\"}\n\n")
		flusher.Flush()
		return
	}

	if task.Status == "completed" || task.Status == "failed" || task.Status == "cancelled" {
		fmt.Fprintf(c.Writer, "event: done\ndata: {\"status\":\"%s\",\"task_id\":%d}\n\n", task.Status, task.ID)
		flusher.Flush()
		return
	}

	// Subscribe for live events
	ch, unsub := hub.Subscribe(int64(taskID))
	defer unsub()

	ctx := c.Request.Context()
	heartbeat := make(chan struct{})

	// Heartbeat goroutine
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				select {
				case heartbeat <- struct{}{}:
				default:
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case ev := <-ch:
			data, _ := json.Marshal(ev.Data)
			fmt.Fprintf(c.Writer, "id: %d\nevent: %s\ndata: %s\n\n", eventID, ev.Type, string(data))
			eventID++
			flusher.Flush()
			if ev.Type == "done" {
				return
			}
		case <-heartbeat:
			fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// GET /codegen/:id
func (h *CodegenHandler) GetTask(c *gin.Context) {
	taskID := parseID(c.Param("id"))
	task, err := h.codegenService.GetTask(taskID)
	if err != nil {
		NotFound(c, 40405, "生成任务不存在")
		return
	}

	data := gin.H{
		"id":            task.ID,
		"source_branch": task.SourceBranch,
		"target_branch": task.TargetBranch,
		"status":        task.Status,
		"extra_context":  task.ExtraContext,
		"prompt":        task.Prompt,
		"diff_stat":     task.DiffStat.Data,
		"claude_cost_usd": task.ClaudeCostUSD,
		"started_at":    task.StartedAt,
		"completed_at":  task.CompletedAt,
		"created_at":    task.CreatedAt,
	}
	if task.ErrorMessage != "" {
		data["error_message"] = task.ErrorMessage
	}
	if task.CommitSHA != "" {
		data["commit_sha"] = task.CommitSHA
	}
	if task.Requirement != nil {
		data["requirement"] = gin.H{"id": task.Requirement.ID, "title": task.Requirement.Title}
	}
	if task.Repository != nil {
		data["repository"] = gin.H{"id": task.Repository.ID, "name": task.Repository.Name, "platform": task.Repository.Platform, "git_url": task.Repository.GitURL}
	}

	// Include review info
	review, _ := h.reviewService.GetReview(taskID)
	if review != nil {
		data["review"] = gin.H{
			"id":           review.ID,
			"ai_score":     review.AIScore,
			"ai_status":    review.AIStatus,
			"human_status": review.HumanStatus,
		}
	}

	Success(c, data)
}

// GET /requirements/:id/codegen-tasks
func (h *CodegenHandler) ListTasks(c *gin.Context) {
	reqID := parseID(c.Param("id"))
	page, pageSize := parsePage(c)

	tasks, total, err := h.codegenService.ListTasksByRequirement(reqID, page, pageSize)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	list := make([]gin.H, 0, len(tasks))
	for _, t := range tasks {
		item := gin.H{
			"id":            t.ID,
			"status":        t.Status,
			"target_branch": t.TargetBranch,
			"prompt":        t.Prompt,
			"claude_cost_usd": t.ClaudeCostUSD,
			"started_at":    t.StartedAt,
			"completed_at":  t.CompletedAt,
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
		list = append(list, item)
	}
	SuccessPaged(c, list, total, page, pageSize)
}

// GET /codegen/:id/diff
func (h *CodegenHandler) GetDiff(c *gin.Context) {
	taskID := parseID(c.Param("id"))
	task, err := h.codegenService.GetTask(taskID)
	if err != nil {
		NotFound(c, 40405, "生成任务不存在")
		return
	}
	if task.Status != "completed" {
		BadRequest(c, 40003, "任务尚未完成，无法获取 diff")
		return
	}

	data := gin.H{
		"target_branch": task.TargetBranch,
		"base_branch":   task.SourceBranch,
	}
	if task.DiffStat.Data != nil {
		files := make([]gin.H, 0)
		for _, f := range task.DiffStat.Data.Files {
			item := gin.H{
				"path":      f.Path,
				"status":    f.Status,
				"additions": f.Additions,
				"deletions": f.Deletions,
			}
			if f.Diff != "" {
				item["diff"] = f.Diff
			}
			files = append(files, item)
		}
		data["files"] = files
	}
	Success(c, data)
}

// GET /codegen/:id/log
func (h *CodegenHandler) GetLog(c *gin.Context) {
	taskID := parseID(c.Param("id"))
	task, err := h.codegenService.GetTask(taskID)
	if err != nil {
		NotFound(c, 40405, "生成任务不存在")
		return
	}

	offset, _ := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "500"), 10, 64)
	if limit > 1000 {
		limit = 1000
	}

	hub := h.codegenService.GetHub()
	totalEvents := hub.GetTotalEvents(int64(taskID))
	events, _ := hub.GetEventsPage(int64(taskID), offset, limit)

	eventList := make([]gin.H, 0, len(events))
	for _, ev := range events {
		eventList = append(eventList, gin.H{
			"id":   ev.ID,
			"type": ev.Type,
			"data": ev.Data,
		})
	}

	Success(c, gin.H{
		"task_id":      task.ID,
		"status":       task.Status,
		"total_events": totalEvents,
		"events":       eventList,
		"has_more":     offset+limit < totalEvents,
	})
}

// POST /requirements/:id/manual-submit
func (h *CodegenHandler) ManualSubmit(c *gin.Context) {
	reqID := parseID(c.Param("id"))

	requirement, err := h.reqService.GetByID(reqID)
	if err != nil {
		NotFound(c, 40404, "需求不存在")
		return
	}

	if requirement.RepositoryID == nil {
		BadRequest(c, 40004, "需求未关联代码仓库，请先关联仓库")
		return
	}

	repo, err := h.repoService.GetByID(*requirement.RepositoryID)
	if err != nil {
		InternalError(c, "仓库不存在")
		return
	}

	var body struct {
		SourceBranch  string `json:"source_branch"`
		CommitMessage string `json:"commit_message"`
		CommitURL     string `json:"commit_url"`
	}
	c.ShouldBindJSON(&body)

	task, err := h.codegenService.ManualSubmit(requirement, repo, body.SourceBranch, body.CommitMessage, body.CommitURL, middleware.GetCurrentUserID(c))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"task_id":       task.ID,
		"status":        task.Status,
		"source_branch": task.SourceBranch,
		"target_branch": task.TargetBranch,
	})
}

// POST /codegen/:id/cancel
func (h *CodegenHandler) Cancel(c *gin.Context) {
	taskID := parseID(c.Param("id"))

	task, err := h.codegenService.GetTask(taskID)
	if err != nil {
		NotFound(c, 40405, "生成任务不存在")
		return
	}

	if task.Status != "pending" && task.Status != "cloning" && task.Status != "running" {
		BadRequest(c, 40003, "任务已完成，无法取消")
		return
	}

	if err := h.codegenService.CancelTask(taskID); err != nil {
		code, msg := parseErrorCode(err)
		BadRequest(c, code, msg)
		return
	}

	Success(c, gin.H{
		"id":     taskID,
		"status": "cancelled",
	})
}
