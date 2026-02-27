package service

import (
	"fmt"
	"time"

	"github.com/codeMaster/backend/internal/model"
	"github.com/codeMaster/backend/pkg/feishu"
	jwtpkg "github.com/codeMaster/backend/pkg/jwt"
	"gorm.io/gorm"
)

type AuthService struct {
	db          *gorm.DB
	feishuOAuth *feishu.OAuthClient
	jwtSecret   string
	jwtExpire   int
}

func NewAuthService(db *gorm.DB, feishuOAuth *feishu.OAuthClient, jwtSecret string, jwtExpire int) *AuthService {
	return &AuthService{
		db:          db,
		feishuOAuth: feishuOAuth,
		jwtSecret:   jwtSecret,
		jwtExpire:   jwtExpire,
	}
}

func (s *AuthService) GetFeishuAuthURL(state string) string {
	return s.feishuOAuth.AuthURL(state)
}

func (s *AuthService) HandleCallback(code string) (*model.User, string, time.Time, bool, error) {
	userInfo, err := s.feishuOAuth.GetUserInfoByCode(code)
	if err != nil {
		return nil, "", time.Time{}, false, fmt.Errorf("feishu auth: %w", err)
	}

	var user model.User
	isNew := false

	result := s.db.Where("feishu_uid = ?", userInfo.OpenID).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			isNew = true
			user = model.User{
				FeishuUID:     userInfo.OpenID,
				FeishuUnionID: userInfo.UnionID,
				Name:          userInfo.Name,
				Avatar:        userInfo.Avatar,
				Email:         userInfo.Email,
				Role:          "rd",
				Status:        1,
			}
			if err := s.db.Create(&user).Error; err != nil {
				return nil, "", time.Time{}, false, fmt.Errorf("create user: %w", err)
			}
		} else {
			return nil, "", time.Time{}, false, result.Error
		}
	}

	now := time.Now()
	s.db.Model(&user).Updates(map[string]interface{}{
		"name":          userInfo.Name,
		"avatar":        userInfo.Avatar,
		"email":         userInfo.Email,
		"last_login_at": &now,
	})

	token, expireAt, err := jwtpkg.GenerateToken(s.jwtSecret, user.ID, user.Role, user.IsAdmin, s.jwtExpire)
	if err != nil {
		return nil, "", time.Time{}, false, fmt.Errorf("generate token: %w", err)
	}

	return &user, token, expireAt, isNew, nil
}

func (s *AuthService) GetUserByID(id uint) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *AuthService) UpdateRole(userID uint, role string) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	user.Role = role
	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *AuthService) RefreshToken(userID uint) (string, time.Time, error) {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return "", time.Time{}, err
	}
	return jwtpkg.GenerateToken(s.jwtSecret, user.ID, user.Role, user.IsAdmin, s.jwtExpire)
}

func (s *AuthService) ToggleAdmin(userID uint, isAdmin bool) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	user.IsAdmin = isAdmin
	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *AuthService) ListUsers(keyword, role string, isAdmin *bool, status *int, page, pageSize int, sortBy, order string) ([]model.User, int64, error) {
	query := s.db.Model(&model.User{})
	if keyword != "" {
		query = query.Where("name LIKE ? OR email LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if isAdmin != nil {
		query = query.Where("is_admin = ?", *isAdmin)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	var total int64
	query.Count(&total)

	if sortBy == "" {
		sortBy = "created_at"
	}
	if order == "" {
		order = "desc"
	}
	query = query.Order(sortBy + " " + order)

	var users []model.User
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (s *AuthService) UpdateUserStatus(userID uint, status int) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	user.Status = status
	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *AuthService) SearchUsers(keyword, role string, excludeProjectID *uint, limit int) ([]model.User, error) {
	query := s.db.Model(&model.User{}).Where("status = 1")
	if keyword != "" {
		query = query.Where("name LIKE ? OR email LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if excludeProjectID != nil {
		query = query.Where("id NOT IN (SELECT user_id FROM project_members WHERE project_id = ?)", *excludeProjectID)
	}

	var users []model.User
	if err := query.Limit(limit).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (s *AuthService) GetOperationLogs(userID *uint, action, resourceType string, startTime, endTime *time.Time, page, pageSize int) ([]model.OperationLog, int64, error) {
	query := s.db.Model(&model.OperationLog{}).Preload("User")
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", endTime)
	}

	var total int64
	query.Count(&total)

	var logs []model.OperationLog
	if err := query.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

func (s *AuthService) CreateOperationLog(log *model.OperationLog) error {
	return s.db.Create(log).Error
}
