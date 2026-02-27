package bot

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/codeMaster/backend/internal/model"
	"github.com/codeMaster/backend/internal/service"
	"github.com/codeMaster/backend/pkg/feishu"
	"gorm.io/gorm"
)

// MessageHandler handles incoming bot messages, dispatching to commands or AI chat.
type MessageHandler struct {
	botClient  *feishu.BotClient
	aiChat     *AIChatClient
	db         *gorm.DB
	projectSvc *service.ProjectService
	reqSvc     *service.RequirementService
	reviewSvc  *service.ReviewService
	cmdHandler *CommandHandler
}

func NewMessageHandler(
	botClient *feishu.BotClient,
	aiChat *AIChatClient,
	db *gorm.DB,
	projectSvc *service.ProjectService,
	reqSvc *service.RequirementService,
	reviewSvc *service.ReviewService,
	codegenSvc *service.CodegenService,
	repoSvc *service.RepositoryService,
) *MessageHandler {
	cmdHandler := NewCommandHandler(botClient, db, projectSvc, reqSvc, reviewSvc, codegenSvc, repoSvc)
	return &MessageHandler{
		botClient:  botClient,
		aiChat:     aiChat,
		db:         db,
		projectSvc: projectSvc,
		reqSvc:     reqSvc,
		reviewSvc:  reviewSvc,
		cmdHandler: cmdHandler,
	}
}

// HandleMessage processes a received message event.
func (h *MessageHandler) HandleMessage(senderOpenID, messageID, msgType, content string) {
	if msgType != "text" {
		h.botClient.ReplyTextMessage(messageID, "暂时只支持文本消息哦")
		return
	}

	text := extractText(content)
	if text == "" {
		return
	}

	// Find user by feishu open_id
	user := h.findUserByOpenID(senderOpenID)

	// 1. Slash commands
	if strings.HasPrefix(text, "/") {
		if strings.TrimSpace(text) == "/clear" {
			if h.aiChat != nil {
				h.aiChat.ClearHistory(senderOpenID)
			}
			h.botClient.ReplyTextMessage(messageID, "AI 对话历史已清除")
			return
		}
		if user == nil {
			h.botClient.ReplyTextMessage(messageID, "请先通过 CodeMaster Web 端完成飞书登录后再使用命令功能")
			return
		}
		h.cmdHandler.Handle(messageID, user, text)
		return
	}

	// 2. AI intent classification → route to command if recognized
	if user != nil && h.aiChat != nil {
		if cmd := h.aiChat.ClassifyIntent(text); cmd != "" {
			log.Printf("[bot] intent classified: %q → %s", text, cmd)
			h.cmdHandler.Handle(messageID, user, cmd)
			return
		}
	}

	// 3. Fallback to AI chat
	h.handleAIChat(senderOpenID, messageID, text)
}

func (h *MessageHandler) handleAIChat(senderOpenID, messageID, text string) {
	if h.aiChat == nil {
		h.botClient.ReplyTextMessage(messageID, "AI 聊天功能未配置，请联系管理员设置 ai_chat 配置")
		return
	}

	reply, err := h.aiChat.Chat(senderOpenID, text)
	if err != nil {
		log.Printf("[bot] AI chat error: %v", err)
		h.botClient.ReplyTextMessage(messageID, "AI 聊天出错，请稍后重试")
		return
	}

	if err := h.botClient.ReplyTextMessage(messageID, reply); err != nil {
		log.Printf("[bot] reply AI chat failed: %v", err)
	}
}

func (h *MessageHandler) findUserByOpenID(openID string) *model.User {
	var user model.User
	if err := h.db.Where("feishu_uid = ?", openID).First(&user).Error; err != nil {
		return nil
	}
	return &user
}

// extractText parses the text from a Feishu message content JSON.
func extractText(content string) string {
	var msg struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(content), &msg); err != nil {
		return ""
	}
	text := strings.TrimSpace(msg.Text)
	// Remove @mentions (format: @_user_N )
	for strings.Contains(text, "@_user_") {
		start := strings.Index(text, "@_user_")
		end := strings.Index(text[start:], " ")
		if end == -1 {
			text = text[:start]
		} else {
			text = text[:start] + text[start+end+1:]
		}
	}
	return strings.TrimSpace(text)
}
