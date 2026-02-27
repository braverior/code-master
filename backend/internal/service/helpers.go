package service

import (
	"context"

	"github.com/codeMaster/backend/internal/gitops"
)

// testConnectionHelper wraps gitops.TestConnection to avoid import cycle.
func testConnectionHelper(ctx context.Context, gitURL, token string) ([]string, error) {
	return gitops.TestConnection(ctx, gitURL, token)
}

// checkPushPermissionHelper wraps gitops.CheckPushPermission to avoid import cycle.
func checkPushPermissionHelper(platform, gitURL, platformProjectID, token string) error {
	return gitops.CheckPushPermission(platform, gitURL, platformProjectID, token)
}
