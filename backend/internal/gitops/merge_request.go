package gitops

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type MergeRequestInput struct {
	Platform          string
	PlatformProjectID string
	AccessToken       string
	SourceBranch      string
	TargetBranch      string
	Title             string
	Description       string
	GitURL            string
}

type MergeRequestResult struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

func CreateMergeRequest(input MergeRequestInput) (*MergeRequestResult, error) {
	switch input.Platform {
	case "gitlab":
		return createGitLabMR(input)
	case "github":
		return createGitHubPR(input)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", input.Platform)
	}
}

func createGitLabMR(input MergeRequestInput) (*MergeRequestResult, error) {
	apiBase := extractAPIBase(input.GitURL)
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests", apiBase, input.PlatformProjectID)

	body := map[string]interface{}{
		"source_branch": input.SourceBranch,
		"target_branch": input.TargetBranch,
		"title":         input.Title,
		"description":   input.Description,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PRIVATE-TOKEN", input.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitlab api request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("gitlab api error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		IID    int    `json:"iid"`
		WebURL string `json:"web_url"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse gitlab response: %w", err)
	}

	return &MergeRequestResult{
		ID:  fmt.Sprintf("%d", result.IID),
		URL: result.WebURL,
	}, nil
}

func createGitHubPR(input MergeRequestInput) (*MergeRequestResult, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/pulls", input.PlatformProjectID)

	body := map[string]interface{}{
		"head":  input.SourceBranch,
		"base":  input.TargetBranch,
		"title": input.Title,
		"body":  input.Description,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+input.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github api request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("github api error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse github response: %w", err)
	}

	return &MergeRequestResult{
		ID:  fmt.Sprintf("%d", result.Number),
		URL: result.HTMLURL,
	}, nil
}

func GetMergeRequestStatus(platform, platformProjectID, mrID, accessToken, gitURL string) (string, error) {
	switch platform {
	case "gitlab":
		return getGitLabMRStatus(gitURL, platformProjectID, mrID, accessToken)
	case "github":
		return getGitHubPRStatus(platformProjectID, mrID, accessToken)
	default:
		return "", fmt.Errorf("unsupported platform: %s", platform)
	}
}

func getGitLabMRStatus(gitURL, projectID, mrID, token string) (string, error) {
	apiBase := extractAPIBase(gitURL)
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%s", apiBase, projectID, mrID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		State string `json:"state"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	switch result.State {
	case "merged":
		return "merged", nil
	case "closed":
		return "closed", nil
	default:
		return "created", nil
	}
}

func getGitHubPRStatus(repo, prNumber, token string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%s", repo, prNumber)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		State    string `json:"state"`
		MergedAt string `json:"merged_at"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.MergedAt != "" {
		return "merged", nil
	}
	if result.State == "closed" {
		return "closed", nil
	}
	return "created", nil
}
