package handler

import (
	"github.com/codeMaster/backend/pkg/feishu"
	"github.com/gin-gonic/gin"
)

type FeishuHandler struct {
	docClient *feishu.DocClient
}

func NewFeishuHandler(docClient *feishu.DocClient) *FeishuHandler {
	return &FeishuHandler{docClient: docClient}
}

// POST /feishu/doc/resolve
func (h *FeishuHandler) ResolveDoc(c *gin.Context) {
	var body struct {
		URL string `json:"url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, 40001, "参数校验失败: "+err.Error())
		return
	}

	docToken := feishu.ExtractDocToken(body.URL)
	if docToken == "" {
		BadRequest(c, 40001, "无法从 URL 中提取文档 ID")
		return
	}

	meta, err := h.docClient.GetDocMeta(docToken)
	if err != nil {
		BadRequest(c, 40002, "获取文档信息失败: "+err.Error())
		return
	}

	Success(c, gin.H{
		"title":       meta.Title,
		"document_id": meta.DocumentID,
		"url":         body.URL,
	})
}
