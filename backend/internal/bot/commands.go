package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/codeMaster/backend/internal/model"
	"github.com/codeMaster/backend/internal/service"
	"github.com/codeMaster/backend/pkg/feishu"
	"gorm.io/gorm"
)

// CommandHandler processes slash commands from bot messages.
type CommandHandler struct {
	botClient  *feishu.BotClient
	db         *gorm.DB
	projectSvc *service.ProjectService
	reqSvc     *service.RequirementService
	reviewSvc  *service.ReviewService
	codegenSvc *service.CodegenService
	repoSvc    *service.RepositoryService
}

func NewCommandHandler(botClient *feishu.BotClient, db *gorm.DB, projectSvc *service.ProjectService, reqSvc *service.RequirementService, reviewSvc *service.ReviewService, codegenSvc *service.CodegenService, repoSvc *service.RepositoryService) *CommandHandler {
	return &CommandHandler{
		botClient:  botClient,
		db:         db,
		projectSvc: projectSvc,
		reqSvc:     reqSvc,
		reviewSvc:  reviewSvc,
		codegenSvc: codegenSvc,
		repoSvc:    repoSvc,
	}
}

// Handle processes a command string and replies to the message.
func (h *CommandHandler) Handle(messageID string, user *model.User, text string) {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return
	}
	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	var card map[string]interface{}
	var textReply string

	switch cmd {
	case "/help":
		textReply = h.handleHelp()
	case "/projects":
		card = h.handleProjects(user)
	case "/myreqs":
		card = h.handleMyReqs(user)
	case "/reqs":
		card = h.handleReqs(user, args)
	case "/status":
		card = h.handleStatus(args)
	case "/reviews":
		card = h.handleReviews(user)
	case "/codegen":
		card = h.handleCodegen(user, args)
	default:
		textReply = fmt.Sprintf("未知命令: %s\n输入 /help 查看可用命令", cmd)
	}

	if card != nil {
		if err := h.botClient.ReplyInteractiveMessage(messageID, card); err != nil {
			log.Printf("[bot] reply card failed: %v", err)
		}
	} else if textReply != "" {
		if err := h.botClient.ReplyTextMessage(messageID, textReply); err != nil {
			log.Printf("[bot] reply text failed: %v", err)
		}
	}
}

func (h *CommandHandler) handleHelp() string {
	return `CodeMaster Bot 命令列表：

/help        - 显示此帮助
/projects    - 我的项目列表
/myreqs      - 我的需求/任务列表
/reqs [项目ID] - 某项目的需求列表
/status <需求ID> - 查看需求状态
/reviews     - 我的待审查列表
/codegen <需求ID> - 触发代码生成
/clear       - 清除 AI 对话历史

直接发送文字即可与 AI 聊天`
}

func (h *CommandHandler) handleProjects(user *model.User) map[string]interface{} {
	isAdmin := false
	if user != nil {
		isAdmin = user.IsAdmin
	}
	projects, _, err := h.projectSvc.List(user.ID, isAdmin, "", "", nil, 1, 20, "updated_at", "desc")
	if err != nil {
		log.Printf("[bot] list projects failed: %v", err)
		return BuildCard(ColorRed, "错误", []cardField{{Key: "原因", Value: "查询项目失败"}}, nil)
	}

	items := make([]ProjectItem, 0, len(projects))
	for _, p := range projects {
		reqCount := h.projectSvc.GetRequirementCount(p.ID)
		items = append(items, ProjectItem{
			ID:       p.ID,
			Name:     p.Name,
			Status:   p.Status,
			ReqCount: reqCount,
		})
	}
	return BuildProjectListCard(items)
}

func (h *CommandHandler) handleMyReqs(user *model.User) map[string]interface{} {
	reqs, _, err := h.reqSvc.ListByUser(user.ID, 1, 20)
	if err != nil {
		log.Printf("[bot] list my reqs failed: %v", err)
		return BuildCard(ColorRed, "错误", []cardField{{Key: "原因", Value: "查询需求失败"}}, nil)
	}

	items := make([]MyRequirementItem, 0, len(reqs))
	for _, r := range reqs {
		projectName := ""
		if r.Project != nil {
			projectName = r.Project.Name
		}
		assigneeName := ""
		if r.Assignee != nil {
			assigneeName = r.Assignee.Name
		}
		items = append(items, MyRequirementItem{
			ID:           r.ID,
			Title:        r.Title,
			Status:       r.Status,
			Priority:     r.Priority,
			ProjectName:  projectName,
			AssigneeName: assigneeName,
		})
	}
	return BuildMyRequirementListCard(items)
}

func (h *CommandHandler) handleReqs(user *model.User, args []string) map[string]interface{} {
	if len(args) == 0 {
		return BuildCard(ColorRed, "参数错误", []cardField{
			{Key: "用法", Value: "/reqs <项目ID>"},
		}, nil)
	}

	projectID, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return BuildCard(ColorRed, "参数错误", []cardField{
			{Key: "原因", Value: "项目ID必须是数字"},
		}, nil)
	}

	project, err := h.projectSvc.GetByID(uint(projectID))
	if err != nil {
		return BuildCard(ColorRed, "错误", []cardField{
			{Key: "原因", Value: "项目不存在"},
		}, nil)
	}

	reqs, _, err := h.reqSvc.List(uint(projectID), "", "", "", nil, nil, 1, 20, "created_at", "desc")
	if err != nil {
		log.Printf("[bot] list reqs failed: %v", err)
		return BuildCard(ColorRed, "错误", []cardField{{Key: "原因", Value: "查询需求失败"}}, nil)
	}

	items := make([]RequirementItem, 0, len(reqs))
	for _, r := range reqs {
		assigneeName := ""
		if r.Assignee != nil {
			assigneeName = r.Assignee.Name
		}
		items = append(items, RequirementItem{
			ID:           r.ID,
			Title:        r.Title,
			Status:       r.Status,
			Priority:     r.Priority,
			AssigneeName: assigneeName,
		})
	}
	return BuildRequirementListCard(project.Name, items)
}

func (h *CommandHandler) handleStatus(args []string) map[string]interface{} {
	if len(args) == 0 {
		return BuildCard(ColorRed, "参数错误", []cardField{
			{Key: "用法", Value: "/status <需求ID>"},
		}, nil)
	}

	reqID, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return BuildCard(ColorRed, "参数错误", []cardField{
			{Key: "原因", Value: "需求ID必须是数字"},
		}, nil)
	}

	req, err := h.reqSvc.GetByID(uint(reqID))
	if err != nil {
		return BuildCard(ColorRed, "错误", []cardField{
			{Key: "原因", Value: "需求不存在"},
		}, nil)
	}

	info := StatusInfo{
		RequirementID: req.ID,
		Title:         req.Title,
		Status:        req.Status,
		Priority:      req.Priority,
	}
	if req.Project != nil {
		info.ProjectName = req.Project.Name
	}

	latestTask := h.reqSvc.GetLatestCodegenTask(req.ID)
	if latestTask != nil {
		info.LatestTaskID = latestTask.ID
		info.LatestTaskStatus = latestTask.Status
	}

	latestReview := h.reqSvc.GetLatestReview(req.ID)
	if latestReview != nil {
		info.LatestReviewAIStatus = latestReview.AIStatus
		info.LatestReviewAIScore = latestReview.AIScore
		info.LatestReviewHumanStatus = latestReview.HumanStatus
	}

	return BuildStatusCard(info)
}

func (h *CommandHandler) handleReviews(user *model.User) map[string]interface{} {
	reviews, _, err := h.reviewSvc.ListPendingReviews(user.ID, nil, 1, 20)
	if err != nil {
		log.Printf("[bot] list reviews failed: %v", err)
		return BuildCard(ColorRed, "错误", []cardField{{Key: "原因", Value: "查询审查列表失败"}}, nil)
	}

	items := make([]ReviewItem, 0, len(reviews))
	for _, r := range reviews {
		reqTitle := ""
		projectName := ""
		if r.CodegenTask != nil && r.CodegenTask.Requirement != nil {
			reqTitle = r.CodegenTask.Requirement.Title
			if r.CodegenTask.Requirement.Project != nil {
				projectName = r.CodegenTask.Requirement.Project.Name
			}
		}
		items = append(items, ReviewItem{
			ReviewID:         r.ID,
			RequirementTitle: reqTitle,
			ProjectName:      projectName,
			AIScore:          r.AIScore,
			AIStatus:         r.AIStatus,
		})
	}
	return BuildReviewListCard(items)
}

func (h *CommandHandler) handleCodegen(user *model.User, args []string) map[string]interface{} {
	if len(args) == 0 {
		return BuildCard(ColorRed, "参数错误", []cardField{
			{Key: "用法", Value: "/codegen <需求ID>"},
		}, nil)
	}

	reqID, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return BuildCard(ColorRed, "参数错误", []cardField{
			{Key: "原因", Value: "需求ID必须是数字"},
		}, nil)
	}

	req, err := h.reqSvc.GetByID(uint(reqID))
	if err != nil {
		return BuildCard(ColorRed, "错误", []cardField{
			{Key: "原因", Value: "需求不存在"},
		}, nil)
	}

	// Validate status
	if req.Status != "draft" && req.Status != "rejected" && req.Status != "generated" {
		return BuildCard(ColorRed, "状态不允许", []cardField{
			{Key: "需求", Value: fmt.Sprintf("#%d %s", req.ID, req.Title)},
			{Key: "当前状态", Value: req.Status},
			{Key: "原因", Value: "只有 draft、rejected、generated 状态的需求可以触发代码生成"},
		}, nil)
	}

	// Must have repository
	if req.RepositoryID == nil {
		return BuildCard(ColorRed, "缺少仓库", []cardField{
			{Key: "需求", Value: fmt.Sprintf("#%d %s", req.ID, req.Title)},
			{Key: "原因", Value: "需求未关联代码仓库，请先在 Web 端设置"},
		}, nil)
	}

	// Must have assignee
	if req.AssigneeID == nil {
		return BuildCard(ColorRed, "缺少指派人", []cardField{
			{Key: "需求", Value: fmt.Sprintf("#%d %s", req.ID, req.Title)},
			{Key: "原因", Value: "需求未指派人员，请先在 Web 端设置"},
		}, nil)
	}

	// No running task
	if h.reqSvc.HasRunningTask(req.ID) {
		return BuildCard(ColorYellow, "任务运行中", []cardField{
			{Key: "需求", Value: fmt.Sprintf("#%d %s", req.ID, req.Title)},
			{Key: "原因", Value: "该需求已有正在运行的代码生成任务，请等待完成"},
		}, nil)
	}

	// Get repository
	repo, err := h.repoSvc.GetByID(*req.RepositoryID)
	if err != nil {
		return BuildCard(ColorRed, "错误", []cardField{
			{Key: "原因", Value: "关联的代码仓库不存在"},
		}, nil)
	}

	// Trigger generation
	task, queuePos, err := h.codegenSvc.TriggerGeneration(req, repo, "", "", user.ID)
	if err != nil {
		log.Printf("[bot] codegen trigger failed: %v", err)
		return BuildCard(ColorRed, "代码生成启动失败", []cardField{
			{Key: "需求", Value: fmt.Sprintf("#%d %s", req.ID, req.Title)},
			{Key: "错误", Value: err.Error()},
		}, nil)
	}

	return BuildCodegenTriggeredCard(req.Title, task.ID, queuePos)
}
