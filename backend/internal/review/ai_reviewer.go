package review

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/codeMaster/backend/internal/model"
	"github.com/codeMaster/backend/pkg/claude"
	"gorm.io/gorm"
)

type AIReviewer struct {
	db *gorm.DB
}

func NewAIReviewer(db *gorm.DB) *AIReviewer {
	return &AIReviewer{db: db}
}

func (r *AIReviewer) RunReview(ctx context.Context, review *model.CodeReview, workDir, diffContent, apiKey, baseURL, modelName string) error {
	r.db.Model(review).Update("ai_status", "running")

	prompt := BuildReviewPrompt(diffContent)

	reviewCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	args := []string{
		"-p", prompt,
		"--output-format", "json",
		"--allowedTools", "Read,Glob,Grep",
	}
	if modelName != "" {
		args = append(args, "--model", modelName)
	}

	cmd := exec.CommandContext(reviewCtx, "claude", args...)
	cmd.Dir = workDir
	cmd.Env = os.Environ()
	if apiKey != "" {
		cmd.Env = append(cmd.Env, "ANTHROPIC_API_KEY="+apiKey)
	}
	if baseURL != "" {
		cmd.Env = append(cmd.Env, "ANTHROPIC_BASE_URL="+baseURL)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		errDetail := string(output)
		if len(errDetail) > 500 {
			errDetail = errDetail[:500]
		}
		r.db.Model(review).Update("ai_status", "failed")
		return fmt.Errorf("claude review: %s: %w", errDetail, err)
	}

	// Extract JSON from Claude CLI output (handles envelope + markdown fences)
	rawJSON := claude.ExtractJSON(output)

	var result model.AIReviewResult
	if err := json.Unmarshal(rawJSON, &result); err != nil {
		r.db.Model(review).Update("ai_status", "failed")
		return fmt.Errorf("parse review result: %w", err)
	}

	score := calculateScore(result)
	status := "passed"
	if score < 60 {
		status = "failed"
	} else if score < 80 {
		status = "warning"
	}

	r.db.Model(review).Updates(map[string]interface{}{
		"ai_review_result": model.JSONAIReviewResult{Data: &result},
		"ai_score":         score,
		"ai_status":        status,
	})
	return nil
}

func calculateScore(result model.AIReviewResult) int {
	score := 100
	for _, issue := range result.Issues {
		switch issue.Severity {
		case "error":
			score -= 15
		case "warning":
			score -= 5
		case "info":
			score -= 1
		}
	}
	if score < 0 {
		score = 0
	}
	return score
}
