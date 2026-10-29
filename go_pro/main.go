package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"corp-portal/business"
	"corp-portal/data"
	"corp-portal/domain"
	"corp-portal/transport"
)

//go:embed templates/*.html
var templateFiles embed.FS

var templates = template.Must(template.ParseFS(templateFiles, "templates/*.html"))

func iptr(v int) *int { return &v }

type directionRow struct {
	ID   int    `gorm:"column:ID;primaryKey"`
	Name string `gorm:"column:Name"`
}

func (directionRow) TableName() string { return "Directions" }

type categoryRow struct {
	ID   int    `gorm:"column:ID;primaryKey"`
	Name string `gorm:"column:Name"`
}

func (categoryRow) TableName() string { return "Categories" }

type documentRow struct {
	ID          int    `gorm:"column:ID;primaryKey;autoIncrement"`
	Title       string `gorm:"column:Title"`
	DirectionID int    `gorm:"column:DirectionID"`
	CategoryID  int    `gorm:"column:CategoryID"`
}

func (documentRow) TableName() string { return "Documents" }

type wikiSectionRow struct {
	ID   int    `gorm:"column:ID;primaryKey"`
	Name string `gorm:"column:Name"`
}

func (wikiSectionRow) TableName() string { return "WikiSections" }

type wikiPageRow struct {
	ID        int    `gorm:"column:ID;primaryKey;autoIncrement"`
	Title     string `gorm:"column:Title"`
	SectionID int    `gorm:"column:SectionID"`
}

func (wikiPageRow) TableName() string { return "WikiPages" }

type projectRow struct {
	ID     int    `gorm:"column:ID;primaryKey"`
	Name   string `gorm:"column:Name"`
	LeadID int    `gorm:"column:LeadID"`
}

func (projectRow) TableName() string { return "Projects" }

type userRow struct {
	ID   int    `gorm:"column:ID;primaryKey"`
	Name string `gorm:"column:Name"`
}

func (userRow) TableName() string { return "Users" }

type roleRow struct {
	ID   int    `gorm:"column:ID;primaryKey"`
	Name string `gorm:"column:Name"`
}

func (roleRow) TableName() string { return "Roles" }

type userRoleRow struct {
	UserID int `gorm:"column:UserID;primaryKey"`
	RoleID int `gorm:"column:RoleID;primaryKey"`
}

func (userRoleRow) TableName() string { return "UserRoles" }

type kanbanTaskRow struct {
	ID         int    `gorm:"column:ID;primaryKey;autoIncrement"`
	Title      string `gorm:"column:Title"`
	ProjectID  int    `gorm:"column:ProjectID"`
	AssigneeID int    `gorm:"column:AssigneeID"`
}

func (kanbanTaskRow) TableName() string { return "KanbanTasks" }

type accessRequestRow struct {
	ID            int        `gorm:"column:ID;primaryKey;autoIncrement"`
	SubjectUserID int        `gorm:"column:SubjectUserID"`
	ResourceType  string     `gorm:"column:ResourceType"`
	DirectionID   *int       `gorm:"column:DirectionID"`
	CategoryID    *int       `gorm:"column:CategoryID"`
	Action        string     `gorm:"column:Action"`
	ExpiresAt     *time.Time `gorm:"column:ExpiresAt"`
	RequestedBy   int        `gorm:"column:RequestedBy"`
	ApprovedBy    *int       `gorm:"column:ApprovedBy"`
	IssuedBy      *int       `gorm:"column:IssuedBy"`
	Status        string     `gorm:"column:Status"`
	CreatedAt     time.Time  `gorm:"column:CreatedAt"`
}

func (accessRequestRow) TableName() string { return "AccessRequests" }

func initDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("corp.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Ошибка БД: %v", err)
	}

	if err := db.AutoMigrate(
		&directionRow{}, &categoryRow{}, &documentRow{},
		&wikiSectionRow{}, &wikiPageRow{},
		&projectRow{}, &userRow{}, &roleRow{}, &userRoleRow{},
		&kanbanTaskRow{}, &accessRequestRow{},
	); err != nil {
		log.Fatalf("Ошибка миграции: %v", err)
	}

	if err := data.MigrateUserAccessRules(db); err != nil {
		log.Fatalf("Ошибка миграции UserAccessRules: %v", err)
	}
	if err := data.MigrateRoleRules(db); err != nil {
		log.Fatalf("Ошибка миграции RoleRules: %v", err)
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		for _, m := range []any{
			&directionRow{}, &categoryRow{}, &documentRow{},
			&wikiSectionRow{}, &wikiPageRow{},
			&projectRow{}, &userRow{}, &roleRow{}, &userRoleRow{},
			&kanbanTaskRow{}, &accessRequestRow{},
		} {
			if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).
				Delete(m).Error; err != nil {
				return err
			}
		}

		if err := data.DeleteAllUserAccessRules(tx); err != nil {
			return err
		}
		if err := data.DeleteAllRoleRules(tx); err != nil {
			return err
		}

		if err := tx.Create(&[]directionRow{
			{1, "Рога и копыта"},
			{2, "Шарашкина контора"},
		}).Error; err != nil {
			return err
		}

		if err := tx.Create(&[]categoryRow{
			{1, "Отчеты"},
			{2, "Регламенты"},
			{3, "Финансы"},
		}).Error; err != nil {
			return err
		}

		if err := tx.Create(&[]documentRow{
			{Title: "Отчет за Q1", DirectionID: 1, CategoryID: 1},
			{Title: "Регламент закупок", DirectionID: 1, CategoryID: 2},
			{Title: "Бюджет 2025", DirectionID: 1, CategoryID: 3},
		}).Error; err != nil {
			return err
		}

		if err := tx.Create(&[]wikiSectionRow{
			{1, "Рога и копыта"},
			{2, "Шарашкина контора"},
		}).Error; err != nil {
			return err
		}

		if err := tx.Create(&[]wikiPageRow{
			{Title: "Инструкция по онбордингу", SectionID: 1},
			{Title: "Регламент совещаний", SectionID: 2},
		}).Error; err != nil {
			return err
		}

		if err := tx.Create(&[]userRow{
			{1, "Анна"}, {2, "Петя"}, {3, "Лидер"}, {4, "Андрей"},
			{5, "Админ"}, {6, "Мария"}, {7, "Иван"}, {8, "Ольга"},
			{9, "Сергей"}, {10, "Дмитрий"}, {11, "Екатерина"}, {12, "Николай"},
		}).Error; err != nil {
			return err
		}

		if err := tx.Create(&[]roleRow{
			{1, "Аналитик"}, {2, "Аналитик закупок"},
			{3, "Лидер проекта"}, {4, "Администратор"},
		}).Error; err != nil {
			return err
		}

		if err := tx.Create(&[]userRoleRow{
			{1, 1}, {2, 1}, {6, 1}, {7, 1}, {8, 1}, {9, 1}, {10, 1}, {11, 1}, {12, 1},
			{3, 3}, {4, 2}, {5, 4},
		}).Error; err != nil {
			return err
		}

		if err := tx.Create(&[]projectRow{
			{1, "Внедрение портала", 3},
			{2, "Миграция данных", 3},
		}).Error; err != nil {
			return err
		}

		if err := tx.Create(&[]kanbanTaskRow{
			{Title: "Собрать требования", ProjectID: 1, AssigneeID: 1},
			{Title: "Согласовать макет", ProjectID: 1, AssigneeID: 2},
			{Title: "Настроить миграцию", ProjectID: 2, AssigneeID: 4},
		}).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		log.Fatalf("Ошибка сидов: %v", err)
	}

	return db
}

func main() {
	db := initDB()

	rules := []domain.AccessRule{
		{ID: 1, RoleID: 1, ResourceType: domain.ResourceDocuments,
			DirectionID: iptr(1), CategoryID: iptr(1), Action: domain.ActionRead},
		{ID: 2, RoleID: 1, ResourceType: domain.ResourceDocuments,
			DirectionID: iptr(1), CategoryID: iptr(2), Action: domain.ActionRead},
		{ID: 3, RoleID: 2, ResourceType: domain.ResourceDocuments,
			DirectionID: iptr(2), CategoryID: iptr(1), Action: domain.ActionRead},

		{ID: 4, RoleID: 1, ResourceType: domain.ResourceDocuments,
			DirectionID: iptr(1), CategoryID: iptr(1), Action: domain.ActionCreate},
		{ID: 5, RoleID: 1, ResourceType: domain.ResourceDocuments,
			DirectionID: iptr(1), CategoryID: iptr(2), Action: domain.ActionCreate},
		{ID: 6, RoleID: 1, ResourceType: domain.ResourceDocuments,
			DirectionID: iptr(1), CategoryID: iptr(3), Action: domain.ActionCreate},

		{ID: 7, RoleID: 1, ResourceType: domain.ResourceDocuments,
			DirectionID: iptr(1), Action: domain.ActionUpdate},
		{ID: 12, RoleID: 1, ResourceType: domain.ResourceDocuments,
			DirectionID: iptr(1), Action: domain.ActionDelete},
		{ID: 8, RoleID: 1, ResourceType: domain.ResourceWiki,
			DirectionID: iptr(1), Action: domain.ActionRead},
		{ID: 9, RoleID: 1, ResourceType: domain.ResourceWiki,
			DirectionID: iptr(2), Action: domain.ActionRead},
		{ID: 13, RoleID: 1, ResourceType: domain.ResourceWiki,
			DirectionID: iptr(1), Action: domain.ActionCreate},
		{ID: 14, RoleID: 1, ResourceType: domain.ResourceWiki,
			DirectionID: iptr(1), Action: domain.ActionUpdate},
		{ID: 15, RoleID: 1, ResourceType: domain.ResourceWiki,
			DirectionID: iptr(1), Action: domain.ActionDelete},
		{ID: 10, RoleID: 3, ResourceType: domain.ResourceKanban,
			ProjectID: iptr(1), Scope: "all", Action: domain.ActionRead},
		{ID: 11, RoleID: 3, ResourceType: domain.ResourceKanban,
			ProjectID: iptr(2), Scope: "all", Action: domain.ActionRead},
	}

	until := time.Now().Add(72 * time.Hour)

	userRules := []domain.UserAccessRule{
		{ID: 1, UserID: 2, ResourceType: domain.ResourceDocuments,
			DirectionID: iptr(1), CategoryID: iptr(1), Action: domain.ActionRead},
		{ID: 2, UserID: 2, ResourceType: domain.ResourceDocuments,
			DirectionID:       iptr(1),
			ExcludeCategoryID: iptr(3),
			Action:            domain.ActionRead},
		{ID: 3, UserID: 2, ResourceType: domain.ResourceDocuments,
			DirectionID: iptr(2), CategoryID: iptr(1),
			Action:    domain.ActionRead,
			ExpiresAt: &until},
		{ID: 4, UserID: 6, ResourceType: domain.ResourceDocuments,
			DirectionID: iptr(2), CategoryID: iptr(1), Action: domain.ActionRead},
		{ID: 5, UserID: 7, ResourceType: domain.ResourceDocuments,
			DirectionID: iptr(2), CategoryID: iptr(1), Action: domain.ActionRead},
		{ID: 6, UserID: 8, ResourceType: domain.ResourceDocuments,
			DirectionID: iptr(2), CategoryID: iptr(1), Action: domain.ActionRead},
	}

	roleRulesRepo := data.NewRoleRulesRepository(db)
	if err := roleRulesRepo.Seed(rules); err != nil {
		log.Fatalf("Ошибка сида RoleRules: %v", err)
	}

	userRulesRepo := data.NewUserAccessRuleRepository(db)
	if err := userRulesRepo.Seed(userRules); err != nil {
		log.Fatalf("Ошибка сида UserAccessRules: %v", err)
	}

	docs := data.NewDocumentRepository(db)
	wiki := data.NewWikiRepository(db)
	kanban := data.NewKanbanRepository(db)
	users := data.NewUserRepository(db)
	access := business.NewAccessService(roleRulesRepo, userRulesRepo)

	requests := data.NewAccessRequestRepository(db)
	h := transport.NewHandler(
		docs, wiki, kanban, users, requests,
		userRulesRepo, roleRulesRepo, access, templates,
	)

	http.HandleFunc("/{$}", h.Documents)
	http.HandleFunc("/documents/create", h.CreateDocument)
	http.HandleFunc("/documents/update", h.UpdateDocument)
	http.HandleFunc("/documents/delete", h.DeleteDocument)
	http.HandleFunc("/wiki", h.Wiki)
	http.HandleFunc("/wiki/create", h.CreateWikiPage)
	http.HandleFunc("/wiki/update", h.UpdateWikiPage)
	http.HandleFunc("/wiki/delete", h.DeleteWikiPage)
	http.HandleFunc("/kanban", h.Kanban)
	http.HandleFunc("/kanban/update", h.UpdateKanbanTask)
	http.HandleFunc("/kanban/delete", h.DeleteKanbanTask)
	http.HandleFunc("/admin", h.Admin)
	http.HandleFunc("/admin/apply-role-change", h.ApplyRoleChange)
	http.HandleFunc("/admin/cancel-role-change", h.CancelRoleChange)
	http.HandleFunc("/admin/remove-user-role", h.RemoveUserRole)
	http.HandleFunc("/admin/add-user-role", h.AddUserRole)
	http.HandleFunc("/admin/delete-user-rule", h.DeleteUserRule)
	http.HandleFunc("/admin/promote-mass-rule", h.PromoteMassRule)
	http.HandleFunc("/switch-user", h.SwitchUser)
	http.HandleFunc("/requests/create", h.CreateRequest)
	http.HandleFunc("/requests/approve", h.ApproveRequest)
	http.HandleFunc("/requests/reject", h.RejectRequest)
	http.HandleFunc("/requests/issue", h.IssueRequest)

	log.Println("Сервер запущен на http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Ошибка запуска: %v", err)
	}
}