package handler

import (
	"strconv"
	"time"

	"github.com/codeMaster/backend/internal/middleware"
	"github.com/codeMaster/backend/internal/model"
	"github.com/codeMaster/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	authService *service.AuthService
}

func NewUserHandler(authService *service.AuthService) *UserHandler {
	return &UserHandler{authService: authService}
}

// GET /admin/users
func (h *UserHandler) ListUsers(c *gin.Context) {
	page, pageSize := parsePage(c)
	keyword := c.Query("keyword")
	role := c.Query("role")
	sortBy := c.DefaultQuery("sort_by", "created_at")
	order := c.DefaultQuery("order", "desc")

	var status *int
	if s := c.Query("status"); s != "" {
		v, _ := strconv.Atoi(s)
		status = &v
	}

	var isAdmin *bool
	if s := c.Query("is_admin"); s != "" {
		v := s == "true" || s == "1"
		isAdmin = &v
	}

	users, total, err := h.authService.ListUsers(keyword, role, isAdmin, status, page, pageSize, sortBy, order)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	list := make([]gin.H, 0, len(users))
	for _, u := range users {
		list = append(list, gin.H{
			"id":            u.ID,
			"name":          u.Name,
			"avatar":        u.Avatar,
			"email":         u.Email,
			"role":          u.Role,
			"is_admin":      u.IsAdmin,
			"status":        u.Status,
			"last_login_at": u.LastLoginAt,
			"created_at":    u.CreatedAt,
		})
	}
	SuccessPaged(c, list, total, page, pageSize)
}

// PUT /admin/users/:id/role
func (h *UserHandler) UpdateUserRole(c *gin.Context) {
	id := parseID(c.Param("id"))
	var req struct {
		Role string `json:"role" binding:"required,oneof=pm rd"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, 40001, "参数校验失败: "+err.Error())
		return
	}

	user, err := h.authService.UpdateRole(id, req.Role)
	if err != nil {
		NotFound(c, 40401, "用户不存在")
		return
	}
	Success(c, gin.H{
		"id":         user.ID,
		"name":       user.Name,
		"role":       user.Role,
		"is_admin":   user.IsAdmin,
		"updated_at": user.UpdatedAt,
	})
}

// PUT /admin/users/:id/admin
func (h *UserHandler) ToggleUserAdmin(c *gin.Context) {
	id := parseID(c.Param("id"))
	var req struct {
		IsAdmin bool `json:"is_admin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, 40001, "参数校验失败: "+err.Error())
		return
	}

	user, err := h.authService.ToggleAdmin(id, req.IsAdmin)
	if err != nil {
		NotFound(c, 40401, "用户不存在")
		return
	}
	Success(c, gin.H{
		"id":         user.ID,
		"name":       user.Name,
		"role":       user.Role,
		"is_admin":   user.IsAdmin,
		"updated_at": user.UpdatedAt,
	})
}

// PUT /admin/users/:id/status
func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	id := parseID(c.Param("id"))
	currentUserID := middleware.GetCurrentUserID(c)
	if id == currentUserID {
		BadRequest(c, 40003, "不能禁用当前登录账号")
		return
	}

	var req struct {
		Status int `json:"status" binding:"oneof=0 1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, 40001, "参数校验失败: "+err.Error())
		return
	}

	user, err := h.authService.UpdateUserStatus(id, req.Status)
	if err != nil {
		NotFound(c, 40401, "用户不存在")
		return
	}
	Success(c, gin.H{
		"id":         user.ID,
		"name":       user.Name,
		"status":     user.Status,
		"updated_at": user.UpdatedAt,
	})
}

// GET /admin/operation-logs
func (h *UserHandler) GetOperationLogs(c *gin.Context) {
	page, pageSize := parsePage(c)

	var userID *uint
	if s := c.Query("user_id"); s != "" {
		v := parseID(s)
		userID = &v
	}
	action := c.Query("action")
	resourceType := c.Query("resource_type")

	var startTime, endTime *time.Time
	if s := c.Query("start_time"); s != "" {
		t, _ := time.Parse(time.RFC3339, s)
		startTime = &t
	}
	if s := c.Query("end_time"); s != "" {
		t, _ := time.Parse(time.RFC3339, s)
		endTime = &t
	}

	logs, total, err := h.authService.GetOperationLogs(userID, action, resourceType, startTime, endTime, page, pageSize)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	list := make([]gin.H, 0, len(logs))
	for _, log := range logs {
		item := gin.H{
			"id":            log.ID,
			"action":        log.Action,
			"resource_type": log.ResourceType,
			"resource_id":   log.ResourceID,
			"detail":        log.Detail,
			"ip":            log.IP,
			"created_at":    log.CreatedAt,
		}
		if log.User != nil {
			item["user"] = gin.H{"id": log.User.ID, "name": log.User.Name}
		}
		list = append(list, item)
	}
	SuccessPaged(c, list, total, page, pageSize)
}

// GET /users/search
func (h *UserHandler) SearchUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		BadRequest(c, 40001, "keyword 不能为空")
		return
	}
	role := c.Query("role")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit > 50 {
		limit = 50
	}

	var excludeProjectID *uint
	if s := c.Query("exclude_project_id"); s != "" {
		v := parseID(s)
		excludeProjectID = &v
	}

	users, err := h.authService.SearchUsers(keyword, role, excludeProjectID, limit)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	list := make([]gin.H, 0, len(users))
	for _, u := range users {
		list = append(list, gin.H{
			"id":       u.ID,
			"name":     u.Name,
			"avatar":   u.Avatar,
			"email":    u.Email,
			"role":     u.Role,
			"is_admin": u.IsAdmin,
		})
	}
	Success(c, list)
}

func LogOperation(authService *service.AuthService, c *gin.Context, action, resourceType string, resourceID uint, detail map[string]interface{}) {
	userID := middleware.GetCurrentUserID(c)
	authService.CreateOperationLog(&model.OperationLog{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Detail:       detail,
		IP:           c.ClientIP(),
	})
}
