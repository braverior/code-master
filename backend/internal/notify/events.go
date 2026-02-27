package notify

// RequirementCreatedEvent is sent when a new requirement is created with an assignee.
type RequirementCreatedEvent struct {
	RequirementID   uint
	Title           string
	ProjectName     string
	CreatorName     string
	AssigneeOpenID  string
	Priority        string
}

// RequirementAssignedEvent is sent when a requirement is assigned (or reassigned) to someone.
type RequirementAssignedEvent struct {
	RequirementID  uint
	Title          string
	ProjectName    string
	AssignerName   string // 谁指派的
	AssigneeOpenID string // 被指派人飞书 open_id
	Priority       string
}

// CodegenCompletedEvent is sent when code generation completes successfully.
type CodegenCompletedEvent struct {
	RequirementID   uint
	Title           string
	ProjectName     string
	TaskID          uint
	CreatorOpenID   string
	AssigneeOpenID  string
	FilesChanged    int
	Additions       int
	Deletions       int
}

// CodegenFailedEvent is sent when code generation fails.
type CodegenFailedEvent struct {
	RequirementID  uint
	Title          string
	ProjectName    string
	TaskID         uint
	CreatorOpenID  string
	AssigneeOpenID string
	ErrorMessage   string
}

// AIReviewCompletedEvent is sent when AI review finishes.
type AIReviewCompletedEvent struct {
	RequirementID  uint
	Title          string
	ProjectName    string
	ReviewID       uint
	CreatorOpenID  string
	AssigneeOpenID string
	AIScore        *int
	AIStatus       string
}

// HumanReviewSubmittedEvent is sent when a human reviewer submits their review.
type HumanReviewSubmittedEvent struct {
	RequirementID  uint
	Title          string
	ProjectName    string
	ReviewID       uint
	CreatorOpenID  string
	AssigneeOpenID string
	ReviewerName   string
	Status         string
	Comment        string
}
