package main

import (
	"fmt"
	"log"
	"os"

	"github.com/codeMaster/backend/internal/bot"
	"github.com/codeMaster/backend/internal/codegen"
	"github.com/codeMaster/backend/internal/config"
	"github.com/codeMaster/backend/internal/gitops"
	"github.com/codeMaster/backend/internal/handler"
	"github.com/codeMaster/backend/internal/model"
	"github.com/codeMaster/backend/internal/notify"
	"github.com/codeMaster/backend/internal/router"
	"github.com/codeMaster/backend/internal/service"
	"github.com/codeMaster/backend/internal/sse"
	"github.com/codeMaster/backend/pkg/feishu"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// Load config
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Initialize git domain mapping
	var domainItems []gitops.DomainMappingItem
	for _, m := range cfg.Codegen.GitDomainMapping {
		domainItems = append(domainItems, gitops.DomainMappingItem{From: m.From, To: m.To})
	}
	gitops.InitDomainMapping(domainItems)

	// Database
	db, err := gorm.Open(mysql.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(
		&model.User{},
		&model.Project{},
		&model.ProjectMember{},
		&model.Repository{},
		&model.Requirement{},
		&model.CodegenTask{},
		&model.CodeReview{},
		&model.OperationLog{},
		&model.UserSetting{},
	); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	// One-time migration: convert role='admin' to is_admin=true, role='rd'
	db.Model(&model.User{}).Where("role = ?", "admin").Updates(map[string]interface{}{
		"is_admin": true,
		"role":     "rd",
	})

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Core components
	sseHub := sse.NewHub(rdb)
	pool := codegen.NewPool(cfg.Codegen.MaxWorkers)
	analyzer := codegen.NewAnalyzer(db, cfg.Encrypt.AESKey, cfg.Codegen.WorkDir)

	// Feishu client
	feishuOAuth := feishu.NewOAuthClient(cfg.Feishu.AppID, cfg.Feishu.AppSecret, cfg.Feishu.RedirectURI)

	// BotClient (for sending messages)
	botClient := feishu.NewBotClient(feishuOAuth)

	// DocClient (for fetching Feishu documents)
	docClient := feishu.NewDocClient(cfg.Feishu.AppID, cfg.Feishu.AppSecret)

	// Notifier
	var notifier notify.Notifier
	if cfg.Feishu.Bot.Enabled {
		notifier = notify.NewFeishuNotifier(botClient)
	} else {
		notifier = notify.NoopNotifier{}
	}

	// Services
	authService := service.NewAuthService(db, feishuOAuth, cfg.JWT.Secret, cfg.JWT.ExpireHours)
	projectService := service.NewProjectService(db)
	repoService := service.NewRepositoryService(db, cfg.Encrypt.AESKey, analyzer)
	reqService := service.NewRequirementService(db)
	codegenService := service.NewCodegenService(db, pool, sseHub, cfg.Encrypt.AESKey, cfg.Codegen.MaxTurns, cfg.Codegen.TimeoutMinutes, cfg.Codegen.WorkDir, cfg.Codegen.UseLocalGit)
	reviewService := service.NewReviewService(db, cfg.Encrypt.AESKey, cfg.Codegen.WorkDir)
	settingService := service.NewSettingService(db, cfg.Encrypt.AESKey)

	// Inject notifiers
	codegenService.SetNotifier(notifier)
	codegenService.SetDocClient(docClient)
	reviewService.SetNotifier(notifier)

	// AI Chat client
	var aiChat *bot.AIChatClient
	if cfg.AIChat.APIKey != "" {
		aiChat = bot.NewAIChatClient(cfg.AIChat)
	}

	// Feishu Bot (WebSocket long-connection)
	if cfg.Feishu.Bot.Enabled {
		feishuBot := bot.New(bot.BotDeps{
			AppID:             cfg.Feishu.AppID,
			AppSecret:         cfg.Feishu.AppSecret,
			EncryptKey:        cfg.Feishu.Bot.EncryptKey,
			VerificationToken: cfg.Feishu.Bot.VerificationToken,
			BotClient:         botClient,
			AIChatClient:      aiChat,
			DB:                db,
			ProjectService:    projectService,
			ReqService:        reqService,
			ReviewService:     reviewService,
			CodegenService:    codegenService,
			RepoService:       repoService,
		})
		go feishuBot.Start()
		defer feishuBot.Stop()
	}

	// Handlers
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(authService)
	projectHandler := handler.NewProjectHandler(projectService)
	repoHandler := handler.NewRepositoryHandler(repoService, projectService)
	requirementHandler := handler.NewRequirementHandler(reqService, projectService, notifier)
	codegenHandler := handler.NewCodegenHandler(codegenService, reqService, repoService, reviewService)
	reviewHandler := handler.NewReviewHandler(reviewService)
	dashboardHandler := handler.NewDashboardHandler(db)
	feishuHandler := handler.NewFeishuHandler(docClient)
	settingHandler := handler.NewSettingHandler(settingService)

	// Gin engine
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// Setup routes
	router.Setup(r, router.Deps{
		DB:                 db,
		JWTSecret:          cfg.JWT.Secret,
		AuthHandler:        authHandler,
		UserHandler:        userHandler,
		ProjectHandler:     projectHandler,
		RepoHandler:        repoHandler,
		RequirementHandler: requirementHandler,
		CodegenHandler:     codegenHandler,
		ReviewHandler:      reviewHandler,
		DashboardHandler:   dashboardHandler,
		FeishuHandler:      feishuHandler,
		SettingHandler:     settingHandler,
	})

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server run: %v", err)
	}
}
