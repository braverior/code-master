package bot

import (
	"fmt"
)

// Card color constants
const (
	ColorBlue   = "blue"
	ColorGreen  = "green"
	ColorRed    = "red"
	ColorYellow = "yellow"
)

type cardField struct {
	Key   string
	Value string
}

type cardAction struct {
	Text string
	URL  string
}

// BuildCard builds a Feishu interactive card JSON structure.
func BuildCard(color, title string, fields []cardField, actions []cardAction) map[string]interface{} {
	elements := make([]interface{}, 0, len(fields)+2)

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
		actionList := make([]interface{}, 0, len(actions))
		for _, a := range actions {
			actionList = append(actionList, map[string]interface{}{
				"tag": "button",
				"text": map[string]interface{}{
					"tag":     "plain_text",
					"content": a.Text,
				},
				"type":      "primary",
				"multi_url": map[string]interface{}{
					"url": a.URL,
				},
			})
		}
		elements = append(elements, map[string]interface{}{
			"tag":     "action",
			"actions": actionList,
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

// BuildProjectListCard builds a card showing a list of projects.
func BuildProjectListCard(projects []ProjectItem) map[string]interface{} {
	if len(projects) == 0 {
		return BuildCard(ColorBlue, "📁 我的项目", []cardField{
			{Key: "提示", Value: "暂无项目"},
		}, nil)
	}

	fields := make([]cardField, 0, len(projects))
	for _, p := range projects {
		fields = append(fields, cardField{
			Key:   fmt.Sprintf("#%d %s", p.ID, p.Name),
			Value: fmt.Sprintf("状态: %s | 需求数: %d", p.Status, p.ReqCount),
		})
	}
	return BuildCard(ColorBlue, "📁 我的项目", fields, nil)
}

// BuildRequirementListCard builds a card showing a list of requirements.
func BuildRequirementListCard(projectName string, reqs []RequirementItem) map[string]interface{} {
	title := fmt.Sprintf("📋 需求列表 - %s", projectName)
	if len(reqs) == 0 {
		return BuildCard(ColorBlue, title, []cardField{
			{Key: "提示", Value: "暂无需求"},
		}, nil)
	}

	fields := make([]cardField, 0, len(reqs))
	for _, r := range reqs {
		assignee := "未指派"
		if r.AssigneeName != "" {
			assignee = r.AssigneeName
		}
		fields = append(fields, cardField{
			Key:   fmt.Sprintf("#%d %s", r.ID, r.Title),
			Value: fmt.Sprintf("[%s] %s | 指派: %s", r.Priority, r.Status, assignee),
		})
	}
	return BuildCard(ColorBlue, title, fields, nil)
}

// BuildMyRequirementListCard builds a card showing a user's requirements across all projects.
func BuildMyRequirementListCard(reqs []MyRequirementItem) map[string]interface{} {
	title := "📋 我的需求/任务"
	if len(reqs) == 0 {
		return BuildCard(ColorBlue, title, []cardField{
			{Key: "提示", Value: "暂无需求"},
		}, nil)
	}

	fields := make([]cardField, 0, len(reqs))
	for _, r := range reqs {
		assignee := "未指派"
		if r.AssigneeName != "" {
			assignee = r.AssigneeName
		}
		project := "未知项目"
		if r.ProjectName != "" {
			project = r.ProjectName
		}
		fields = append(fields, cardField{
			Key:   fmt.Sprintf("#%d %s", r.ID, r.Title),
			Value: fmt.Sprintf("项目: %s | [%s] %s | 指派: %s", project, r.Priority, r.Status, assignee),
		})
	}
	return BuildCard(ColorBlue, title, fields, nil)
}

// BuildStatusCard builds a detailed status card for a requirement.
func BuildStatusCard(s StatusInfo) map[string]interface{} {
	fields := []cardField{
		{Key: "需求", Value: fmt.Sprintf("#%d %s", s.RequirementID, s.Title)},
		{Key: "项目", Value: s.ProjectName},
		{Key: "状态", Value: s.Status},
		{Key: "优先级", Value: s.Priority},
	}
	if s.LatestTaskStatus != "" {
		fields = append(fields, cardField{
			Key: "最新代码生成", Value: fmt.Sprintf("任务#%d - %s", s.LatestTaskID, s.LatestTaskStatus),
		})
	}
	if s.LatestReviewAIStatus != "" {
		scoreText := "N/A"
		if s.LatestReviewAIScore != nil {
			scoreText = fmt.Sprintf("%d/100", *s.LatestReviewAIScore)
		}
		fields = append(fields, cardField{
			Key: "最新Review", Value: fmt.Sprintf("AI: %s (%s) | 人工: %s", s.LatestReviewAIStatus, scoreText, s.LatestReviewHumanStatus),
		})
	}

	color := ColorBlue
	switch s.Status {
	case "approved", "merged":
		color = ColorGreen
	case "rejected":
		color = ColorRed
	case "generating", "reviewing":
		color = ColorYellow
	}

	return BuildCard(color, "📊 需求状态", fields, nil)
}

// BuildReviewListCard builds a card showing pending reviews.
func BuildReviewListCard(reviews []ReviewItem) map[string]interface{} {
	if len(reviews) == 0 {
		return BuildCard(ColorGreen, "📝 待审查列表", []cardField{
			{Key: "提示", Value: "暂无待审查项"},
		}, nil)
	}

	fields := make([]cardField, 0, len(reviews))
	for _, r := range reviews {
		scoreText := "N/A"
		if r.AIScore != nil {
			scoreText = fmt.Sprintf("%d", *r.AIScore)
		}
		fields = append(fields, cardField{
			Key:   fmt.Sprintf("Review#%d - %s", r.ReviewID, r.RequirementTitle),
			Value: fmt.Sprintf("项目: %s | AI评分: %s | AI状态: %s", r.ProjectName, scoreText, r.AIStatus),
		})
	}
	return BuildCard(ColorYellow, "📝 待审查列表", fields, nil)
}

// BuildCodegenTriggeredCard builds a green card indicating code generation has started.
func BuildCodegenTriggeredCard(reqTitle string, taskID uint, queuePos int) map[string]interface{} {
	fields := []cardField{
		{Key: "需求", Value: reqTitle},
		{Key: "任务ID", Value: fmt.Sprintf("%d", taskID)},
		{Key: "队列位置", Value: fmt.Sprintf("%d", queuePos)},
	}
	return BuildCard(ColorGreen, "🚀 代码生成已启动", fields, nil)
}

// --- Data transfer types for card building ---

type ProjectItem struct {
	ID       uint
	Name     string
	Status   string
	ReqCount int64
}

type RequirementItem struct {
	ID           uint
	Title        string
	Status       string
	Priority     string
	AssigneeName string
}

type MyRequirementItem struct {
	ID           uint
	Title        string
	Status       string
	Priority     string
	ProjectName  string
	AssigneeName string
}

type StatusInfo struct {
	RequirementID            uint
	Title                    string
	ProjectName              string
	Status                   string
	Priority                 string
	LatestTaskID             uint
	LatestTaskStatus         string
	LatestReviewAIStatus     string
	LatestReviewAIScore      *int
	LatestReviewHumanStatus  string
}

type ReviewItem struct {
	ReviewID         uint
	RequirementTitle string
	ProjectName      string
	AIScore          *int
	AIStatus         string
}
