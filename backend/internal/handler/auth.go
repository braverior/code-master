package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/codeMaster/backend/internal/middleware"
	"github.com/codeMaster/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// GET /auth/feishu/login
func (h *AuthHandler) FeishuLogin(c *gin.Context) {
	redirectURI := c.DefaultQuery("redirect_uri", "/")
	// Encode redirect_uri into state so it survives the OAuth round-trip.
	// Format: "<random_hex>|<redirect_uri>"
	state := generateState() + "|" + redirectURI
	authURL := h.authService.GetFeishuAuthURL(state)
	c.Redirect(http.StatusFound, authURL)
}

// GET /auth/feishu/callback
func (h *AuthHandler) FeishuCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		BadRequest(c, 40001, "code 不能为空")
		return
	}

	user, token, _, isNew, err := h.authService.HandleCallback(code)
	if err != nil {
		Error(c, http.StatusInternalServerError, 50105, "飞书授权失败: "+err.Error())
		return
	}

	// Extract redirect_uri from state parameter (format: "<random_hex>|<redirect_uri>")
	redirectURI := "/"
	if state := c.Query("state"); state != "" {
		if idx := strings.Index(state, "|"); idx >= 0 {
			redirectURI = state[idx+1:]
		}
	}

	target := fmt.Sprintf("%s?token=%s&is_new_user=%t", redirectURI, token, isNew)
	_ = user
	c.Redirect(http.StatusFound, target)
}

// GET /auth/me
func (h *AuthHandler) GetMe(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		Unauthorized(c, 40103, "用户未认证")
		return
	}
	isNew := user.Role == "rd" && user.LastLoginAt == nil
	Success(c, gin.H{
		"id":            user.ID,
		"name":          user.Name,
		"avatar":        user.Avatar,
		"email":         user.Email,
		"role":          user.Role,
		"is_admin":      user.IsAdmin,
		"status":        user.Status,
		"is_new_user":   isNew,
		"last_login_at": user.LastLoginAt,
		"created_at":    user.CreatedAt,
	})
}

// PUT /auth/role
func (h *AuthHandler) UpdateRole(c *gin.Context) {
	var req struct {
		UserID *uint  `json:"user_id"`
		Role   string `json:"role" binding:"required,oneof=pm rd"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, 40001, "参数校验失败: "+err.Error())
		return
	}

	currentUser := middleware.GetCurrentUser(c)

	if req.UserID != nil {
		// Admin modifying another user
		if !currentUser.IsAdmin {
			Forbidden(c, 40301, "权限不足，仅管理员可修改他人角色")
			return
		}
		user, err := h.authService.UpdateRole(*req.UserID, req.Role)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
		Success(c, gin.H{
			"id":         user.ID,
			"name":       user.Name,
			"role":       user.Role,
			"updated_at": user.UpdatedAt,
		})
		return
	}

	// Self role selection (first time only)
	user, err := h.authService.UpdateRole(currentUser.ID, req.Role)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{
		"id":         user.ID,
		"name":       user.Name,
		"role":       user.Role,
		"updated_at": user.UpdatedAt,
	})
}

// POST /auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	token, expireAt, err := h.authService.RefreshToken(userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{
		"token":     token,
		"expire_at": expireAt,
	})
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
