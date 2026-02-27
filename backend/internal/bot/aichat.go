package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/codeMaster/backend/internal/config"
)

// ChatMessage represents a single message in a conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIChatClient handles AI chat via OpenAI-compatible API.
type AIChatClient struct {
	baseURL    string
	apiKey     string
	model      string
	maxHistory int

	mu            sync.RWMutex
	conversations map[string][]ChatMessage // key: open_id
}

// NewAIChatClient creates a new AI chat client from config.
func NewAIChatClient(cfg config.AIChatConfig) *AIChatClient {
	maxHistory := cfg.MaxHistory
	if maxHistory <= 0 {
		maxHistory = 20
	}
	return &AIChatClient{
		baseURL:       cfg.BaseURL,
		apiKey:        cfg.APIKey,
		model:         cfg.Model,
		maxHistory:    maxHistory,
		conversations: make(map[string][]ChatMessage),
	}
}

const systemPrompt = `你是 CodeMaster Bot，一个智能编程助手。你可以回答关于软件开发、代码架构、最佳实践等方面的问题。请用中文回复。回答应简洁、准确、有帮助。`

const intentPrompt = `你是一个意图识别器。根据用户消息判断是否属于以下业务查询之一，如果是则返回对应命令，如果不是则返回 NONE。

可用命令：
- /projects — 查看我的项目列表（用户问"我的项目""有哪些项目"等）
- /myreqs — 查看我的需求/任务列表（用户问"我的任务""我有哪些需求""今天有什么任务"等，不指定具体项目时使用）
- /reqs <项目ID> — 查看某个项目的需求列表（用户提到具体项目并问需求/任务）
- /status <需求ID> — 查看某个需求的详细状态（用户问某个需求/任务的状态、进度、情况）
- /reviews — 查看我的待审查列表（用户问待审查、待review、需要审核的内容）
- /codegen <需求ID> — 触发代码生成（用户说"完成第X个需求""帮我生成第X个需求的代码""执行需求X""开始生成需求X""帮我完成需求X"等）
- /help — 查看帮助（用户问功能、怎么用）

规则：
1. 只输出命令本身（如 /projects 或 /status 42），不要输出其他任何文字
2. 如果用户问"我的需求""我的任务""有哪些任务"等但没有指定具体项目ID，返回 /myreqs
3. 如果用户问"我的项目""有哪些项目"，返回 /projects
4. 如果用户说"完成第X个需求""帮我完成需求X""开始生成需求X""执行需求X"等，返回 /codegen X
5. 如果不属于以上任何业务查询（比如闲聊、编程问题、知识问答），返回 NONE
6. ID 从用户消息中提取数字`

// Chat sends a user message and returns the AI response.
func (c *AIChatClient) Chat(openID, userMessage string) (string, error) {
	c.mu.Lock()
	history := c.conversations[openID]
	history = append(history, ChatMessage{Role: "user", Content: userMessage})

	// Build messages with system prompt
	messages := make([]ChatMessage, 0, len(history)+1)
	messages = append(messages, ChatMessage{Role: "system", Content: systemPrompt})
	messages = append(messages, history...)
	c.mu.Unlock()

	// Call API
	reply, err := c.callAPI(messages)
	if err != nil {
		return "", err
	}

	// Update history
	c.mu.Lock()
	history = append(history, ChatMessage{Role: "assistant", Content: reply})
	// Trim if exceeds max
	if len(history) > c.maxHistory*2 {
		history = history[len(history)-c.maxHistory*2:]
	}
	c.conversations[openID] = history
	c.mu.Unlock()

	return reply, nil
}

// ClearHistory clears conversation history for a user.
func (c *AIChatClient) ClearHistory(openID string) {
	c.mu.Lock()
	delete(c.conversations, openID)
	c.mu.Unlock()
}

// ClassifyIntent uses AI to determine if the user message maps to a business command.
// Returns the command string (e.g. "/projects", "/status 42") or "" if not a business query.
func (c *AIChatClient) ClassifyIntent(userMessage string) string {
	messages := []ChatMessage{
		{Role: "system", Content: intentPrompt},
		{Role: "user", Content: userMessage},
	}
	reply, err := c.callAPI(messages)
	if err != nil {
		return ""
	}
	reply = strings.TrimSpace(reply)
	if reply == "NONE" || reply == "" {
		return ""
	}
	// Ensure it starts with /
	if !strings.HasPrefix(reply, "/") {
		return ""
	}
	return reply
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *AIChatClient) callAPI(messages []ChatMessage) (string, error) {
	reqBody := chatRequest{
		Model:    c.model,
		Messages: messages,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	url := c.baseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call AI API: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)

	var result chatResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("decode AI response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("AI API error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("AI API returned no choices")
	}

	return result.Choices[0].Message.Content, nil
}
