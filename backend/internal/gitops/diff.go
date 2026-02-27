package gitops

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/codeMaster/backend/internal/model"
)

func GetDiffStat(ctx context.Context, repoDir, baseBranch, featureBranch string) (*model.DiffStat, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--stat", baseBranch+".."+featureBranch)
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff --stat: %s: %w", string(output), err)
	}

	stat := &model.DiffStat{}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "file") && strings.Contains(line, "changed") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				stat.FilesChanged, _ = strconv.Atoi(parts[0])
			}
			for i, p := range parts {
				if strings.HasPrefix(p, "insertion") && i > 0 {
					stat.Additions, _ = strconv.Atoi(parts[i-1])
				}
				if strings.HasPrefix(p, "deletion") && i > 0 {
					stat.Deletions, _ = strconv.Atoi(parts[i-1])
				}
			}
		}
	}
	return stat, nil
}

func GetDiffFiles(ctx context.Context, repoDir, baseBranch, featureBranch string) ([]model.DiffFile, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--numstat", baseBranch+".."+featureBranch)
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff --numstat: %s: %w", string(output), err)
	}

	var files []model.DiffFile
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		add, _ := strconv.Atoi(parts[0])
		del, _ := strconv.Atoi(parts[1])
		status := "modified"
		if del == 0 && add > 0 {
			status = "added"
		}
		files = append(files, model.DiffFile{
			Path:      parts[2],
			Status:    status,
			Additions: add,
			Deletions: del,
		})
	}
	return files, nil
}

func GetDiffContent(ctx context.Context, repoDir, baseBranch, featureBranch, filePath string) (string, error) {
	args := []string{"diff", baseBranch + ".." + featureBranch}
	if filePath != "" {
		args = append(args, "--", filePath)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git diff: %s: %w", string(output), err)
	}
	return string(output), nil
}
