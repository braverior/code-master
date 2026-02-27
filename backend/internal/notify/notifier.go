package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/codeMaster/backend/pkg/feishu"
)

// Notifier defines the interface for sending notifications.
type Notifier interface {
	NotifyRequirementCreated(ctx context.Context, e RequirementCreatedEvent) error
	NotifyRequirementAssigned(ctx context.Context, e RequirementAssignedEvent) error
	NotifyCodegenCompleted(ctx context.Context, e CodegenCompletedEvent) error
	NotifyCodegenFailed(ctx context.Context, e CodegenFailedEvent) error
	NotifyAIReviewCompleted(ctx context.Context, e AIReviewCompletedEvent) error
	NotifyHumanReviewSubmitted(ctx context.Context, e HumanReviewSubmittedEvent) error
}

// NoopNotifier is a no-op implementation used when bot is disabled.
type NoopNotifier struct{}

func (NoopNotifier) NotifyRequirementCreated(context.Context, RequirementCreatedEvent) error   { return nil }
func (NoopNotifier) NotifyRequirementAssigned(context.Context, RequirementAssignedEvent) error { return nil }
func (NoopNotifier) NotifyCodegenCompleted(context.Context, CodegenCompletedEvent) error       { return nil }
func (NoopNotifier) NotifyCodegenFailed(context.Context, CodegenFailedEvent) error             { return nil }
func (NoopNotifier) NotifyAIReviewCompleted(context.Context, AIReviewCompletedEvent) error     { return nil }
func (NoopNotifier) NotifyHumanReviewSubmitted(context.Context, HumanReviewSubmittedEvent) error { return nil }

// FeishuNotifier sends interactive card notifications via Feishu bot.
type FeishuNotifier struct {
	botClient *feishu.BotClient
}

func NewFeishuNotifier(botClient *feishu.BotClient) *FeishuNotifier {
	return &FeishuNotifier{botClient: botClient}
}

func (n *FeishuNotifier) NotifyRequirementCreated(_ context.Context, e RequirementCreatedEvent) error {
	if e.AssigneeOpenID == "" {
		return nil
	}
	card := buildCard("blue", "📋 新需求指派", []cardField{
		{Key: "项目", Value: e.ProjectName},
		{Key: "需求", Value: e.Title},
		{Key: "优先级", Value: e.Priority},
		{Key: "创建人", Value: e.CreatorName},
	}, nil)
	return n.send(e.AssigneeOpenID, card)
}

func (n *FeishuNotifier) NotifyRequirementAssigned(_ context.Context, e RequirementAssignedEvent) error {
	if e.AssigneeOpenID == "" {
		return nil
	}
	card := buildCard("blue", "📋 需求已指派给你", []cardField{
		{Key: "项目", Value: e.ProjectName},
		{Key: "需求", Value: e.Title},
		{Key: "优先级", Value: e.Priority},
		{Key: "指派人", Value: e.AssignerName},
	}, nil)
	return n.send(e.AssigneeOpenID, card)
}

func (n *FeishuNotifier) NotifyCodegenCompleted(_ context.Context, e CodegenCompletedEvent) error {
	fields := []cardField{
		{Key: "项目", Value: e.ProjectName},
		{Key: "需求", Value: e.Title},
		{Key: "任务ID", Value: fmt.Sprintf("%d", e.TaskID)},
		{Key: "变更统计", Value: fmt.Sprintf("%d 文件, +%d -%d", e.FilesChanged, e.Additions, e.Deletions)},
	}
	card := buildCard("green", "✅ 代码生成完成", fields, nil)

	var firstErr error
	for _, openID := range uniqueNonEmpty(e.CreatorOpenID, e.AssigneeOpenID) {
		if err := n.send(openID, card); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (n *FeishuNotifier) NotifyCodegenFailed(_ context.Context, e CodegenFailedEvent) error {
	fields := []cardField{
		{Key: "项目", Value: e.ProjectName},
		{Key: "需求", Value: e.Title},
		{Key: "任务ID", Value: fmt.Sprintf("%d", e.TaskID)},
		{Key: "错误", Value: truncate(e.ErrorMessage, 200)},
	}
	card := buildCard("red", "❌ 代码生成失败", fields, nil)

	var firstErr error
	for _, openID := range uniqueNonEmpty(e.CreatorOpenID, e.AssigneeOpenID) {
		if err := n.send(openID, card); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (n *FeishuNotifier) NotifyAIReviewCompleted(_ context.Context, e AIReviewCompletedEvent) error {
	color := "yellow"
	if e.AIStatus == "passed" {
		color = "green"
	} else if e.AIStatus == "failed" {
		color = "red"
	}

	scoreText := "N/A"
	if e.AIScore != nil {
		scoreText = fmt.Sprintf("%d/100", *e.AIScore)
	}

	fields := []cardField{
		{Key: "项目", Value: e.ProjectName},
		{Key: "需求", Value: e.Title},
		{Key: "AI评分", Value: scoreText},
		{Key: "AI状态", Value: e.AIStatus},
	}
	card := buildCard(color, "🤖 AI Review 完成", fields, nil)

	var firstErr error
	for _, openID := range uniqueNonEmpty(e.CreatorOpenID, e.AssigneeOpenID) {
		if err := n.send(openID, card); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (n *FeishuNotifier) NotifyHumanReviewSubmitted(_ context.Context, e HumanReviewSubmittedEvent) error {
	color := "red"
	title := "👎 人工审查未通过"
	if e.Status == "approved" {
		color = "green"
		title = "👍 人工审查通过"
	}

	fields := []cardField{
		{Key: "项目", Value: e.ProjectName},
		{Key: "需求", Value: e.Title},
		{Key: "审查人", Value: e.ReviewerName},
		{Key: "状态", Value: e.Status},
	}
	if e.Comment != "" {
		fields = append(fields, cardField{Key: "意见", Value: truncate(e.Comment, 200)})
	}
	card := buildCard(color, title, fields, nil)

	var firstErr error
	for _, openID := range uniqueNonEmpty(e.CreatorOpenID, e.AssigneeOpenID) {
		if err := n.send(openID, card); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (n *FeishuNotifier) send(openID string, card map[string]interface{}) error {
	if err := n.botClient.SendInteractiveMessage(openID, card); err != nil {
		log.Printf("[notify] send feishu message to %s failed: %v", openID, err)
		return err
	}
	return nil
}

// --- card building helpers ---

type cardField struct {
	Key   string
	Value string
}

func buildCard(color, title string, fields []cardField, actions []map[string]interface{}) map[string]interface{} {
	elements := make([]interface{}, 0, len(fields)+1)

	// Fields as markdown
	for _, f := range fields {
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": fmt.Sprintf("**%s：**%s", f.Key, f.Value),
			},
		})
	}

	if len(actions) > 0 {
		elements = append(elements, map[string]interface{}{
			"tag":     "action",
			"actions": actions,
		})
	}

	return map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": title,
			},
			"template": color,
		},
		"elements": elements,
	}
}

func uniqueNonEmpty(ids ...string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, id := range ids {
		if id != "" && !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// ensure FeishuNotifier implements Notifier at compile time
var _ Notifier = (*FeishuNotifier)(nil)

// cardJSON is a helper for debugging; not used in production path.
func cardJSON(card map[string]interface{}) string {
	b, _ := json.MarshalIndent(card, "", "  ")
	return string(b)
}
