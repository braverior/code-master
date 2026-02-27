package codegen

import (
	"encoding/json"
)

type StreamEvent struct {
	Type     string          `json:"type"`
	SubType  string          `json:"subtype,omitempty"`
	Content  string          `json:"content,omitempty"`
	ToolName string          `json:"tool,omitempty"`
	ToolID   string          `json:"id,omitempty"`
	Input    json.RawMessage `json:"input,omitempty"`
	Output   string          `json:"output,omitempty"`
	ExitCode *int            `json:"exit_code,omitempty"`
	Summary  string          `json:"summary,omitempty"`
	CostUSD  float64         `json:"cost_usd,omitempty"`
	FilePath string          `json:"-"`
}

// rawStreamLine matches the actual Claude Code --output-format stream-json --verbose format.
//
// Real examples:
//
//	{"type":"system","subtype":"init","cwd":"...","session_id":"...","tools":[...]}
//	{"type":"assistant","message":{"content":[{"type":"text","text":"..."}],...},"session_id":"..."}
//	{"type":"assistant","message":{"content":[{"type":"tool_use","id":"...","name":"Read","input":{...}}],...}}
//	{"type":"user","message":{"content":[{"tool_use_id":"...","type":"tool_result","content":"..."}]},...}
//	{"type":"result","subtype":"success","total_cost_usd":0.15,"result":"...","num_turns":2}
type rawStreamLine struct {
	Type    string `json:"type"`
	SubType string `json:"subtype,omitempty"`

	// "assistant" and "user" types nest content under message.content
	Message *rawMessage `json:"message,omitempty"`

	// "result" type fields
	TotalCostUSD float64 `json:"total_cost_usd,omitempty"`
	Result       string  `json:"result,omitempty"`
	NumTurns     int     `json:"num_turns,omitempty"`
}

type rawMessage struct {
	Role    string              `json:"role,omitempty"`
	Content []rawContentElement `json:"content,omitempty"`
}

type rawContentElement struct {
	Type string `json:"type"`

	// text content
	Text string `json:"text,omitempty"`

	// tool_use content
	Name  string          `json:"name,omitempty"`
	ID    string          `json:"id,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// tool_result content (inside "user" type messages)
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
}

func ParseStreamJSON(line string) (*StreamEvent, error) {
	var raw rawStreamLine
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return nil, err
	}

	event := &StreamEvent{}

	switch raw.Type {
	case "assistant":
		if raw.Message == nil || len(raw.Message.Content) == 0 {
			// No content — skip
			return nil, nil
		}
		c := raw.Message.Content[0]
		switch c.Type {
		case "text":
			event.Type = "output"
			event.SubType = raw.SubType
			if event.SubType == "" {
				event.SubType = "text"
			}
			event.Content = c.Text
		case "tool_use":
			event.Type = "tool_use"
			event.ToolName = c.Name
			event.ToolID = c.ID
			event.Input = c.Input
			event.FilePath = extractFilePath(c.Input)
		default:
			return nil, nil
		}

	case "user":
		// Tool result: {"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"...","content":"..."}]}}
		if raw.Message == nil || len(raw.Message.Content) == 0 {
			return nil, nil
		}
		c := raw.Message.Content[0]
		if c.Type != "tool_result" {
			return nil, nil
		}
		event.Type = "tool_result"
		event.ToolID = c.ToolUseID
		if c.Content != nil {
			// content can be a string or a complex object; store as string
			var s string
			if err := json.Unmarshal(c.Content, &s); err != nil {
				// Not a plain string — store the raw JSON
				event.Output = string(c.Content)
			} else {
				event.Output = s
			}
		}

	case "result":
		event.Type = "result"
		event.CostUSD = raw.TotalCostUSD
		event.Content = raw.Result

	default:
		// system, etc. — skip
		return nil, nil
	}

	return event, nil
}

func extractFilePath(input json.RawMessage) string {
	if input == nil {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal(input, &m); err != nil {
		return ""
	}
	if fp, ok := m["file_path"].(string); ok {
		return fp
	}
	if fp, ok := m["command"].(string); ok {
		return fp
	}
	return ""
}

func (e *StreamEvent) ToSSEData() map[string]interface{} {
	data := map[string]interface{}{
		"type": e.Type,
	}
	switch e.Type {
	case "output":
		data["type"] = e.SubType
		data["content"] = e.Content
	case "tool_use":
		data["type"] = "tool_use"
		data["tool"] = e.ToolName
		data["id"] = e.ToolID
		if e.Input != nil {
			var inputMap map[string]interface{}
			json.Unmarshal(e.Input, &inputMap)
			data["input"] = inputMap
		}
	case "tool_result":
		data["type"] = "tool_result"
		data["id"] = e.ToolID
		data["output"] = e.Output
		if e.ExitCode != nil {
			data["exit_code"] = *e.ExitCode
		}
	case "result":
		data["type"] = "result"
		data["cost_usd"] = e.CostUSD
	}
	return data
}
