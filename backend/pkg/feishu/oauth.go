package feishu

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type OAuthClient struct {
	AppID       string
	AppSecret   string
	RedirectURI string
}

type UserInfo struct {
	OpenID  string `json:"open_id"`
	UnionID string `json:"union_id"`
	Name    string `json:"name"`
	Avatar  string `json:"avatar_url"`
	Email   string `json:"email"`
}

func NewOAuthClient(appID, appSecret, redirectURI string) *OAuthClient {
	return &OAuthClient{
		AppID:       appID,
		AppSecret:   appSecret,
		RedirectURI: redirectURI,
	}
}

func (c *OAuthClient) AuthURL(state string) string {
	params := url.Values{
		"app_id":       {c.AppID},
		"redirect_uri": {c.RedirectURI},
		"state":        {state},
	}
	return "https://open.feishu.cn/open-apis/authen/v1/authorize?" + params.Encode()
}

// feishuResp is the common response wrapper; Feishu uses both "msg" and "message"
// depending on the API, so we capture both.
type feishuResp struct {
	Code    int             `json:"code"`
	Msg     string          `json:"msg"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (r *feishuResp) ErrMsg() string {
	if r.Msg != "" {
		return r.Msg
	}
	return r.Message
}

func (c *OAuthClient) GetAppAccessToken() (string, error) {
	body := fmt.Sprintf(`{"app_id":"%s","app_secret":"%s"}`, c.AppID, c.AppSecret)
	resp, err := http.Post(
		"https://open.feishu.cn/open-apis/auth/v3/app_access_token/internal",
		"application/json",
		strings.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("request app token: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)

	// This endpoint returns a flat response (app_access_token at root level, no "data" wrapper)
	var result struct {
		Code           int    `json:"code"`
		Msg            string `json:"msg"`
		AppAccessToken string `json:"app_access_token"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("decode app token resp: %w (body: %s)", err, string(respBytes))
	}
	if result.Code != 0 {
		return "", fmt.Errorf("get app_access_token failed (code=%d): %s", result.Code, result.Msg)
	}
	if result.AppAccessToken == "" {
		return "", fmt.Errorf("empty app_access_token (body: %s)", string(respBytes))
	}
	return result.AppAccessToken, nil
}

func (c *OAuthClient) GetUserInfoByCode(code string) (*UserInfo, error) {
	appToken, err := c.GetAppAccessToken()
	if err != nil {
		return nil, err
	}

	// Step 1: exchange code for user_access_token
	reqBody := fmt.Sprintf(`{"grant_type":"authorization_code","code":"%s"}`, code)
	req, _ := http.NewRequest("POST",
		"https://open.feishu.cn/open-apis/authen/v1/oidc/access_token",
		strings.NewReader(reqBody),
	)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+appToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request user token: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	var tokenResult feishuResp
	if err := json.Unmarshal(respBytes, &tokenResult); err != nil {
		return nil, fmt.Errorf("decode user token resp: %w (body: %s)", err, string(respBytes))
	}
	if tokenResult.Code != 0 {
		return nil, fmt.Errorf("exchange code failed (code=%d): %s (body: %s)", tokenResult.Code, tokenResult.ErrMsg(), string(respBytes))
	}

	var tokenData struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(tokenResult.Data, &tokenData); err != nil {
		return nil, fmt.Errorf("parse token data: %w (body: %s)", err, string(respBytes))
	}

	// Step 2: use user_access_token to get user info
	return c.getUserInfo(tokenData.AccessToken)
}

func (c *OAuthClient) getUserInfo(userAccessToken string) (*UserInfo, error) {
	req, _ := http.NewRequest("GET",
		"https://open.feishu.cn/open-apis/authen/v1/user_info",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+userAccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request user info: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	var result feishuResp
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("decode user info resp: %w (body: %s)", err, string(respBytes))
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("get user info failed (code=%d): %s", result.Code, result.ErrMsg())
	}

	var userInfo UserInfo
	if err := json.Unmarshal(result.Data, &userInfo); err != nil {
		return nil, fmt.Errorf("parse user info: %w (body: %s)", err, string(respBytes))
	}
	return &userInfo, nil
}
