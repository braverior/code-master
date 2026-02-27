package claude

import (
	"encoding/json"
	"regexp"
	"strings"
)

// ExtractJSON extracts a JSON object from Claude CLI output.
// It handles two layers of wrapping:
// 1. Claude CLI --output-format json envelope: {"type":"result","result":"..."}
// 2. Markdown code fences in the result text: ```json ... ```
func ExtractJSON(output []byte) []byte {
	raw := output

	// Layer 1: unwrap Claude CLI JSON envelope
	var envelope struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(output, &envelope); err == nil && envelope.Result != "" {
		raw = []byte(envelope.Result)
	}

	// Layer 2: strip markdown code fences if present
	s := strings.TrimSpace(string(raw))
	if strings.HasPrefix(s, "```") {
		re := regexp.MustCompile("(?s)```(?:json)?\\s*\n?(.*?)\\s*```")
		if m := re.FindStringSubmatch(s); len(m) > 1 {
			s = strings.TrimSpace(m[1])
		}
	}

	return []byte(s)
}
