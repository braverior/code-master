package gitops

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func CreateBranch(ctx context.Context, repoDir, branchName string) error {
	cmd := exec.CommandContext(ctx, "git", "checkout", "-b", branchName)
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout -b: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func Push(ctx context.Context, repoDir, branch, gitURL, token string, useLocalGit bool) error {
	gitURL = rewriteGitURL(gitURL)

	log.Printf("[gitops.Push] useLocalGit=%v, gitURL=%s, token_len=%d, branch=%s", useLocalGit, gitURL, len(token), branch)
	env := append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	if useLocalGit {
		// Use local git credentials — push via origin remote
		unshallowCmd := exec.CommandContext(ctx, "git", "fetch", "--unshallow", "origin")
		unshallowCmd.Dir = repoDir
		unshallowCmd.Env = env
		unshallowCmd.CombinedOutput() // ignore errors — may not be shallow

		pushCmd := "git push -u origin " + branch
		log.Printf("[gitops.Push] exec: %s", pushCmd)
		cmd := exec.CommandContext(ctx, "git", "push", "-u", "origin", branch)
		cmd.Dir = repoDir
		cmd.Env = env
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git push: %s: %w", strings.TrimSpace(string(output)), err)
		}
		return nil
	}

	// Token-based push: use auth URL directly and disable credential helpers
	// (same pattern as Clone/FetchAndCheckout to avoid system credential helper interference)
	if token == "" {
		return fmt.Errorf("token is empty, cannot push")
	}
	authURL, err := injectToken(gitURL, token)
	if err != nil {
		return fmt.Errorf("inject token: %w", err)
	}
	log.Printf("[gitops.Push] authURL=%s (original gitURL=%s)", authURL, gitURL)

	// Unshallow if needed — shallow clones may fail to push
	unshallowCmd := exec.CommandContext(ctx, "git", "-c", "credential.helper=",
		"fetch", "--unshallow", authURL)
	unshallowCmd.Dir = repoDir
	unshallowCmd.Env = env
	unshallowCmd.CombinedOutput() // ignore errors — may not be shallow

	// Push directly with auth URL, bypassing credential helpers
	refspec := "HEAD:refs/heads/" + branch
	log.Printf("[gitops.Push] exec: git -c credential.helper= push <authURL> %s", refspec)
	cmd := exec.CommandContext(ctx, "git", "-c", "credential.helper=",
		"push", authURL, refspec)
	cmd.Dir = repoDir
	cmd.Env = env
	output, pushErr := cmd.CombinedOutput()
	log.Printf("[gitops.Push] output=%s, err=%v", strings.TrimSpace(string(output)), pushErr)
	if pushErr != nil {
		// Sanitize error message to avoid leaking token
		errMsg := strings.TrimSpace(string(output))
		if token != "" {
			errMsg = strings.ReplaceAll(errMsg, token, "***")
		}
		return fmt.Errorf("git push: %s: %w", errMsg, pushErr)
	}
	return nil
}

func ConfigUser(ctx context.Context, repoDir string) error {
	cmd1 := exec.CommandContext(ctx, "git", "config", "user.name", "CodeMaster Bot")
	cmd1.Dir = repoDir
	if out, err := cmd1.CombinedOutput(); err != nil {
		return fmt.Errorf("git config user.name: %s: %w", string(out), err)
	}

	cmd2 := exec.CommandContext(ctx, "git", "config", "user.email", "codemaster@bot.local")
	cmd2.Dir = repoDir
	if out, err := cmd2.CombinedOutput(); err != nil {
		return fmt.Errorf("git config user.email: %s: %w", string(out), err)
	}
	return nil
}

// AddAndCommit stages all changes and creates a commit.
// Returns the commit SHA and true if a commit was created, empty string and false if there was nothing to commit.
func AddAndCommit(ctx context.Context, repoDir, message string) (string, bool, error) {
	// git add -A
	addCmd := exec.CommandContext(ctx, "git", "add", "-A")
	addCmd.Dir = repoDir
	if out, err := addCmd.CombinedOutput(); err != nil {
		return "", false, fmt.Errorf("git add: %s: %w", strings.TrimSpace(string(out)), err)
	}

	// Check if there are staged changes
	diffCmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--quiet")
	diffCmd.Dir = repoDir
	if err := diffCmd.Run(); err == nil {
		// exit 0 means no differences — nothing to commit
		return "", false, nil
	}

	// git commit
	commitCmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
	commitCmd.Dir = repoDir
	if out, err := commitCmd.CombinedOutput(); err != nil {
		return "", false, fmt.Errorf("git commit: %s: %w", strings.TrimSpace(string(out)), err)
	}

	// git rev-parse HEAD
	revCmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	revCmd.Dir = repoDir
	sha, err := revCmd.Output()
	if err != nil {
		return "", true, nil // commit succeeded, just can't get SHA
	}

	return strings.TrimSpace(string(sha)), true, nil
}
