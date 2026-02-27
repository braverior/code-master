package router

import (
	"github.com/codeMaster/backend/internal/handler"
	"github.com/codeMaster/backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Deps struct {
	DB                 *gorm.DB
	JWTSecret          string
	AuthHandler        *handler.AuthHandler
	UserHandler        *handler.UserHandler
	ProjectHandler     *handler.ProjectHandler
	RepoHandler        *handler.RepositoryHandler
	RequirementHandler *handler.RequirementHandler
	CodegenHandler     *handler.CodegenHandler
	ReviewHandler      *handler.ReviewHandler
	DashboardHandler   *handler.DashboardHandler
	FeishuHandler      *handler.FeishuHandler
	SettingHandler     *handler.SettingHandler
}

func Setup(r *gin.Engine, deps Deps) {
	r.Use(middleware.CORSMiddleware())

	api := r.Group("/api/v1")

	// Public routes (no auth)
	auth := api.Group("/auth")
	{
		auth.GET("/feishu/login", deps.AuthHandler.FeishuLogin)
		auth.GET("/feishu/callback", deps.AuthHandler.FeishuCallback)
	}

	// Authenticated routes
	authed := api.Group("")
	authed.Use(middleware.AuthMiddleware(deps.JWTSecret, deps.DB))
	{
		// Auth
		authed.GET("/auth/me", deps.AuthHandler.GetMe)
		authed.PUT("/auth/role", deps.AuthHandler.UpdateRole)
		authed.POST("/auth/refresh", deps.AuthHandler.RefreshToken)

		// User search (all authenticated users)
		authed.GET("/users/search", deps.UserHandler.SearchUsers)

		// Admin routes
		admin := authed.Group("/admin")
		admin.Use(middleware.RequireAdmin())
		{
			admin.GET("/users", deps.UserHandler.ListUsers)
			admin.PUT("/users/:id/role", deps.UserHandler.UpdateUserRole)
			admin.PUT("/users/:id/admin", deps.UserHandler.ToggleUserAdmin)
			admin.PUT("/users/:id/status", deps.UserHandler.UpdateUserStatus)
			admin.GET("/operation-logs", deps.UserHandler.GetOperationLogs)
		}

		// Projects
		projects := authed.Group("/projects")
		{
			projects.POST("", middleware.RequireRole("pm"), deps.ProjectHandler.Create)
			projects.GET("", deps.ProjectHandler.List)
			projects.GET("/:id", deps.ProjectHandler.GetDetail)
			projects.PUT("/:id", deps.ProjectHandler.Update)
			projects.PUT("/:id/archive", deps.ProjectHandler.Archive)
			projects.POST("/:id/members", deps.ProjectHandler.AddMembers)
			projects.DELETE("/:id/members/:user_id", deps.ProjectHandler.RemoveMember)

			// Repositories under projects
			projects.POST("/:id/repos", deps.RepoHandler.Create)
			projects.GET("/:id/repos", deps.RepoHandler.List)

			// Requirements under projects
			projects.POST("/:id/requirements", middleware.RequireRole("pm"), deps.RequirementHandler.Create)
			projects.GET("/:id/requirements", deps.RequirementHandler.List)
		}

		// Repositories (standalone)
		repos := authed.Group("/repos")
		{
			repos.GET("/:id", deps.RepoHandler.GetDetail)
			repos.PUT("/:id", deps.RepoHandler.Update)
			repos.DELETE("/:id", deps.RepoHandler.Delete)
			repos.POST("/:id/test-connection", deps.RepoHandler.TestConnection)
			repos.POST("/:id/analyze", deps.RepoHandler.Analyze)
			repos.GET("/:id/analysis", deps.RepoHandler.GetAnalysis)
		}

		// Requirements (standalone)
		requirements := authed.Group("/requirements")
		{
			requirements.GET("", deps.RequirementHandler.ListAll)
			requirements.GET("/:id", deps.RequirementHandler.GetDetail)
			requirements.PUT("/:id", deps.RequirementHandler.Update)
			requirements.DELETE("/:id", deps.RequirementHandler.Delete)

			// Code generation
			requirements.POST("/:id/generate", deps.CodegenHandler.Generate)
			requirements.POST("/:id/manual-submit", deps.CodegenHandler.ManualSubmit)
			requirements.GET("/:id/codegen-tasks", deps.CodegenHandler.ListTasks)
		}

		// CodeGen tasks
		codegen := authed.Group("/codegen")
		{
			codegen.GET("/:id", deps.CodegenHandler.GetTask)
			codegen.GET("/:id/stream", deps.CodegenHandler.Stream)
			codegen.GET("/:id/diff", deps.CodegenHandler.GetDiff)
			codegen.GET("/:id/log", deps.CodegenHandler.GetLog)
			codegen.POST("/:id/cancel", deps.CodegenHandler.Cancel)

			// Review under codegen
			codegen.POST("/:id/review", deps.ReviewHandler.TriggerAIReview)
			codegen.GET("/:id/review", deps.ReviewHandler.GetReview)
		}

		// Reviews
		reviews := authed.Group("/reviews")
		{
			reviews.GET("/pending", deps.ReviewHandler.ListPending)
			reviews.GET("/list", deps.ReviewHandler.ListReviews)
			reviews.GET("/:id", deps.ReviewHandler.GetReviewByID)
			reviews.PUT("/:id/human", deps.ReviewHandler.SubmitHumanReview)
			reviews.POST("/:id/merge-request", deps.ReviewHandler.CreateMergeRequest)
			reviews.GET("/:id/merge-request", deps.ReviewHandler.GetMergeRequestStatus)
		}

		// Settings
		settings := authed.Group("/settings")
		{
			settings.GET("/llm", deps.SettingHandler.GetLLMSettings)
			settings.PUT("/llm", deps.SettingHandler.UpdateLLMSettings)
		}

		// Dashboard
		dashboard := authed.Group("/dashboard")
		{
			dashboard.GET("/stats", deps.DashboardHandler.GetStats)
			dashboard.GET("/my-tasks", deps.DashboardHandler.GetMyTasks)
		}

		// Feishu utilities
		if deps.FeishuHandler != nil {
			feishuGroup := authed.Group("/feishu")
			{
				feishuGroup.POST("/doc/resolve", deps.FeishuHandler.ResolveDoc)
			}
		}
	}
}
