package feishu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type BotClient struct {
	oauth *OAuthClient
}

func NewBotClient(oauth *OAuthClient) *BotClient {
	return &BotClient{oauth: oauth}
}

// SendInteractiveMessage sends an interactive card to a user by open_id.
func (c *BotClient) SendInteractiveMessage(openID string, card map[string]interface{}) error {
	cardJSON, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("marshal card: %w", err)
	}
	return c.sendMessage(openID, "interactive", string(cardJSON))
}

// SendTextMessage sends a plain text message to a user by open_id.
func (c *BotClient) SendTextMessage(openID, text string) error {
	content, _ := json.Marshal(map[string]string{"text": text})
	return c.sendMessage(openID, "text", string(content))
}

// ReplyInteractiveMessage replies with an interactive card to a specific message.
func (c *BotClient) ReplyInteractiveMessage(messageID string, card map[string]interface{}) error {
	cardJSON, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("marshal card: %w", err)
	}
	return c.replyMessage(messageID, "interactive", string(cardJSON))
}

// ReplyTextMessage replies with plain text to a specific message.
func (c *BotClient) ReplyTextMessage(messageID, text string) error {
	content, _ := json.Marshal(map[string]string{"text": text})
	return c.replyMessage(messageID, "text", string(content))
}

func (c *BotClient) sendMessage(receiveID, msgType, content string) error {
	token, err := c.oauth.GetAppAccessToken()
	if err != nil {
		return fmt.Errorf("get app token: %w", err)
	}

	body := map[string]string{
		"receive_id": receiveID,
		"msg_type":   msgType,
		"content":    content,
	}
	bodyBytes, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST",
		"https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=open_id",
		bytes.NewReader(bodyBytes),
	)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	var result feishuResp
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return fmt.Errorf("decode send resp: %w", err)
	}
	if result.Code != 0 {
		return fmt.Errorf("send message failed (code=%d): %s", result.Code, result.ErrMsg())
	}
	return nil
}

func (c *BotClient) replyMessage(messageID, msgType, content string) error {
	token, err := c.oauth.GetAppAccessToken()
	if err != nil {
		return fmt.Errorf("get app token: %w", err)
	}

	body := map[string]string{
		"msg_type": msgType,
		"content":  content,
	}
	bodyBytes, _ := json.Marshal(body)

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages/%s/reply", messageID)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("reply message: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	var result feishuResp
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return fmt.Errorf("decode reply resp: %w", err)
	}
	if result.Code != 0 {
		return fmt.Errorf("reply message failed (code=%d): %s", result.Code, result.ErrMsg())
	}
	return nil
}
