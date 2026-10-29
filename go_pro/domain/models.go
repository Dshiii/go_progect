package domain

import "time"

type Direction struct {
	ID   int
	Name string
}

type Category struct {
	ID   int
	Name string
}

type Document struct {
	ID          int
	Title       string
	DirectionID int
	CategoryID  int
	Direction   string
	Category    string
}

type WikiSection struct {
	ID   int
	Name string
}

type WikiPage struct {
	ID        int
	Title     string
	SectionID int
	Section   string
}

type Project struct {
	ID     int
	Name   string
	LeadID int
	Lead   string
}

type KanbanTask struct {
	ID         int
	Title      string
	ProjectID  int
	Project    string
	AssigneeID int
	Assignee   string
}

type Action string

const (
	ActionRead   Action = "read"
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
)

type ResourceType string

const (
	ResourceDocuments ResourceType = "documents"
	ResourceWiki      ResourceType = "wiki"
	ResourceKanban    ResourceType = "kanban"
)

type Role struct {
	ID   int    `gorm:"column:ID"`
	Name string `gorm:"column:Name"`
}

type User struct {
	ID    int    `gorm:"column:ID"`
	Name  string `gorm:"column:Name"`
	Roles []Role `gorm:"-"`
}

type AccessRule struct {
	ID                int
	RoleID            int
	ResourceType      ResourceType
	DirectionID       *int
	CategoryID        *int
	ExcludeCategoryID *int
	ProjectID         *int
	Scope             string
	Action            Action
}

type UserAccessRule struct {
	ID                int
	UserID            int
	ResourceType      ResourceType
	DirectionID       *int
	CategoryID        *int
	ExcludeCategoryID *int
	ProjectID         *int
	Scope             string
	Action            Action
	ExpiresAt         *time.Time
}

type AccessRequest struct {
	ID            int
	SubjectUserID int
	ResourceType  ResourceType
	DirectionID   *int
	CategoryID    *int
	Action        Action
	ExpiresAt     *time.Time
	RequestedBy   int
	ApprovedBy    *int
	IssuedBy      *int
	Status        string
	CreatedAt     time.Time
}

const (
	RequestStatusPending  = "pending"
	RequestStatusApproved = "approved"
	RequestStatusIssued   = "issued"
	RequestStatusRejected = "rejected"
)

func (u User) IsAdmin() bool {
	for _, r := range u.Roles {
		if r.ID == 4 || r.Name == "Администратор" {
			return true
		}
	}
	return false
}

func (u User) HasRole(name string) bool {
	for _, r := range u.Roles {
		if r.Name == name {
			return true
		}
	}
	return false
}