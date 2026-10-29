package data

import (
	"context"
	"fmt"
	"log"

	"gorm.io/gorm"

	"corp-portal/business"
	"corp-portal/domain"
)

func applyScopes(db *gorm.DB, scopes []business.AllowedScope, dirCol, catCol string) *gorm.DB {
	orQ := db.Session(&gorm.Session{NewDB: true}).Where("1 = 0")
	for _, sc := range scopes {
		if sc.DirectionID == -1 {
			orQ = orQ.Or("1 = 1")
			continue
		}
		cond := dirCol + " = ?"
		args := []any{sc.DirectionID}
		if catCol != "" {
			if sc.ExcludeCategoryID != nil {
				cond += " AND " + catCol + " != ?"
				args = append(args, *sc.ExcludeCategoryID)
			}
			if sc.CategoryID != nil {
				cond += " AND " + catCol + " = ?"
				args = append(args, *sc.CategoryID)
			}
		}
		orQ = orQ.Or(cond, args...)
	}
	return orQ
}

type DocumentRepository struct{ db *gorm.DB }

func NewDocumentRepository(db *gorm.DB) *DocumentRepository { return &DocumentRepository{db: db} }

func (r *DocumentRepository) SearchDocuments(ctx context.Context, filter business.DocumentFilter, text string) ([]domain.Document, error) {
	if len(filter.AllowedScopes) == 0 {
		return []domain.Document{}, nil
	}
	q := r.db.WithContext(ctx).Table("Documents AS d").
		Select(`d.ID, d.Title, d.DirectionID, d.CategoryID,
		        dir.Name AS DirectionName, cat.Name AS CategoryName`).
		Joins("JOIN Directions dir ON dir.ID = d.DirectionID").
		Joins("JOIN Categories cat ON cat.ID = d.CategoryID").
		Where(applyScopes(r.db, filter.AllowedScopes, "d.DirectionID", "d.CategoryID"))

	if text != "" {
		like := "%" + text + "%"
		q = q.Where("(d.Title LIKE ? OR dir.Name LIKE ? OR cat.Name LIKE ?)", like, like, like)
	}

	var rows []struct {
		ID          int
		Title       string
		DirectionID int
		CategoryID  int
		DirectionName string `gorm:"column:DirectionName"`
		CategoryName  string `gorm:"column:CategoryName"`
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Document, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Document{
			ID: row.ID, Title: row.Title,
			DirectionID: row.DirectionID, CategoryID: row.CategoryID,
			Direction: row.DirectionName, Category: row.CategoryName,
		})
	}
	return out, nil
}

func (r *DocumentRepository) GetDocument(ctx context.Context, id int) (domain.Document, error) {
	var row struct {
		ID          int
		Title       string
		DirectionID int
		CategoryID  int
		DirectionName string `gorm:"column:DirectionName"`
		CategoryName  string `gorm:"column:CategoryName"`
	}
	err := r.db.WithContext(ctx).Table("Documents AS d").
		Select(`d.ID, d.Title, d.DirectionID, d.CategoryID,
		        dir.Name AS DirectionName, cat.Name AS CategoryName`).
		Joins("JOIN Directions dir ON dir.ID = d.DirectionID").
		Joins("JOIN Categories cat ON cat.ID = d.CategoryID").
		Where("d.ID = ?", id).
		First(&row).Error
	if err != nil {
		return domain.Document{}, err
	}
	return domain.Document{
		ID: row.ID, Title: row.Title,
		DirectionID: row.DirectionID, CategoryID: row.CategoryID,
		Direction: row.DirectionName, Category: row.CategoryName,
	}, nil
}

func (r *DocumentRepository) CreateDocument(ctx context.Context, title string, dirID, catID int) error {
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO Documents (Title, DirectionID, CategoryID) VALUES (?, ?, ?)`,
		title, dirID, catID).Error
}

func (r *DocumentRepository) UpdateDocument(ctx context.Context, id int, title string, dirID, catID int) error {
	return r.db.WithContext(ctx).Exec(
		`UPDATE Documents SET Title = ?, DirectionID = ?, CategoryID = ? WHERE ID = ?`,
		title, dirID, catID, id).Error
}

func (r *DocumentRepository) DeleteDocument(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Exec(`DELETE FROM Documents WHERE ID = ?`, id).Error
}

type WikiRepository struct{ db *gorm.DB }

func NewWikiRepository(db *gorm.DB) *WikiRepository { return &WikiRepository{db: db} }

func (r *WikiRepository) SearchPages(ctx context.Context, filter business.DocumentFilter, text string) ([]domain.WikiPage, error) {
	if len(filter.AllowedScopes) == 0 {
		return []domain.WikiPage{}, nil
	}
	q := r.db.WithContext(ctx).Table("WikiPages AS w").
		Select(`w.ID, w.Title, w.SectionID, s.Name AS SectionName`).
		Joins("JOIN WikiSections s ON s.ID = w.SectionID").
		Where(applyScopes(r.db, filter.AllowedScopes, "w.SectionID", ""))

	if text != "" {
		like := "%" + text + "%"
		q = q.Where("(w.Title LIKE ? OR s.Name LIKE ?)", like, like)
	}

	var rows []struct {
		ID          int
		Title       string
		SectionID   int
		SectionName string `gorm:"column:SectionName"`
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.WikiPage, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.WikiPage{
			ID: row.ID, Title: row.Title,
			SectionID: row.SectionID, Section: row.SectionName,
		})
	}
	return out, nil
}

func (r *WikiRepository) GetPage(ctx context.Context, id int) (domain.WikiPage, error) {
	var row struct {
		ID          int
		Title       string
		SectionID   int
		SectionName string `gorm:"column:SectionName"`
	}
	err := r.db.WithContext(ctx).Table("WikiPages AS w").
		Select(`w.ID, w.Title, w.SectionID, s.Name AS SectionName`).
		Joins("JOIN WikiSections s ON s.ID = w.SectionID").
		Where("w.ID = ?", id).
		First(&row).Error
	if err != nil {
		return domain.WikiPage{}, err
	}
	return domain.WikiPage{
		ID: row.ID, Title: row.Title,
		SectionID: row.SectionID, Section: row.SectionName,
	}, nil
}

func (r *WikiRepository) CreatePage(ctx context.Context, title string, sectionID int) error {
	return r.db.WithContext(ctx).Exec(`INSERT INTO WikiPages (Title, SectionID) VALUES (?, ?)`, title, sectionID).Error
}

func (r *WikiRepository) UpdatePage(ctx context.Context, id int, title string, sectionID int) error {
	return r.db.WithContext(ctx).Exec(`UPDATE WikiPages SET Title = ?, SectionID = ? WHERE ID = ?`, title, sectionID, id).Error
}

func (r *WikiRepository) DeletePage(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Exec(`DELETE FROM WikiPages WHERE ID = ?`, id).Error
}

type KanbanRepository struct{ db *gorm.DB }

func NewKanbanRepository(db *gorm.DB) *KanbanRepository { return &KanbanRepository{db: db} }

func (r *KanbanRepository) SearchTasks(ctx context.Context, f business.KanbanFilter, text string) ([]domain.KanbanTask, error) {
	q := r.db.WithContext(ctx).Table("KanbanTasks AS k").
		Select(`k.ID, k.Title, k.ProjectID, p.Name AS ProjectName,
		        k.AssigneeID, u.Name AS AssigneeName`).
		Joins("JOIN Projects p ON p.ID = k.ProjectID").
		Joins("JOIN Users u ON u.ID = k.AssigneeID")

	switch {
	case f.AllProjects:
	case len(f.ProjectIDs) > 0:
		q = q.Where("k.ProjectID IN ?", f.ProjectIDs)
	default:
		q = q.Where("k.AssigneeID = ?", f.UserID)
	}

	if text != "" {
		like := "%" + text + "%"
		q = q.Where("(k.Title LIKE ? OR p.Name LIKE ? OR u.Name LIKE ?)", like, like, like)
	}

	var rows []struct {
		ID           int
		Title        string
		ProjectID    int
		ProjectName  string `gorm:"column:ProjectName"`
		AssigneeID   int
		AssigneeName string `gorm:"column:AssigneeName"`
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.KanbanTask, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.KanbanTask{
			ID: row.ID, Title: row.Title,
			ProjectID: row.ProjectID, Project: row.ProjectName,
			AssigneeID: row.AssigneeID, Assignee: row.AssigneeName,
		})
	}
	return out, nil
}

func (r *KanbanRepository) GetTask(ctx context.Context, id int) (domain.KanbanTask, error) {
	var row struct {
		ID           int
		Title        string
		ProjectID    int
		ProjectName  string `gorm:"column:ProjectName"`
		AssigneeID   int
		AssigneeName string `gorm:"column:AssigneeName"`
	}
	err := r.db.WithContext(ctx).Table("KanbanTasks AS k").
		Select(`k.ID, k.Title, k.ProjectID, p.Name AS ProjectName,
		        k.AssigneeID, u.Name AS AssigneeName`).
		Joins("JOIN Projects p ON p.ID = k.ProjectID").
		Joins("JOIN Users u ON u.ID = k.AssigneeID").
		Where("k.ID = ?", id).
		First(&row).Error
	if err != nil {
		return domain.KanbanTask{}, err
	}
	return domain.KanbanTask{
		ID: row.ID, Title: row.Title,
		ProjectID: row.ProjectID, Project: row.ProjectName,
		AssigneeID: row.AssigneeID, Assignee: row.AssigneeName,
	}, nil
}

func (r *KanbanRepository) UpdateTask(ctx context.Context, id int, title string, projectID, assigneeID int) error {
	return r.db.WithContext(ctx).Exec(`UPDATE KanbanTasks SET Title = ?, ProjectID = ?, AssigneeID = ? WHERE ID = ?`, title, projectID, assigneeID, id).Error
}

func (r *KanbanRepository) DeleteTask(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Exec(`DELETE FROM KanbanTasks WHERE ID = ?`, id).Error
}

func (r *KanbanRepository) ListProjects(ctx context.Context) ([]domain.Project, error) {
	rows, err := r.db.WithContext(ctx).Raw(`SELECT ID, Name FROM Projects`).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Project
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out = append(out, domain.Project{ID: id, Name: name})
	}
	return out, rows.Err()
}

type UserRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) CountUsersByRole(roleID int) (int64, error) {
	var count int64
	err := r.db.Table("UserRoles").Where("RoleID = ?", roleID).Count(&count).Error
	return count, err
}

func (r *UserRepository) ListUserIDsByRole(roleID int) ([]int, error) {
	var ids []int
	err := r.db.Table("UserRoles").Where("RoleID = ?", roleID).Pluck("UserID", &ids).Error
	return ids, err
}

func (r *UserRepository) DeleteRoleFromAllUsers(roleID int) (int64, error) {
	res := r.db.Exec(`DELETE FROM UserRoles WHERE RoleID = ?`, roleID)
	return res.RowsAffected, res.Error
}

func (r *UserRepository) AddRoleToUser(userID, roleID int) error {
	return r.db.Exec(`INSERT INTO UserRoles (UserID, RoleID) VALUES (?, ?)`, userID, roleID).Error
}

func (r *UserRepository) RemoveRoleFromUser(userID, roleID int) error {
	return r.db.Exec(`DELETE FROM UserRoles WHERE UserID = ? AND RoleID = ?`, userID, roleID).Error
}

func (r *UserRepository) CreateRole(name string) (int, error) {
	res := r.db.Exec(`INSERT INTO Roles (Name) VALUES (?)`, name)
	if res.Error != nil {
		return 0, res.Error
	}
	var id int
	if err := r.db.Raw(`SELECT last_insert_rowid()`).Scan(&id).Error; err != nil {
		return 0, err
	}
	return id, nil
}

func (r *UserRepository) loadRolesForUser(userID int) ([]domain.Role, error) {
	rows, err := r.db.Raw(`
		SELECT r.ID, r.Name
		FROM Roles r
		JOIN UserRoles ur ON ur.RoleID = r.ID
		WHERE ur.UserID = ?
	`, userID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Role
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out = append(out, domain.Role{ID: id, Name: name})
	}
	return out, rows.Err()
}

func (r *UserRepository) ListUsers() ([]domain.User, error) {
	var rows []struct {
		ID   int
		Name string
	}
	if err := r.db.Table("Users").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.User, 0, len(rows))
	for _, row := range rows {
		roles, err := r.loadRolesForUser(row.ID)
		if err != nil {
			log.Printf("ListUsers: roles for user %d: %v", row.ID, err)
			roles = nil
		}
		out = append(out, domain.User{ID: row.ID, Name: row.Name, Roles: roles})
	}
	return out, nil
}

func (r *UserRepository) GetUserByID(id int) (domain.User, error) {
	var row struct {
		ID   int
		Name string
	}
	if err := r.db.Table("Users").Where("ID = ?", id).First(&row).Error; err != nil {
		return domain.User{}, err
	}
	roles, err := r.loadRolesForUser(row.ID)
	if err != nil {
		return domain.User{}, fmt.Errorf("load roles for user %d: %w", row.ID, err)
	}
	return domain.User{ID: row.ID, Name: row.Name, Roles: roles}, nil
}

type AccessRequestRepository struct{ db *gorm.DB }

func NewAccessRequestRepository(db *gorm.DB) *AccessRequestRepository {
	return &AccessRequestRepository{db: db}
}

func (r *AccessRequestRepository) Create(ctx context.Context, req domain.AccessRequest) error {
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO AccessRequests
		 (SubjectUserID, ResourceType, DirectionID, CategoryID, Action,
		  ExpiresAt, RequestedBy, Status, CreatedAt)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.SubjectUserID, req.ResourceType, req.DirectionID, req.CategoryID, req.Action,
		req.ExpiresAt, req.RequestedBy, domain.RequestStatusPending, req.CreatedAt).Error
}

func (r *AccessRequestRepository) Approve(ctx context.Context, id, approverID int) error {
	return r.db.WithContext(ctx).Exec(`UPDATE AccessRequests SET ApprovedBy = ?, Status = ? WHERE ID = ?`, approverID, domain.RequestStatusApproved, id).Error
}

func (r *AccessRequestRepository) Issue(ctx context.Context, id, issuerID int) error {
	return r.db.WithContext(ctx).Exec(`UPDATE AccessRequests SET IssuedBy = ?, Status = ? WHERE ID = ?`, issuerID, domain.RequestStatusIssued, id).Error
}

func (r *AccessRequestRepository) Reject(ctx context.Context, id, approverID int) error {
	return r.db.WithContext(ctx).Exec(`UPDATE AccessRequests SET ApprovedBy = ?, Status = ? WHERE ID = ?`, approverID, domain.RequestStatusRejected, id).Error
}

func (r *AccessRequestRepository) ListAll(ctx context.Context) ([]domain.AccessRequest, error) {
	var out []domain.AccessRequest
	err := r.db.WithContext(ctx).Table("AccessRequests").Find(&out).Error
	return out, err
}

func (r *AccessRequestRepository) GetByID(ctx context.Context, id int) (domain.AccessRequest, error) {
	var req domain.AccessRequest
	err := r.db.WithContext(ctx).Table("AccessRequests").Where("ID = ?", id).First(&req).Error
	return req, err
}