package service

import (
	"fmt"

	"github.com/codeMaster/backend/internal/model"
	"gorm.io/gorm"
)

type ProjectService struct {
	db *gorm.DB
}

func NewProjectService(db *gorm.DB) *ProjectService {
	return &ProjectService{db: db}
}

func (s *ProjectService) Create(name, description string, ownerID uint, docLinks model.DocLinks, memberIDs []uint) (*model.Project, error) {
	var count int64
	s.db.Model(&model.Project{}).Where("name = ?", name).Count(&count)
	if count > 0 {
		return nil, fmt.Errorf("40005:项目名称已存在")
	}

	project := &model.Project{
		Name:        name,
		Description: description,
		OwnerID:     ownerID,
		DocLinks:    docLinks,
		Status:      "active",
	}
	if err := s.db.Create(project).Error; err != nil {
		return nil, err
	}

	// Add owner as member with their actual role
	var owner model.User
	s.db.First(&owner, ownerID)
	ownerRole := owner.Role
	if ownerRole == "" {
		ownerRole = "rd"
	}
	ownerMember := &model.ProjectMember{
		ProjectID: project.ID,
		UserID:    ownerID,
		Role:      ownerRole,
	}
	s.db.Create(ownerMember)

	// Add additional members
	for _, uid := range memberIDs {
		if uid == ownerID {
			continue
		}
		member := &model.ProjectMember{
			ProjectID: project.ID,
			UserID:    uid,
			Role:      "rd",
		}
		s.db.Create(member)
	}

	s.db.Preload("Owner").First(project, project.ID)
	return project, nil
}

func (s *ProjectService) List(userID uint, isAdmin bool, keyword, status string, ownerID *uint, page, pageSize int, sortBy, order string) ([]model.Project, int64, error) {
	query := s.db.Model(&model.Project{})

	if !isAdmin {
		query = query.Where("id IN (SELECT project_id FROM project_members WHERE user_id = ?)", userID)
	}
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if ownerID != nil {
		query = query.Where("owner_id = ?", *ownerID)
	}

	var total int64
	query.Count(&total)

	if sortBy == "" {
		sortBy = "updated_at"
	}
	if order == "" {
		order = "desc"
	}
	query = query.Order(sortBy + " " + order)

	var projects []model.Project
	if err := query.Preload("Owner").Offset((page-1)*pageSize).Limit(pageSize).Find(&projects).Error; err != nil {
		return nil, 0, err
	}
	return projects, total, nil
}

func (s *ProjectService) GetByID(id uint) (*model.Project, error) {
	var project model.Project
	if err := s.db.Preload("Owner").Preload("Members.User").First(&project, id).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (s *ProjectService) Update(id uint, updates map[string]interface{}) (*model.Project, error) {
	if name, ok := updates["name"]; ok {
		var count int64
		s.db.Model(&model.Project{}).Where("name = ? AND id != ?", name, id).Count(&count)
		if count > 0 {
			return nil, fmt.Errorf("40005:项目名称已存在")
		}
	}
	if err := s.db.Model(&model.Project{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetByID(id)
}

func (s *ProjectService) Archive(id uint) error {
	var runningCount int64
	s.db.Model(&model.CodegenTask{}).
		Joins("JOIN requirements ON codegen_tasks.requirement_id = requirements.id").
		Where("requirements.project_id = ? AND codegen_tasks.status IN ?", id, []string{"pending", "cloning", "running"}).
		Count(&runningCount)
	if runningCount > 0 {
		return fmt.Errorf("40003:项目存在进行中的代码生成任务，无法归档")
	}
	return s.db.Model(&model.Project{}).Where("id = ?", id).Update("status", "archived").Error
}

func (s *ProjectService) IsMember(projectID, userID uint) bool {
	var count int64
	s.db.Model(&model.ProjectMember{}).Where("project_id = ? AND user_id = ?", projectID, userID).Count(&count)
	return count > 0
}

func (s *ProjectService) AddMembers(projectID uint, userIDs []uint, role string) ([]model.UserBrief, []uint, error) {
	var added []model.UserBrief
	var skipped []uint

	for _, uid := range userIDs {
		var user model.User
		if err := s.db.First(&user, uid).Error; err != nil {
			return nil, nil, fmt.Errorf("40401:用户不存在: id=%d", uid)
		}

		var count int64
		s.db.Model(&model.ProjectMember{}).Where("project_id = ? AND user_id = ?", projectID, uid).Count(&count)
		if count > 0 {
			skipped = append(skipped, uid)
			continue
		}

		member := &model.ProjectMember{
			ProjectID: projectID,
			UserID:    uid,
			Role:      role,
		}
		s.db.Create(member)
		added = append(added, model.UserBrief{ID: user.ID, Name: user.Name, Role: role})
	}
	return added, skipped, nil
}

func (s *ProjectService) RemoveMember(projectID, userID uint) error {
	var project model.Project
	if err := s.db.First(&project, projectID).Error; err != nil {
		return err
	}
	if project.OwnerID == userID {
		return fmt.Errorf("40003:不能移除项目所有者")
	}

	result := s.db.Where("project_id = ? AND user_id = ?", projectID, userID).Delete(&model.ProjectMember{})
	if result.RowsAffected == 0 {
		return fmt.Errorf("40401:该用户不是项目成员")
	}
	return nil
}

func (s *ProjectService) GetProjectStats(projectID uint) map[string]int64 {
	stats := make(map[string]int64)
	statuses := []string{"draft", "generating", "generated", "reviewing", "approved", "merged"}
	for _, st := range statuses {
		var count int64
		s.db.Model(&model.Requirement{}).Where("project_id = ? AND status = ?", projectID, st).Count(&count)
		stats[st] = count
	}
	var total int64
	s.db.Model(&model.Requirement{}).Where("project_id = ?", projectID).Count(&total)
	stats["total_requirements"] = total
	return stats
}

func (s *ProjectService) GetMemberCount(projectID uint) int64 {
	var count int64
	s.db.Model(&model.ProjectMember{}).Where("project_id = ?", projectID).Count(&count)
	return count
}

func (s *ProjectService) GetRepoCount(projectID uint) int64 {
	var count int64
	s.db.Model(&model.Repository{}).Where("project_id = ?", projectID).Count(&count)
	return count
}

func (s *ProjectService) GetRequirementCount(projectID uint) int64 {
	var count int64
	s.db.Model(&model.Requirement{}).Where("project_id = ?", projectID).Count(&count)
	return count
}

func (s *ProjectService) GetOpenRequirementCount(projectID uint) int64 {
	var count int64
	s.db.Model(&model.Requirement{}).Where("project_id = ? AND status NOT IN ?", projectID, []string{"merged"}).Count(&count)
	return count
}
