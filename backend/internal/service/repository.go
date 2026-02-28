package service

import (
	"context"
	"fmt"

	"github.com/codeMaster/backend/internal/codegen"
	"github.com/codeMaster/backend/internal/model"
	"github.com/codeMaster/backend/pkg/encrypt"
	"gorm.io/gorm"
)

type RepositoryService struct {
	db       *gorm.DB
	aesKey   string
	analyzer *codegen.Analyzer
}

func NewRepositoryService(db *gorm.DB, aesKey string, analyzer *codegen.Analyzer) *RepositoryService {
	return &RepositoryService{db: db, aesKey: aesKey, analyzer: analyzer}
}

func (s *RepositoryService) Create(repo *model.Repository, rawToken string) error {
	if rawToken != "" {
		ctx := context.Background()

		// Step 1: verify read permission via git ls-remote
		if _, err := testConnectionHelper(ctx, repo.GitURL, rawToken); err != nil {
			return fmt.Errorf("50102:仓库连接失败: access token 无效或无权限")
		}

		// Step 2: verify push permission via platform API
		if err := checkPushPermissionHelper(repo.Platform, repo.GitURL, repo.PlatformProjectID, rawToken); err != nil {
			return fmt.Errorf("50103:Token 无推送权限: %s", err.Error())
		}

		encrypted, err := encrypt.AESEncrypt(s.aesKey, rawToken)
		if err != nil {
			return fmt.Errorf("encrypt token: %w", err)
		}
		repo.AccessToken = encrypted
	}
	repo.AnalysisStatus = "pending"
	return s.db.Create(repo).Error
}

func (s *RepositoryService) List(projectID uint, analysisStatus string, page, pageSize int) ([]model.Repository, int64, error) {
	query := s.db.Model(&model.Repository{}).Where("project_id = ?", projectID)
	if analysisStatus != "" {
		query = query.Where("analysis_status = ?", analysisStatus)
	}

	var total int64
	query.Count(&total)

	var repos []model.Repository
	if err := query.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&repos).Error; err != nil {
		return nil, 0, err
	}
	return repos, total, nil
}

func (s *RepositoryService) GetByID(id uint) (*model.Repository, error) {
	var repo model.Repository
	if err := s.db.Preload("Project").First(&repo, id).Error; err != nil {
		return nil, err
	}
	return &repo, nil
}

func (s *RepositoryService) Update(id uint, updates map[string]interface{}) (*model.Repository, error) {
	if rawToken, ok := updates["access_token"]; ok {
		// When token is being updated, validate permissions first
		repo, err := s.GetByID(id)
		if err != nil {
			return nil, err
		}

		token := rawToken.(string)
		ctx := context.Background()

		// Verify read permission
		if _, err := testConnectionHelper(ctx, repo.GitURL, token); err != nil {
			return nil, fmt.Errorf("50102:仓库连接失败: access token 无效或无权限")
		}

		// Verify push permission via platform API
		if err := checkPushPermissionHelper(repo.Platform, repo.GitURL, repo.PlatformProjectID, token); err != nil {
			return nil, fmt.Errorf("50103:Token 无推送权限: %s", err.Error())
		}

		encrypted, err := encrypt.AESEncrypt(s.aesKey, token)
		if err != nil {
			return nil, fmt.Errorf("encrypt token: %w", err)
		}
		updates["access_token"] = encrypted
	}
	if err := s.db.Model(&model.Repository{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetByID(id)
}

func (s *RepositoryService) Delete(id uint) error {
	var runningCount int64
	s.db.Model(&model.CodegenTask{}).Where("repository_id = ? AND status IN ?", id, []string{"pending", "cloning", "running"}).Count(&runningCount)
	if runningCount > 0 {
		return fmt.Errorf("40003:该仓库存在进行中的生成任务，无法解除关联")
	}
	return s.db.Delete(&model.Repository{}, id).Error
}

func (s *RepositoryService) TestConnection(id uint, userID uint) (bool, []string, bool, error) {
	repo, err := s.GetByID(id)
	if err != nil {
		return false, nil, false, err
	}

	// Resolve token: prefer user's personal token, fall back to repo's stored token
	token := s.getUserGitToken(userID)
	if token == "" {
		token, err = encrypt.AESDecrypt(s.aesKey, repo.AccessToken)
		if err != nil {
			return false, nil, false, fmt.Errorf("无可用的 Git Token，请在个人设置中配置")
		}
	}

	ctx := context.Background()

	// Test read permission
	branches, err := testConnectionHelper(ctx, repo.GitURL, token)
	if err != nil {
		return false, nil, false, err
	}

	// Test push permission via platform API
	canPush := true
	if err := checkPushPermissionHelper(repo.Platform, repo.GitURL, repo.PlatformProjectID, token); err != nil {
		canPush = false
	}

	return true, branches, canPush, nil
}

// getUserGitToken retrieves user's personal git token from UserSetting.
func (s *RepositoryService) getUserGitToken(userID uint) string {
	if userID == 0 {
		return ""
	}
	var setting model.UserSetting
	if err := s.db.Where("user_id = ?", userID).First(&setting).Error; err != nil {
		return ""
	}
	return setting.GitlabToken
}

func (s *RepositoryService) TriggerAnalysis(id uint, userID uint) error {
	repo, err := s.GetByID(id)
	if err != nil {
		return err
	}
	if repo.AnalysisStatus == "running" {
		return fmt.Errorf("40003:分析任务正在进行中，请稍后")
	}

	// Query user's LLM settings and git token
	var gitToken, apiKey, baseURL, modelName string
	if userID > 0 {
		var setting model.UserSetting
		if err := s.db.Where("user_id = ?", userID).First(&setting).Error; err == nil {
			gitToken = setting.GitlabToken
			apiKey = setting.APIKey
			baseURL = setting.BaseURL
			modelName = setting.Model
		}
	}

	go s.analyzer.Analyze(context.Background(), repo, gitToken, apiKey, baseURL, modelName)
	return nil
}

func (s *RepositoryService) GetDecryptedToken(id uint) (string, error) {
	repo, err := s.GetByID(id)
	if err != nil {
		return "", err
	}
	return encrypt.AESDecrypt(s.aesKey, repo.AccessToken)
}
