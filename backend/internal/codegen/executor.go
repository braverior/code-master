package codegen

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/codeMaster/backend/internal/gitops"
	"github.com/codeMaster/backend/internal/model"
	"github.com/codeMaster/backend/internal/sse"
	"github.com/codeMaster/backend/pkg/encrypt"
	"github.com/codeMaster/backend/pkg/feishu"
	"gorm.io/gorm"
)

type Executor struct {
	db           *gorm.DB
	hub          *sse.Hub
	aesKey       string
	maxTurns     int
	timeoutMin   int
	workDir      string
	useLocalGit  bool
	task         *model.CodegenTask
	requirement  *model.Requirement
	repo         *model.Repository
	extraContext string
	docClient    *feishu.DocClient
	apiKey       string
	baseURL      string
	modelName    string // user's preferred model (e.g. "claude-sonnet-4-20250514")
	gitToken     string // user's personal git token (plaintext); takes priority over repo.AccessToken
	eventID      atomic.Int64
	pid          atomic.Int32
	cancelled    atomic.Bool
}

type ExecutorConfig struct {
	DB           *gorm.DB
	Hub          *sse.Hub
	AESKey       string
	MaxTurns     int
	TimeoutMin   int
	WorkDir      string
	UseLocalGit  bool
	Task         *model.CodegenTask
	Requirement  *model.Requirement
	Repo         *model.Repository
	ExtraContext string
	DocClient    *feishu.DocClient
	APIKey       string
	BaseURL      string
	ModelName    string // user's preferred model
	GitToken     string // user's personal git token (plaintext)
}

func NewExecutor(cfg ExecutorConfig) *Executor {
	return &Executor{
		db:           cfg.DB,
		hub:          cfg.Hub,
		aesKey:       cfg.AESKey,
		maxTurns:     cfg.MaxTurns,
		timeoutMin:   cfg.TimeoutMin,
		workDir:      cfg.WorkDir,
		useLocalGit:  cfg.UseLocalGit,
		task:         cfg.Task,
		requirement:  cfg.Requirement,
		repo:         cfg.Repo,
		extraContext: cfg.ExtraContext,
		docClient:    cfg.DocClient,
		apiKey:       cfg.APIKey,
		baseURL:      cfg.BaseURL,
		modelName:    cfg.ModelName,
		gitToken:     cfg.GitToken,
	}
}

func (e *Executor) Run(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(e.timeoutMin)*time.Minute)
	defer cancel()

	// Phase 1: Clone
	e.updateStatus("cloning")
	e.broadcastStatus("cloning", "正在克隆仓库...")

	workDir := filepath.Join(e.workDir, "codegen", strconv.FormatUint(uint64(e.task.ID), 10))
	defer os.RemoveAll(workDir)

	// Resolve git token: prefer user's personal token, fall back to repo's encrypted token
	token := e.gitToken
	if token == "" {
		var err error
		token, err = encrypt.AESDecrypt(e.aesKey, e.repo.AccessToken)
		if err != nil {
			e.broadcastLog("error", "clone", "解密 access token 失败", map[string]interface{}{"error": err.Error()})
			return e.fail("解密 access token 失败: " + err.Error())
		}
		log.Printf("[executor] using repo AccessToken (len=%d)", len(token))
	} else {
		log.Printf("[executor] using user's personal GitToken (len=%d)", len(token))
	}

	e.broadcastLog("info", "clone", "开始克隆仓库", map[string]interface{}{
		"git_url":  e.repo.GitURL,
		"branch":   e.task.SourceBranch,
		"work_dir": workDir,
	})

	// Always clone from source branch first
	if err := gitops.Clone(ctx, e.repo.GitURL, token, e.task.SourceBranch, workDir); err != nil {
		e.broadcastLog("error", "clone", "克隆仓库失败", map[string]interface{}{"error": err.Error()})
		return e.fail("clone 失败: " + err.Error())
	}
	e.broadcastLog("info", "clone", "仓库克隆完成", nil)

	if err := gitops.ConfigUser(ctx, workDir); err != nil {
		return e.fail("配置 git user 失败: " + err.Error())
	}

	// Try to fetch and checkout existing target branch from remote (iterative development)
	if err := gitops.FetchAndCheckout(ctx, workDir, e.repo.GitURL, token, e.task.TargetBranch); err != nil {
		// Target branch doesn't exist on remote — create new local branch
		e.broadcastLog("info", "clone", "远程分支不存在，创建新分支", map[string]interface{}{
			"branch": e.task.TargetBranch,
		})
		if err := gitops.CreateBranch(ctx, workDir, e.task.TargetBranch); err != nil {
			return e.fail("创建分支失败: " + err.Error())
		}
	} else {
		e.broadcastLog("info", "clone", "已切换到远程已有分支，基于上次结果继续开发", map[string]interface{}{
			"branch": e.task.TargetBranch,
		})
	}

	// Phase 2: Fetch latest doc content + Build prompt
	docContent := e.fetchDocContent()

	var analysisResult *model.AnalysisResult
	if e.repo.AnalysisResult.Data != nil {
		analysisResult = e.repo.AnalysisResult.Data
	}
	prompt := BuildPrompt(PromptInput{
		RepoAnalysis: analysisResult,
		Requirement:  e.requirement,
		ExtraContext: e.extraContext,
		DocContent:   docContent,
	})
	e.db.Model(e.task).Update("prompt", prompt)

	// Phase 3: Execute Claude Code
	e.updateStatus("running")
	now := time.Now()
	e.db.Model(e.task).Update("started_at", &now)

	args := []string{
		"-p", prompt,
		"--output-format", "stream-json",
		"--verbose",
		"--allowedTools", "Read,Write,Edit,Glob,Grep,Bash",
		"--max-turns", strconv.Itoa(e.maxTurns),
	}
	if e.modelName != "" {
		args = append(args, "--model", e.modelName)
	}
	e.broadcastLog("info", "claude", "Claude Code 启动参数", map[string]interface{}{
		"command":     "claude",
		"args":        args,
		"work_dir":    workDir,
		"timeout_min": e.timeoutMin,
	})

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("CLAUDE_CODE_MAX_TIMEOUT=%d", e.timeoutMin*60*1000),
	)
	if e.apiKey != "" {
		cmd.Env = append(cmd.Env, "ANTHROPIC_API_KEY="+e.apiKey)
	}
	if e.baseURL != "" {
		cmd.Env = append(cmd.Env, "ANTHROPIC_BASE_URL="+e.baseURL)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		e.broadcastLog("error", "claude", "获取 stdout 失败", map[string]interface{}{"error": err.Error()})
		return e.fail("获取 stdout 失败: " + err.Error())
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		e.broadcastLog("error", "claude", "获取 stderr 失败", map[string]interface{}{"error": err.Error()})
		return e.fail("获取 stderr 失败: " + err.Error())
	}

	if err := cmd.Start(); err != nil {
		e.broadcastLog("error", "claude", "启动 Claude Code 失败", map[string]interface{}{"error": err.Error()})
		return e.fail("启动 Claude Code 失败: " + err.Error())
	}

	e.pid.Store(int32(cmd.Process.Pid))
	e.broadcastLog("info", "claude", "Claude Code 进程已启动", map[string]interface{}{
		"pid": cmd.Process.Pid,
	})
	e.broadcastStatus("running", "Claude Code 已启动，正在分析项目...")

	// Capture stderr in background
	var stderrBuf bytes.Buffer
	var stderrDone sync.WaitGroup
	stderrDone.Add(1)
	go func() {
		defer stderrDone.Done()
		io.Copy(&stderrBuf, stderr)
	}()

	// Phase 4: Stream reading
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	filesWritten := 0
	filesRead := 0
	filesEdited := 0
	turnsUsed := 0
	var costUSD float64
	var lastLines []string // keep last N stdout lines for debugging

	for scanner.Scan() {
		if e.cancelled.Load() {
			break
		}

		line := scanner.Text()

		// Keep last 20 raw lines for debugging on failure
		lastLines = append(lastLines, line)
		if len(lastLines) > 20 {
			lastLines = lastLines[1:]
		}

		event, err := ParseStreamJSON(line)
		if err != nil || event == nil {
			continue
		}

		sseData := event.ToSSEData()
		id := e.eventID.Add(1)
		e.hub.Broadcast(int64(e.task.ID), sse.Event{
			ID:   id,
			Type: "output",
			Data: sseData,
		})

		switch event.Type {
		case "tool_use":
			turnsUsed++
			switch event.ToolName {
			case "Write":
				filesWritten++
			case "Read", "Glob", "Grep":
				filesRead++
			case "Edit":
				filesEdited++
			}
			pid := e.eventID.Add(1)
			e.hub.Broadcast(int64(e.task.ID), sse.Event{
				ID:   pid,
				Type: "progress",
				Data: map[string]interface{}{
					"files_read":     filesRead,
					"files_written":  filesWritten,
					"files_edited":   filesEdited,
					"turns_used":     turnsUsed,
					"max_turns":      e.maxTurns,
					"current_action": fmt.Sprintf("%s %s", event.ToolName, event.FilePath),
				},
			})
		case "result":
			costUSD = event.CostUSD
		}
	}

	// Wait for stderr goroutine to finish before calling cmd.Wait
	stderrDone.Wait()

	if err := cmd.Wait(); err != nil {
		if e.cancelled.Load() {
			return nil
		}

		stderrStr := strings.TrimSpace(stderrBuf.String())

		// Build detailed error log
		detail := map[string]interface{}{
			"exit_error": err.Error(),
		}
		if stderrStr != "" {
			detail["stderr"] = truncateStr(stderrStr, 4000)
		}
		if len(lastLines) > 0 {
			detail["last_stdout_lines"] = lastLines
		}

		// Determine a human-readable failure reason from stderr
		failReason := "Claude Code 执行失败: " + err.Error()
		if stderrStr != "" {
			e.broadcastLog("error", "claude", "Claude Code stderr 输出", map[string]interface{}{
				"stderr": truncateStr(stderrStr, 4000),
			})
			// Use the first meaningful line of stderr as the failure summary
			if firstLine := firstNonEmptyLine(stderrStr); firstLine != "" {
				failReason = "Claude Code 执行失败: " + truncateStr(firstLine, 200)
			}
		}

		e.broadcastLog("error", "claude", "Claude Code 执行失败", detail)
		return e.fail(failReason)
	}

	e.broadcastLog("info", "claude", "Claude Code 执行完成", map[string]interface{}{
		"cost_usd": costUSD,
	})

	// Phase 5: Add, commit, collect diff, and push
	commitMsg := fmt.Sprintf("feat: %s\n\nGenerated by CodeMaster (task #%d)", e.requirement.Title, e.task.ID)
	commitSHA, hasChanges, err := gitops.AddAndCommit(ctx, workDir, commitMsg)
	if err != nil {
		e.broadcastLog("error", "push", "提交代码失败", map[string]interface{}{"error": err.Error()})
		return e.fail("git commit 失败: " + err.Error())
	}
	if !hasChanges {
		e.broadcastLog("info", "push", "Claude Code 未产生任何代码变更，视为完成", nil)

		// No changes — mark as completed with zero diff
		completedAt := time.Now()
		e.db.Model(e.task).Updates(map[string]interface{}{
			"status":          "completed",
			"completed_at":    &completedAt,
			"claude_cost_usd": costUSD,
		})
		e.db.Model(&model.Requirement{}).Where("id = ?", e.requirement.ID).Update("status", "generated")

		id := e.eventID.Add(1)
		e.hub.Broadcast(int64(e.task.ID), sse.Event{ID: id, Type: "status", Data: map[string]interface{}{
			"status":        "completed",
			"files_changed": 0,
			"additions":     0,
			"deletions":     0,
		}})
		doneID := e.eventID.Add(1)
		e.hub.Broadcast(int64(e.task.ID), sse.Event{ID: doneID, Type: "done", Data: map[string]interface{}{
			"task_id": e.task.ID,
			"status":  "completed",
		}})
		e.hub.SetExpire(int64(e.task.ID), 24*time.Hour)
		return nil
	}
	e.broadcastLog("info", "push", "代码已提交", map[string]interface{}{"message": commitMsg})

	diffStat, _ := gitops.GetDiffStat(ctx, workDir, e.task.SourceBranch, e.task.TargetBranch)
	diffFiles, _ := gitops.GetDiffFiles(ctx, workDir, e.task.SourceBranch, e.task.TargetBranch)
	for i := range diffFiles {
		if content, err := gitops.GetDiffContent(ctx, workDir, e.task.SourceBranch, e.task.TargetBranch, diffFiles[i].Path); err == nil {
			diffFiles[i].Diff = content
		}
	}
	if diffStat != nil {
		diffStat.Files = diffFiles
	}

	e.broadcastLog("info", "push", "正在推送代码到远程仓库", map[string]interface{}{
		"branch": e.task.TargetBranch,
	})

	if err := gitops.Push(ctx, workDir, e.task.TargetBranch, e.repo.GitURL, token, e.useLocalGit); err != nil {
		e.broadcastLog("error", "push", "代码推送失败", map[string]interface{}{"error": err.Error()})
		return e.fail("push 失败: " + err.Error())
	}

	e.broadcastLog("info", "push", "代码推送完成", nil)

	// Phase 6: Complete
	completedAt := time.Now()
	updates := map[string]interface{}{
		"status":         "completed",
		"completed_at":   &completedAt,
		"claude_cost_usd": costUSD,
		"commit_sha":     commitSHA,
	}
	if diffStat != nil {
		updates["diff_stat"] = model.JSONDiffStat{Data: diffStat}
	}
	e.db.Model(e.task).Updates(updates)
	e.db.Model(&model.Requirement{}).Where("id = ?", e.requirement.ID).Update("status", "generated")

	id := e.eventID.Add(1)
	statusData := map[string]interface{}{
		"status": "completed",
	}
	if diffStat != nil {
		statusData["files_changed"] = diffStat.FilesChanged
		statusData["additions"] = diffStat.Additions
		statusData["deletions"] = diffStat.Deletions
	}
	e.hub.Broadcast(int64(e.task.ID), sse.Event{ID: id, Type: "status", Data: statusData})

	doneID := e.eventID.Add(1)
	e.hub.Broadcast(int64(e.task.ID), sse.Event{ID: doneID, Type: "done", Data: map[string]interface{}{
		"task_id": e.task.ID,
		"status":  "completed",
	}})

	e.hub.SetExpire(int64(e.task.ID), 24*time.Hour)
	return nil
}

func (e *Executor) Cancel() error {
	e.cancelled.Store(true)
	pid := e.pid.Load()
	if pid == 0 {
		return fmt.Errorf("task not running")
	}
	process, err := os.FindProcess(int(pid))
	if err != nil {
		return err
	}
	process.Signal(os.Interrupt)

	time.AfterFunc(3*time.Second, func() {
		process.Kill()
	})

	e.updateStatus("cancelled")
	completedAt := time.Now()
	e.db.Model(e.task).Update("completed_at", &completedAt)
	e.db.Model(&model.Requirement{}).Where("id = ?", e.requirement.ID).Update("status", "draft")

	id := e.eventID.Add(1)
	e.hub.Broadcast(int64(e.task.ID), sse.Event{ID: id, Type: "done", Data: map[string]interface{}{
		"task_id": e.task.ID,
		"status":  "cancelled",
	}})

	e.hub.SetExpire(int64(e.task.ID), 24*time.Hour)
	return nil
}

func (e *Executor) GetPID() int32 {
	return e.pid.Load()
}

func (e *Executor) updateStatus(status string) {
	e.db.Model(e.task).Update("status", status)
}

func (e *Executor) broadcastStatus(status, message string) {
	id := e.eventID.Add(1)
	data := map[string]interface{}{"status": status, "message": message}
	if status == "running" {
		if pid := e.pid.Load(); pid > 0 {
			data["pid"] = pid
		}
	}
	e.hub.Broadcast(int64(e.task.ID), sse.Event{
		ID:   id,
		Type: "status",
		Data: data,
	})
}

func (e *Executor) broadcastLog(level, phase, message string, detail map[string]interface{}) {
	id := e.eventID.Add(1)
	data := map[string]interface{}{
		"level":   level,
		"phase":   phase,
		"message": message,
	}
	if detail != nil {
		data["detail"] = detail
	}
	e.hub.Broadcast(int64(e.task.ID), sse.Event{
		ID:   id,
		Type: "log",
		Data: data,
	})
}

func (e *Executor) fail(errMsg string) error {
	e.db.Model(e.task).Updates(map[string]interface{}{
		"status":        "failed",
		"error_message": errMsg,
		"completed_at":  time.Now(),
	})
	e.db.Model(&model.Requirement{}).Where("id = ?", e.requirement.ID).Update("status", "draft")

	id := e.eventID.Add(1)
	e.hub.Broadcast(int64(e.task.ID), sse.Event{
		ID:   id,
		Type: "task_error",
		Data: map[string]interface{}{"message": errMsg},
	})
	doneID := e.eventID.Add(1)
	e.hub.Broadcast(int64(e.task.ID), sse.Event{
		ID:   doneID,
		Type: "done",
		Data: map[string]interface{}{"task_id": e.task.ID, "status": "failed"},
	})
	e.hub.SetExpire(int64(e.task.ID), 24*time.Hour)
	return fmt.Errorf(errMsg)
}

// fetchDocContent fetches the latest content from all linked Feishu documents.
// Returns combined content from all docs, or falls back to stored DocContent.
func (e *Executor) fetchDocContent() string {
	if e.docClient == nil || len(e.requirement.DocLinks) == 0 {
		return e.requirement.DocContent
	}

	var parts []string
	for _, link := range e.requirement.DocLinks {
		token := feishu.ExtractDocToken(link.URL)
		if token == "" {
			continue
		}
		content, err := e.docClient.GetDocContent(token)
		if err != nil {
			log.Printf("[codegen] fetch doc %q failed: %v", link.Title, err)
			continue
		}
		if content == "" {
			continue
		}
		title := link.Title
		if title == "" {
			title = link.URL
		}
		parts = append(parts, fmt.Sprintf("### %s\n\n%s", title, content))
	}

	if len(parts) == 0 {
		return e.requirement.DocContent
	}

	combined := strings.Join(parts, "\n\n---\n\n")

	// Save to requirement for record keeping
	e.db.Model(e.requirement).Update("doc_content", combined)

	e.broadcastLog("info", "docs", fmt.Sprintf("已获取 %d 个关联文档内容", len(parts)), nil)
	return combined
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...(truncated)"
}

func firstNonEmptyLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}
