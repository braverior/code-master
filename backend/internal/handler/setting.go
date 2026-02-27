package handler

import (
	"strings"

	"github.com/codeMaster/backend/internal/middleware"
	"github.com/codeMaster/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type SettingHandler struct {
	settingService *service.SettingService
}

func NewSettingHandler(settingService *service.SettingService) *SettingHandler {
	return &SettingHandler{settingService: settingService}
}

// GET /settings/llm
func (h *SettingHandler) GetLLMSettings(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	setting, err := h.settingService.GetByUserID(userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	if setting == nil {
		Success(c, gin.H{
			"base_url":     "",
			"api_key":      "",
			"model":        "",
			"gitlab_token": "",
		})
		return
	}

	Success(c, gin.H{
		"base_url":     setting.BaseURL,
		"api_key":      maskSecret(setting.APIKey, "sk-****"),
		"model":        setting.Model,
		"gitlab_token": maskSecret(setting.GitlabToken, "****"),
	})
}

// PUT /settings/llm
func (h *SettingHandler) UpdateLLMSettings(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)

	var req struct {
		BaseURL     string `json:"base_url"`
		APIKey      string `json:"api_key"`
		Model       string `json:"model"`
		GitlabToken string `json:"gitlab_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, 40001, "参数校验失败: "+err.Error())
		return
	}

	// If secrets look masked (contain ****), keep the old values
	if strings.Contains(req.APIKey, "****") || strings.Contains(req.GitlabToken, "****") {
		existing, _ := h.settingService.GetByUserID(userID)
		if existing != nil {
			if strings.Contains(req.APIKey, "****") {
				req.APIKey = existing.APIKey
			}
			if strings.Contains(req.GitlabToken, "****") {
				req.GitlabToken = existing.GitlabToken
			}
		}
	}

	setting, err := h.settingService.Upsert(userID, req.BaseURL, req.APIKey, req.Model, req.GitlabToken)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"base_url":     setting.BaseURL,
		"api_key":      maskSecret(setting.APIKey, "sk-****"),
		"model":        setting.Model,
		"gitlab_token": maskSecret(setting.GitlabToken, "****"),
	})
}

func maskSecret(value, prefix string) string {
	if len(value) <= 4 {
		return value
	}
	return prefix + value[len(value)-4:]
}
