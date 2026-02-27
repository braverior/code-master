package handler

import (
	"github.com/codeMaster/backend/internal/middleware"
	"github.com/codeMaster/backend/internal/model"
	"github.com/codeMaster/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type ProjectHandler struct {
	projectService *service.ProjectService
}

func NewProjectHandler(projectService *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projectService: projectService}
}

// POST /projects
func (h *ProjectHandler) Create(c *gin.Context) {
	var req struct {
		Name        string         `json:"name" binding:"required,max=128"`
		Description string         `json:"description" binding:"max=5000"`
		DocLinks    model.DocLinks `json:"doc_links"`
		MemberIDs   []uint         `json:"member_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, 40001, "参数校验失败: "+err.Error())
		return
	}

	userID := middleware.GetCurrentUserID(c)
	project, err := h.projectService.Create(req.Name, req.Description, userID, req.DocLinks, req.MemberIDs)
	if err != nil {
		code, msg := parseErrorCode(err)
		BadRequest(c, code, msg)
		return
	}

	data := gin.H{
		"id":          project.ID,
		"name":        project.Name,
		"description": project.Description,
		"doc_links":   project.DocLinks,
		"status":      project.Status,
		"created_at":  project.CreatedAt,
	}
	if project.Owner != nil {
		data["owner"] = project.Owner.Brief()
	}

	Success(c, data)
}

// GET /projects
func (h *ProjectHandler) List(c *gin.Context) {
	page, pageSize := parsePage(c)
	userID := middleware.GetCurrentUserID(c)
	isAdmin := middleware.GetCurrentUserIsAdmin(c)
	keyword := c.Query("keyword")
	status := c.Query("status")
	sortBy := c.DefaultQuery("sort_by", "updated_at")
	order := c.DefaultQuery("order", "desc")

	var ownerID *uint
	if s := c.Query("owner_id"); s != "" {
		v := parseID(s)
		ownerID = &v
	}

	projects, total, err := h.projectService.List(userID, isAdmin, keyword, status, ownerID, page, pageSize, sortBy, order)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	list := make([]gin.H, 0, len(projects))
	for _, p := range projects {
		item := gin.H{
			"id":                       p.ID,
			"name":                     p.Name,
			"description":              p.Description,
			"status":                   p.Status,
			"member_count":             h.projectService.GetMemberCount(p.ID),
			"repo_count":              h.projectService.GetRepoCount(p.ID),
			"requirement_count":        h.projectService.GetRequirementCount(p.ID),
			"open_requirement_count":   h.projectService.GetOpenRequirementCount(p.ID),
			"created_at":              p.CreatedAt,
			"updated_at":              p.UpdatedAt,
		}
		if p.Owner != nil {
			item["owner"] = p.Owner.Brief()
		}
		list = append(list, item)
	}
	SuccessPaged(c, list, total, page, pageSize)
}

// GET /projects/:id
func (h *ProjectHandler) GetDetail(c *gin.Context) {
	id := parseID(c.Param("id"))
	userID := middleware.GetCurrentUserID(c)

	project, err := h.projectService.GetByID(id)
	if err != nil {
		NotFound(c, 40402, "项目不存在")
		return
	}

	if !middleware.GetCurrentUserIsAdmin(c) && !h.projectService.IsMember(id, userID) {
		Forbidden(c, 40302, "非项目成员，无权查看")
		return
	}

	members := make([]gin.H, 0)
	for _, m := range project.Members {
		item := gin.H{
			"id":        m.UserID,
			"role":      m.Role,
			"joined_at": m.JoinedAt,
		}
		if m.User != nil {
			item["name"] = m.User.Name
			item["avatar"] = m.User.Avatar
		}
		members = append(members, item)
	}

	stats := h.projectService.GetProjectStats(id)

	Success(c, gin.H{
		"id":          project.ID,
		"name":        project.Name,
		"description": project.Description,
		"doc_links":   project.DocLinks,
		"owner":       project.Owner.Brief(),
		"members":     members,
		"stats":       stats,
		"status":      project.Status,
		"created_at":  project.CreatedAt,
		"updated_at":  project.UpdatedAt,
	})
}

// PUT /projects/:id
func (h *ProjectHandler) Update(c *gin.Context) {
	id := parseID(c.Param("id"))
	userID := middleware.GetCurrentUserID(c)

	project, err := h.projectService.GetByID(id)
	if err != nil {
		NotFound(c, 40402, "项目不存在")
		return
	}
	if !middleware.GetCurrentUserIsAdmin(c) && project.OwnerID != userID {
		Forbidden(c, 40303, "非项目所有者，无权编辑")
		return
	}

	var req struct {
		Name        *string         `json:"name"`
		Description *string         `json:"description"`
		DocLinks    *model.DocLinks `json:"doc_links"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, 40001, "参数校验失败: "+err.Error())
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.DocLinks != nil {
		updates["doc_links"] = *req.DocLinks
	}

	updated, err := h.projectService.Update(id, updates)
	if err != nil {
		code, msg := parseErrorCode(err)
		BadRequest(c, code, msg)
		return
	}

	Success(c, gin.H{
		"id":          updated.ID,
		"name":        updated.Name,
		"description": updated.Description,
		"doc_links":   updated.DocLinks,
		"updated_at":  updated.UpdatedAt,
	})
}

// PUT /projects/:id/archive
func (h *ProjectHandler) Archive(c *gin.Context) {
	id := parseID(c.Param("id"))
	userID := middleware.GetCurrentUserID(c)

	project, err := h.projectService.GetByID(id)
	if err != nil {
		NotFound(c, 40402, "项目不存在")
		return
	}
	if !middleware.GetCurrentUserIsAdmin(c) && project.OwnerID != userID {
		Forbidden(c, 40303, "非项目所有者，无权归档")
		return
	}

	if err := h.projectService.Archive(id); err != nil {
		code, msg := parseErrorCode(err)
		BadRequest(c, code, msg)
		return
	}

	Success(c, gin.H{"id": id, "status": "archived"})
}

// POST /projects/:id/members
func (h *ProjectHandler) AddMembers(c *gin.Context) {
	id := parseID(c.Param("id"))
	userID := middleware.GetCurrentUserID(c)

	project, err := h.projectService.GetByID(id)
	if err != nil {
		NotFound(c, 40402, "项目不存在")
		return
	}
	if !middleware.GetCurrentUserIsAdmin(c) && project.OwnerID != userID {
		Forbidden(c, 40303, "非项目所有者，无权添加成员")
		return
	}

	var req struct {
		UserIDs []uint `json:"user_ids" binding:"required"`
		Role    string `json:"role" binding:"required,oneof=pm rd"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, 40001, "参数校验失败: "+err.Error())
		return
	}

	added, skipped, err := h.projectService.AddMembers(id, req.UserIDs, req.Role)
	if err != nil {
		code, msg := parseErrorCode(err)
		BadRequest(c, code, msg)
		return
	}

	Success(c, gin.H{"added": added, "skipped": skipped})
}

// DELETE /projects/:id/members/:user_id
func (h *ProjectHandler) RemoveMember(c *gin.Context) {
	projectID := parseID(c.Param("id"))
	memberUserID := parseID(c.Param("user_id"))
	userID := middleware.GetCurrentUserID(c)

	project, err := h.projectService.GetByID(projectID)
	if err != nil {
		NotFound(c, 40402, "项目不存在")
		return
	}
	if !middleware.GetCurrentUserIsAdmin(c) && project.OwnerID != userID {
		Forbidden(c, 40303, "非项目所有者，无权移除成员")
		return
	}

	if err := h.projectService.RemoveMember(projectID, memberUserID); err != nil {
		code, msg := parseErrorCode(err)
		BadRequest(c, code, msg)
		return
	}

	Success(c, gin.H{"message": "成员已移除"})
}
