package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/codeMaster/backend/internal/service"
	"github.com/codeMaster/backend/pkg/feishu"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"gorm.io/gorm"
)

// BotDeps holds the dependencies needed to create a Bot.
type BotDeps struct {
	AppID             string
	AppSecret         string
	EncryptKey        string
	VerificationToken string
	BotClient         *feishu.BotClient
	AIChatClient      *AIChatClient
	DB                *gorm.DB
	ProjectService    *service.ProjectService
	ReqService        *service.RequirementService
	ReviewService     *service.ReviewService
	CodegenService    *service.CodegenService
	RepoService       *service.RepositoryService
}

// Bot manages the Feishu WebSocket long-connection lifecycle.
type Bot struct {
	wsClient *larkws.Client
	handler  *MessageHandler
	ctx      context.Context
	cancel   context.CancelFunc
}

// New creates a new Bot instance with WebSocket client and message handler.
func New(deps BotDeps) *Bot {
	ctx, cancel := context.WithCancel(context.Background())

	handler := NewMessageHandler(
		deps.BotClient,
		deps.AIChatClient,
		deps.DB,
		deps.ProjectService,
		deps.ReqService,
		deps.ReviewService,
		deps.CodegenService,
		deps.RepoService,
	)

	eventDispatcher := dispatcher.NewEventDispatcher(deps.VerificationToken, deps.EncryptKey)
	eventDispatcher.OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
		if event == nil || event.Event == nil || event.Event.Message == nil || event.Event.Sender == nil {
			return nil
		}

		msg := event.Event.Message
		sender := event.Event.Sender

		// Skip bot's own messages
		if sender.SenderType != nil && *sender.SenderType == "app" {
			return nil
		}

		senderOpenID := ""
		if sender.SenderId != nil && sender.SenderId.OpenId != nil {
			senderOpenID = *sender.SenderId.OpenId
		}

		messageID := ""
		if msg.MessageId != nil {
			messageID = *msg.MessageId
		}

		msgType := ""
		if msg.MessageType != nil {
			msgType = *msg.MessageType
		}

		content := ""
		if msg.Content != nil {
			content = *msg.Content
		}

		go handler.HandleMessage(senderOpenID, messageID, msgType, content)
		return nil
	})

	wsClient := larkws.NewClient(
		deps.AppID,
		deps.AppSecret,
		larkws.WithEventHandler(eventDispatcher),
		larkws.WithLogLevel(larkcore.LogLevelInfo),
	)

	return &Bot{
		wsClient: wsClient,
		handler:  handler,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins the WebSocket connection. This blocks until the context is cancelled.
func (b *Bot) Start() {
	log.Println("[bot] Starting Feishu bot with WebSocket long-connection...")
	if err := b.wsClient.Start(b.ctx); err != nil {
		if b.ctx.Err() != nil {
			log.Println("[bot] Bot stopped by context cancellation")
			return
		}
		log.Printf("[bot] WebSocket connection error: %v", err)
	}
}

// Stop gracefully stops the bot.
func (b *Bot) Stop() {
	log.Println("[bot] Stopping Feishu bot...")
	b.cancel()
}

// String returns a descriptive string for the bot.
func (b *Bot) String() string {
	return fmt.Sprintf("FeishuBot(ws)")
}
