package feishu

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type DocClient struct {
	AppID     string
	AppSecret string
}

func NewDocClient(appID, appSecret string) *DocClient {
	return &DocClient{AppID: appID, AppSecret: appSecret}
}

type docContentResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Content string `json:"content"`
	} `json:"data"`
}

type DocMeta struct {
	Title      string `json:"title"`
	DocumentID string `json:"document_id"`
}

type docMetaResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Document DocMeta `json:"document"`
	} `json:"data"`
}

func (c *DocClient) getAppToken() (string, error) {
	return (&OAuthClient{AppID: c.AppID, AppSecret: c.AppSecret}).GetAppAccessToken()
}

// GetDocMeta fetches the document title and metadata from Feishu.
func (c *DocClient) GetDocMeta(docToken string) (*DocMeta, error) {
	appToken, err := c.getAppToken()
	if err != nil {
		return nil, fmt.Errorf("get app token: %w", err)
	}
	url := fmt.Sprintf("https://open.feishu.cn/open-apis/docx/v1/documents/%s", docToken)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+appToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request doc meta: %w", err)
	}
	defer resp.Body.Close()
	var result docMetaResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode doc meta: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("feishu doc error: %s", result.Msg)
	}
	return &result.Data.Document, nil
}

func (c *DocClient) GetDocContent(docToken string) (string, error) {
	appToken, err := c.getAppToken()
	if err != nil {
		return "", fmt.Errorf("get app token: %w", err)
	}
	url := fmt.Sprintf("https://open.feishu.cn/open-apis/docx/v1/documents/%s/raw_content", docToken)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+appToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request doc content: %w", err)
	}
	defer resp.Body.Close()
	var result docContentResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode doc content: %w", err)
	}
	if result.Code != 0 {
		return "", fmt.Errorf("feishu doc error: %s", result.Msg)
	}
	return result.Data.Content, nil
}

func ExtractDocToken(feishuURL string) string {
	// Remove query string and fragment
	u := feishuURL
	if idx := strings.Index(u, "?"); idx != -1 {
		u = u[:idx]
	}
	if idx := strings.Index(u, "#"); idx != -1 {
		u = u[:idx]
	}
	parts := strings.Split(strings.TrimRight(u, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
