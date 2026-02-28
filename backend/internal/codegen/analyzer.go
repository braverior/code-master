package codegen

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/codeMaster/backend/internal/gitops"
	"github.com/codeMaster/backend/internal/model"
	"github.com/codeMaster/backend/pkg/claude"
	"github.com/codeMaster/backend/pkg/encrypt"
	"gorm.io/gorm"
)

type Analyzer struct {
	db      *gorm.DB
	aesKey  string
	workDir string
}

func NewAnalyzer(db *gorm.DB, aesKey, workDir string) *Analyzer {
	return &Analyzer{db: db, aesKey: aesKey, workDir: workDir}
}

func (a *Analyzer) setFailed(repo *model.Repository, errMsg string) {
	a.db.Model(repo).Updates(map[string]interface{}{
		"analysis_status": "failed",
		"analysis_error":  errMsg,
	})
}

func (a *Analyzer) Analyze(ctx context.Context, repo *model.Repository, gitToken, apiKey, baseURL, modelName string) error {
	a.db.Model(repo).Updates(map[string]interface{}{
		"analysis_status": "running",
		"analysis_error":  "",
	})

	workDir := filepath.Join(a.workDir, "analysis", strconv.FormatUint(uint64(repo.ID), 10))
	defer os.RemoveAll(workDir)

	// Resolve token: prefer user's personal token, fall back to repo's stored token
	token := gitToken
	if token == "" {
		var err error
		token, err = encrypt.AESDecrypt(a.aesKey, repo.AccessToken)
		if err != nil {
			a.setFailed(repo, "无可用的 Git Token，请在个人设置中配置")
			return fmt.Errorf("no git token available: %w", err)
		}
	}

	if err := gitops.Clone(ctx, repo.GitURL, token, repo.DefaultBranch, workDir); err != nil {
		a.setFailed(repo, "克隆仓库失败: "+err.Error())
		return fmt.Errorf("clone: %w", err)
	}

	prompt := `分析这个代码仓库的结构和功能。输出严格 JSON 格式:
{
  "modules": [{"path": "", "description": "", "files_count": 0}],
  "tech_stack": [],
  "entry_points": [],
  "directory_structure": "",
  "code_style": {"naming": "", "error_handling": "", "test_framework": ""}
}
只输出 JSON，不要任何其他内容。`

	analyzeCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	args := []string{
		"-p", prompt,
		"--output-format", "json",
		"--allowedTools", "Read,Glob,Grep",
	}
	if modelName != "" {
		args = append(args, "--model", modelName)
	}

	cmd := exec.CommandContext(analyzeCtx, "claude", args...)
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
		a.setFailed(repo, fmt.Sprintf("Claude 分析执行失败: %v\n%s", err, errDetail))
		return fmt.Errorf("claude analyze: %s: %w", errDetail, err)
	}

	// Extract JSON from Claude CLI output (handles envelope + markdown fences)
	rawJSON := claude.ExtractJSON(output)

	var result model.AnalysisResult
	if err := json.Unmarshal(rawJSON, &result); err != nil {
		a.setFailed(repo, "解析分析结果失败: "+err.Error())
		return fmt.Errorf("parse analysis result: %w", err)
	}

	now := time.Now()
	a.db.Model(repo).Updates(map[string]interface{}{
		"analysis_result": model.JSONAnalysisResult{Data: &result},
		"analysis_status": "completed",
		"analyzed_at":     &now,
	})
	return nil
}
