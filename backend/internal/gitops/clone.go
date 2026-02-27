package gitops

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Clone(ctx context.Context, gitURL, token, branch, destDir string) error {
	if err := os.MkdirAll(filepath.Dir(destDir), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	authURL, err := injectToken(gitURL, token)
	if err != nil {
		return fmt.Errorf("inject token: %w", err)
	}

	args := []string{"-c", "credential.helper=", "clone", "--depth", "1", "--branch", branch, authURL, destDir}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func TestConnection(ctx context.Context, gitURL, token string) ([]string, error) {
	authURL, err := injectToken(gitURL, token)
	if err != nil {
		return nil, fmt.Errorf("inject token: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "-c", "credential.helper=", "ls-remote", "--heads", authURL)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git ls-remote: %s: %w", strings.TrimSpace(string(output)), err)
	}

	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			ref := parts[1]
			branch := strings.TrimPrefix(ref, "refs/heads/")
			branches = append(branches, branch)
		}
	}
	return branches, nil
}

// CheckPushPermission verifies write access via the platform API.
// For GitLab: checks project access_level >= 30 (Developer).
// For GitHub: checks permissions.push == true.
func CheckPushPermission(platform, gitURL, platformProjectID, token string) error {
	switch platform {
	case "gitlab":
		return checkGitLabPushPermission(gitURL, platformProjectID, token)
	case "github":
		return checkGitHubPushPermission(platformProjectID, token)
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}
}

func checkGitLabPushPermission(gitURL, projectID, token string) error {
	apiBase := extractAPIBase(gitURL)
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s", apiBase, projectID)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("gitlab api request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("gitlab api error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Permissions struct {
			ProjectAccess *struct {
				AccessLevel int `json:"access_level"`
			} `json:"project_access"`
			GroupAccess *struct {
				AccessLevel int `json:"access_level"`
			} `json:"group_access"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse gitlab response: %w", err)
	}

	// GitLab access levels: 10=Guest, 20=Reporter, 30=Developer, 40=Maintainer, 50=Owner
	// Developer (30) and above can push
	maxLevel := 0
	if result.Permissions.ProjectAccess != nil && result.Permissions.ProjectAccess.AccessLevel > maxLevel {
		maxLevel = result.Permissions.ProjectAccess.AccessLevel
	}
	if result.Permissions.GroupAccess != nil && result.Permissions.GroupAccess.AccessLevel > maxLevel {
		maxLevel = result.Permissions.GroupAccess.AccessLevel
	}

	if maxLevel < 30 {
		return fmt.Errorf("access_level=%d (需要 Developer 30+), 请检查 Token 的 write_repository 权限及项目成员角色", maxLevel)
	}
	return nil
}

func checkGitHubPushPermission(repoFullName, token string) error {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s", repoFullName)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("github api request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("github api error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Permissions struct {
			Push bool `json:"push"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse github response: %w", err)
	}

	if !result.Permissions.Push {
		return fmt.Errorf("Token 无 push 权限, 请检查 Token scope 及仓库协作者权限")
	}
	return nil
}

// extractAPIBase extracts the scheme+host from a git URL.
// e.g. "https://git.domob-inc.cn/group/repo.git" => "https://git.domob-inc.cn"
func extractAPIBase(gitURL string) string {
	u, err := url.Parse(gitURL)
	if err != nil {
		return "https://gitlab.com"
	}
	return fmt.Sprintf("%s://%s", u.Scheme, u.Host)
}

// FetchAndCheckout tries to fetch a branch from remote and check it out locally.
// Used for iterative development: clone default branch first, then switch to existing target branch.
// Returns error if the branch does not exist on remote.
func FetchAndCheckout(ctx context.Context, repoDir, gitURL, token, branch string) error {
	authURL, err := injectToken(gitURL, token)
	if err != nil {
		return fmt.Errorf("inject token: %w", err)
	}

	env := append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	fetchCmd := exec.CommandContext(ctx, "git", "-c", "credential.helper=",
		"fetch", "--depth", "1", authURL, branch)
	fetchCmd.Dir = repoDir
	fetchCmd.Env = env
	if output, err := fetchCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch %s: %s: %w", branch, strings.TrimSpace(string(output)), err)
	}

	checkoutCmd := exec.CommandContext(ctx, "git", "checkout", "-b", branch, "FETCH_HEAD")
	checkoutCmd.Dir = repoDir
	checkoutCmd.Env = env
	if output, err := checkoutCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout %s: %s: %w", branch, strings.TrimSpace(string(output)), err)
	}

	return nil
}

func injectToken(gitURL, token string) (string, error) {
	// Ensure .git suffix to avoid redirects that drop credentials
	if !strings.HasSuffix(gitURL, ".git") {
		gitURL += ".git"
	}

	u, err := url.Parse(gitURL)
	if err != nil {
		return "", err
	}
	u.User = url.UserPassword("oauth2", token)
	return u.String(), nil
}
