package handler

import (
	"github.com/codeMaster/backend/internal/middleware"
	"github.com/codeMaster/backend/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DashboardHandler struct {
	db *gorm.DB
}

func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// GET /dashboard/stats
func (h *DashboardHandler) GetStats(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)

	var myProjects int64
	h.db.Model(&model.ProjectMember{}).Where("user_id = ?", userID).Count(&myProjects)

	var myOpenReqs int64
	h.db.Model(&model.Requirement{}).
		Where("(creator_id = ? OR assignee_id = ?) AND status NOT IN ?", userID, userID, []string{"merged"}).
		Count(&myOpenReqs)

	var pendingReviews int64
	h.db.Model(&model.CodeReview{}).
		Joins("JOIN codegen_tasks ON code_reviews.codegen_task_id = codegen_tasks.id").
		Joins("JOIN requirements ON codegen_tasks.requirement_id = requirements.id").
		Where("code_reviews.human_status = ? AND code_reviews.ai_status IN ?", "pending", []string{"passed", "warning", "failed"}).
		Where("requirements.project_id IN (SELECT project_id FROM project_members WHERE user_id = ?)", userID).
		Count(&pendingReviews)

	var codegenRunning int64
	h.db.Model(&model.CodegenTask{}).Where("status IN ?", []string{"pending", "cloning", "running"}).Count(&codegenRunning)

	// Recent activity (last 10)
	var recentTasks []model.CodegenTask
	h.db.Preload("Requirement.Project").
		Where("status IN ? AND requirement_id IN (SELECT id FROM requirements WHERE project_id IN (SELECT project_id FROM project_members WHERE user_id = ?))", []string{"completed", "failed"}, userID).
		Order("created_at desc").Limit(10).Find(&recentTasks)

	recentActivity := make([]gin.H, 0)
	for _, t := range recentTasks {
		actType := "codegen_completed"
		if t.Status == "failed" {
			actType = "codegen_failed"
		}
		item := gin.H{
			"type": actType,
			"time": t.CompletedAt,
		}
		if t.Requirement != nil {
			item["requirement"] = gin.H{"id": t.Requirement.ID, "title": t.Requirement.Title}
			if t.Requirement.Project != nil {
				item["project"] = gin.H{"id": t.Requirement.Project.ID, "name": t.Requirement.Project.Name}
			}
		}
		recentActivity = append(recentActivity, item)
	}

	Success(c, gin.H{
		"my_projects":          myProjects,
		"my_open_requirements": myOpenReqs,
		"my_pending_reviews":   pendingReviews,
		"codegen_running":      codegenRunning,
		"recent_activity":      recentActivity,
	})
}

// GET /dashboard/my-tasks
func (h *DashboardHandler) GetMyTasks(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)

	// Pending generate: requirements assigned to user in draft status
	var pendingGenerate []model.Requirement
	h.db.Preload("Project").
		Where("assignee_id = ? AND status = ?", userID, "draft").
		Order("priority asc, created_at asc").Limit(10).Find(&pendingGenerate)

	pendingList := make([]gin.H, 0)
	for _, r := range pendingGenerate {
		item := gin.H{
			"requirement_id": r.ID,
			"title":          r.Title,
			"priority":       r.Priority,
			"created_at":     r.CreatedAt,
		}
		if r.Project != nil {
			item["project"] = gin.H{"id": r.Project.ID, "name": r.Project.Name}
		}
		pendingList = append(pendingList, item)
	}

	// Running tasks
	var runningTasks []model.CodegenTask
	h.db.Preload("Requirement").
		Where("status IN ? AND requirement_id IN (SELECT id FROM requirements WHERE assignee_id = ?)", []string{"pending", "cloning", "running"}, userID).
		Order("created_at desc").Limit(10).Find(&runningTasks)

	runningList := make([]gin.H, 0)
	for _, t := range runningTasks {
		item := gin.H{
			"task_id":    t.ID,
			"status":     t.Status,
			"started_at": t.StartedAt,
		}
		if t.Requirement != nil {
			item["requirement"] = gin.H{"id": t.Requirement.ID, "title": t.Requirement.Title}
		}
		runningList = append(runningList, item)
	}

	// Pending reviews
	var pendingReviews []model.CodeReview
	h.db.Preload("CodegenTask.Requirement").
		Joins("JOIN codegen_tasks ON code_reviews.codegen_task_id = codegen_tasks.id").
		Joins("JOIN requirements ON codegen_tasks.requirement_id = requirements.id").
		Where("code_reviews.human_status = ? AND code_reviews.ai_status IN ?", "pending", []string{"passed", "warning", "failed"}).
		Where("requirements.project_id IN (SELECT project_id FROM project_members WHERE user_id = ?)", userID).
		Order("code_reviews.created_at desc").Limit(10).Find(&pendingReviews)

	reviewList := make([]gin.H, 0)
	for _, rev := range pendingReviews {
		item := gin.H{
			"review_id":  rev.ID,
			"ai_score":   rev.AIScore,
			"created_at": rev.CreatedAt,
		}
		if rev.CodegenTask != nil && rev.CodegenTask.Requirement != nil {
			item["requirement"] = gin.H{
				"id":    rev.CodegenTask.Requirement.ID,
				"title": rev.CodegenTask.Requirement.Title,
			}
		}
		reviewList = append(reviewList, item)
	}

	Success(c, gin.H{
		"pending_generate": pendingList,
		"running_tasks":    runningList,
		"pending_reviews":  reviewList,
	})
}
