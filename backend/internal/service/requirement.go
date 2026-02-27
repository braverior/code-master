package service

import (
	"fmt"

	"github.com/codeMaster/backend/internal/model"
	"gorm.io/gorm"
)

type RequirementService struct {
	db *gorm.DB
}

func NewRequirementService(db *gorm.DB) *RequirementService {
	return &RequirementService{db: db}
}

func (s *RequirementService) Create(req *model.Requirement) error {
	return s.db.Create(req).Error
}

func (s *RequirementService) List(projectID uint, status, priority, keyword string, assigneeID, creatorID *uint, page, pageSize int, sortBy, order string) ([]model.Requirement, int64, error) {
	query := s.db.Model(&model.Requirement{}).Where("project_id = ?", projectID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if keyword != "" {
		query = query.Where("title LIKE ?", "%"+keyword+"%")
	}
	if assigneeID != nil {
		query = query.Where("assignee_id = ?", *assigneeID)
	}
	if creatorID != nil {
		query = query.Where("creator_id = ?", *creatorID)
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

	var reqs []model.Requirement
	if err := query.Preload("Creator").Preload("Assignee").Preload("Repository").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&reqs).Error; err != nil {
		return nil, 0, err
	}
	return reqs, total, nil
}

func (s *RequirementService) GetByID(id uint) (*model.Requirement, error) {
	var req model.Requirement
	if err := s.db.Preload("Creator").Preload("Assignee").Preload("Repository").Preload("Project").First(&req, id).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func (s *RequirementService) Update(id uint, updates map[string]interface{}) (*model.Requirement, error) {
	if err := s.db.Model(&model.Requirement{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetByID(id)
}

func (s *RequirementService) Delete(id uint) error {
	return s.db.Delete(&model.Requirement{}, id).Error
}

func (s *RequirementService) GetLatestCodegenTask(requirementID uint) *model.CodegenTask {
	var task model.CodegenTask
	if err := s.db.Where("requirement_id = ?", requirementID).Order("created_at desc").First(&task).Error; err != nil {
		return nil
	}
	return &task
}

func (s *RequirementService) GetLatestReview(requirementID uint) *model.CodeReview {
	var review model.CodeReview
	err := s.db.Joins("JOIN codegen_tasks ON code_reviews.codegen_task_id = codegen_tasks.id").
		Where("codegen_tasks.requirement_id = ?", requirementID).
		Order("code_reviews.created_at desc").First(&review).Error
	if err != nil {
		return nil
	}
	return &review
}

// ListByUser returns requirements where the user is either creator or assignee, across all projects.
func (s *RequirementService) ListByUser(userID uint, page, pageSize int) ([]model.Requirement, int64, error) {
	query := s.db.Model(&model.Requirement{}).
		Where("creator_id = ? OR assignee_id = ?", userID, userID)

	var total int64
	query.Count(&total)

	var reqs []model.Requirement
	if err := query.Preload("Creator").Preload("Assignee").Preload("Project").Preload("Repository").
		Order("updated_at desc").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&reqs).Error; err != nil {
		return nil, 0, err
	}
	return reqs, total, nil
}

// ListAccessible returns requirements from projects the user is a member of.
// Supports filtering by scope: "all" (default), "created" (creator_id=user), "assigned" (assignee_id=user).
// Also supports status and keyword filtering.
func (s *RequirementService) ListAccessible(userID uint, scope, status, keyword string, page, pageSize int) ([]model.Requirement, int64, error) {
	// Find project IDs where the user is a member
	var projectIDs []uint
	s.db.Model(&model.ProjectMember{}).Where("user_id = ?", userID).Pluck("project_id", &projectIDs)
	if len(projectIDs) == 0 {
		return []model.Requirement{}, 0, nil
	}

	query := s.db.Model(&model.Requirement{}).Where("project_id IN ?", projectIDs)

	switch scope {
	case "created":
		query = query.Where("creator_id = ?", userID)
	case "assigned":
		query = query.Where("assignee_id = ?", userID)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword != "" {
		query = query.Where("title LIKE ?", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	var reqs []model.Requirement
	if err := query.Preload("Creator").Preload("Assignee").Preload("Project").Preload("Repository").
		Order("updated_at desc").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&reqs).Error; err != nil {
		return nil, 0, err
	}
	return reqs, total, nil
}

func (s *RequirementService) HasRunningTask(requirementID uint) bool {
	var count int64
	s.db.Model(&model.CodegenTask{}).
		Where("requirement_id = ? AND status IN ?", requirementID, []string{"pending", "cloning", "running"}).
		Count(&count)
	return count > 0
}

func (s *RequirementService) ValidateAssignee(projectID uint, assigneeID uint) error {
	var count int64
	s.db.Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, assigneeID).
		Count(&count)
	if count == 0 {
		return fmt.Errorf("40002:assignee_id 必须是项目成员")
	}
	return nil
}

func (s *RequirementService) ValidateRepository(projectID uint, repoID uint) error {
	var count int64
	s.db.Model(&model.Repository{}).
		Where("project_id = ? AND id = ?", projectID, repoID).
		Count(&count)
	if count == 0 {
		return fmt.Errorf("40002:repository_id 必须是项目关联的仓库")
	}
	return nil
}

func (s *RequirementService) DB() *gorm.DB {
	return s.db
}
