package handler

import (
	"github.com/codeMaster/backend/internal/middleware"
	"github.com/codeMaster/backend/internal/model"
	"github.com/codeMaster/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type RepositoryHandler struct {
	repoService    *service.RepositoryService
	projectService *service.ProjectService
}

func NewRepositoryHandler(repoService *service.RepositoryService, projectService *service.ProjectService) *RepositoryHandler {
	return &RepositoryHandler{repoService: repoService, projectService: projectService}
}

// POST /projects/:id/repos
func (h *RepositoryHandler) Create(c *gin.Context) {
	projectID := parseID(c.Param("id"))

	var req struct {
		Name              string `json:"name" binding:"required,max=128"`
		GitURL            string `json:"git_url" binding:"required"`
		Platform          string `json:"platform" binding:"required,oneof=gitlab github"`
		PlatformProjectID string `json:"platform_project_id"`
		DefaultBranch     string `json:"default_branch"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, 40001, "参数校验失败: "+err.Error())
		return
	}

	if req.Platform == "gitlab" && req.PlatformProjectID == "" {
		BadRequest(c, 40001, "GitLab 平台需要提供 platform_project_id")
		return
	}

	defaultBranch := req.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "develop"
	}

	repo := &model.Repository{
		ProjectID:         projectID,
		Name:              req.Name,
		GitURL:            req.GitURL,
		Platform:          req.Platform,
		PlatformProjectID: req.PlatformProjectID,
		DefaultBranch:     defaultBranch,
	}

	if err := h.repoService.Create(repo, ""); err != nil {
		code, msg := parseErrorCode(err)
		BadRequest(c, code, msg)
		return
	}

	Success(c, gin.H{
		"id":              repo.ID,
		"name":            repo.Name,
		"git_url":         repo.GitURL,
		"platform":        repo.Platform,
		"default_branch":  repo.DefaultBranch,
		"analysis_status": repo.AnalysisStatus,
		"created_at":      repo.CreatedAt,
	})
}

// GET /projects/:id/repos
func (h *RepositoryHandler) List(c *gin.Context) {
	projectID := parseID(c.Param("id"))
	page, pageSize := parsePage(c)
	analysisStatus := c.Query("analysis_status")

	repos, total, err := h.repoService.List(projectID, analysisStatus, page, pageSize)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	list := make([]gin.H, 0, len(repos))
	for _, r := range repos {
		item := gin.H{
			"id":              r.ID,
			"name":            r.Name,
			"git_url":         r.GitURL,
			"platform":        r.Platform,
			"default_branch":  r.DefaultBranch,
			"analysis_status": r.AnalysisStatus,
			"analysis_result": r.AnalysisResult.Data,
			"analyzed_at":     r.AnalyzedAt,
			"created_at":      r.CreatedAt,
		}
		if r.AnalysisError != "" {
			item["analysis_error"] = r.AnalysisError
		}
		list = append(list, item)
	}
	SuccessPaged(c, list, total, page, pageSize)
}

// GET /repos/:id
func (h *RepositoryHandler) GetDetail(c *gin.Context) {
	id := parseID(c.Param("id"))
	repo, err := h.repoService.GetByID(id)
	if err != nil {
		NotFound(c, 40403, "仓库不存在")
		return
	}

	data := gin.H{
		"id":                  repo.ID,
		"name":                repo.Name,
		"git_url":             repo.GitURL,
		"platform":            repo.Platform,
		"platform_project_id": repo.PlatformProjectID,
		"default_branch":      repo.DefaultBranch,
		"analysis_status":     repo.AnalysisStatus,
		"analysis_result":     repo.AnalysisResult.Data,
		"analyzed_at":         repo.AnalyzedAt,
		"created_at":          repo.CreatedAt,
	}
	if repo.Project != nil {
		data["project"] = gin.H{"id": repo.Project.ID, "name": repo.Project.Name}
	}
	Success(c, data)
}

// PUT /repos/:id
func (h *RepositoryHandler) Update(c *gin.Context) {
	id := parseID(c.Param("id"))

	var req struct {
		Name          *string `json:"name"`
		DefaultBranch *string `json:"default_branch"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, 40001, "参数校验失败: "+err.Error())
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.DefaultBranch != nil {
		updates["default_branch"] = *req.DefaultBranch
	}

	repo, err := h.repoService.Update(id, updates)
	if err != nil {
		code, msg := parseErrorCode(err)
		BadRequest(c, code, msg)
		return
	}

	Success(c, gin.H{
		"id":             repo.ID,
		"name":           repo.Name,
		"default_branch": repo.DefaultBranch,
		"updated_at":     repo.UpdatedAt,
	})
}

// DELETE /repos/:id
func (h *RepositoryHandler) Delete(c *gin.Context) {
	id := parseID(c.Param("id"))
	userID := middleware.GetCurrentUserID(c)

	repo, err := h.repoService.GetByID(id)
	if err != nil {
		NotFound(c, 40403, "仓库不存在")
		return
	}

	project, _ := h.projectService.GetByID(repo.ProjectID)
	if project != nil && !middleware.GetCurrentUserIsAdmin(c) && project.OwnerID != userID {
		Forbidden(c, 40303, "非项目所有者，无权解除仓库关联")
		return
	}

	if err := h.repoService.Delete(id); err != nil {
		code, msg := parseErrorCode(err)
		BadRequest(c, code, msg)
		return
	}

	Success(c, gin.H{"message": "仓库关联已解除"})
}

// POST /repos/:id/test-connection
func (h *RepositoryHandler) TestConnection(c *gin.Context) {
	id := parseID(c.Param("id"))
	userID := middleware.GetCurrentUserID(c)
	connected, branches, canPush, err := h.repoService.TestConnection(id, userID)
	if err != nil {
		Success(c, gin.H{
			"connected": false,
			"error":     err.Error(),
			"permissions": gin.H{
				"read": false,
				"push": false,
			},
		})
		return
	}
	Success(c, gin.H{
		"connected": connected,
		"branches":  branches,
		"permissions": gin.H{
			"read": true,
			"push": canPush,
		},
	})
}

// POST /repos/:id/analyze
func (h *RepositoryHandler) Analyze(c *gin.Context) {
	id := parseID(c.Param("id"))
	userID := middleware.GetCurrentUserID(c)
	if err := h.repoService.TriggerAnalysis(id, userID); err != nil {
		code, msg := parseErrorCode(err)
		BadRequest(c, code, msg)
		return
	}
	Success(c, gin.H{
		"id":              id,
		"analysis_status": "running",
		"message":         "分析任务已启动",
	})
}

// GET /repos/:id/analysis
func (h *RepositoryHandler) GetAnalysis(c *gin.Context) {
	id := parseID(c.Param("id"))
	repo, err := h.repoService.GetByID(id)
	if err != nil {
		NotFound(c, 40403, "仓库不存在")
		return
	}
	Success(c, gin.H{
		"analysis_status": repo.AnalysisStatus,
		"analysis_error":  repo.AnalysisError,
		"analyzed_at":     repo.AnalyzedAt,
		"result":          repo.AnalysisResult.Data,
	})
}
