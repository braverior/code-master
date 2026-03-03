package handler

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/codeMaster/backend/internal/model"
	"github.com/codeMaster/backend/internal/service"
	"github.com/codeMaster/backend/pkg/feishu"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type OpenHandler struct {
	rdb        *redis.Client
	reqService *service.RequirementService
	docClient  *feishu.DocClient
}

func NewOpenHandler(rdb *redis.Client, reqService *service.RequirementService, docClient *feishu.DocClient) *OpenHandler {
	return &OpenHandler{
		rdb:        rdb,
		reqService: reqService,
		docClient:  docClient,
	}
}

// GET /open/requirements/:id?token=xxx
func (h *OpenHandler) GetRequirementDetail(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		Unauthorized(c, 40101, "缺少 token 参数")
		return
	}

	// Look up token in Redis
	key := fmt.Sprintf("share_token:%s", tokenStr)
	val, err := h.rdb.Get(context.Background(), key).Result()
	if err != nil {
		Unauthorized(c, 40102, "token 无效或已过期")
		return
	}

	tokenReqID, _ := strconv.ParseUint(val, 10, 64)
	id := parseID(c.Param("id"))
	if id == 0 || uint(tokenReqID) != id {
		Forbidden(c, 40301, "token 与需求 ID 不匹配")
		return
	}

	req, err := h.reqService.GetByID(id)
	if err != nil {
		NotFound(c, 40404, "需求不存在")
		return
	}

	// Build response: only requirement essentials + doc content
	data := gin.H{
		"title":       req.Title,
		"description": req.Description,
		"doc_content": h.fetchDocContent(req.DocLinks),
	}

	Success(c, data)
}

func (h *OpenHandler) fetchDocContent(docLinks model.DocLinks) string {
	if h.docClient == nil || len(docLinks) == 0 {
		return ""
	}

	var parts []string
	for _, link := range docLinks {
		token := feishu.ExtractDocToken(link.URL)
		if token == "" {
			continue
		}
		content, err := h.docClient.GetDocContent(token)
		if err != nil {
			log.Printf("[open] fetch doc %q failed: %v", link.Title, err)
			continue
		}
		if content == "" {
			continue
		}
		title := link.Title
		if title == "" {
			title = link.URL
		}
		parts = append(parts, fmt.Sprintf("### %s\n\n%s", title, content))
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n\n---\n\n")
}
