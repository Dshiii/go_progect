package data

import (
	"context"

	"gorm.io/gorm"

	"corp-portal/domain"
)

type roleRuleRow struct {
	ID                int    `gorm:"column:ID;primaryKey;autoIncrement"`
	RoleID            int    `gorm:"column:RoleID"`
	ResourceType      string `gorm:"column:ResourceType"`
	DirectionID       *int   `gorm:"column:DirectionID"`
	CategoryID        *int   `gorm:"column:CategoryID"`
	ExcludeCategoryID *int   `gorm:"column:ExcludeCategoryID"`
	ProjectID         *int   `gorm:"column:ProjectID"`
	Scope             string `gorm:"column:Scope"`
	Action            string `gorm:"column:Action"`
}

func (roleRuleRow) TableName() string { return "RoleRules" }

type RoleRulesRepository struct{ db *gorm.DB }

func NewRoleRulesRepository(db *gorm.DB) *RoleRulesRepository {
	return &RoleRulesRepository{db: db}
}

func MigrateRoleRules(db *gorm.DB) error {
	return db.AutoMigrate(&roleRuleRow{})
}

func DeleteAllRoleRules(tx *gorm.DB) error {
	return tx.Session(&gorm.Session{AllowGlobalUpdate: true}).
		Delete(&roleRuleRow{}).Error
}

func (r *RoleRulesRepository) Seed(rules []domain.AccessRule) error {
	for _, rule := range rules {
		if err := r.db.Create(&roleRuleRow{
			RoleID:            rule.RoleID,
			ResourceType:      string(rule.ResourceType),
			DirectionID:       rule.DirectionID,
			CategoryID:        rule.CategoryID,
			ExcludeCategoryID: rule.ExcludeCategoryID,
			ProjectID:         rule.ProjectID,
			Scope:             rule.Scope,
			Action:            string(rule.Action),
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *RoleRulesRepository) ListAll(ctx context.Context) ([]domain.AccessRule, error) {
	var rows []roleRuleRow
	if err := r.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.AccessRule, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.AccessRule{
			ID:                row.ID,
			RoleID:            row.RoleID,
			ResourceType:      domain.ResourceType(row.ResourceType),
			DirectionID:       row.DirectionID,
			CategoryID:        row.CategoryID,
			ExcludeCategoryID: row.ExcludeCategoryID,
			ProjectID:         row.ProjectID,
			Scope:             row.Scope,
			Action:            domain.Action(row.Action),
		})
	}
	return out, nil
}

func (r *RoleRulesRepository) Create(ctx context.Context, rule domain.AccessRule) error {
	return r.db.WithContext(ctx).Create(&roleRuleRow{
		RoleID:            rule.RoleID,
		ResourceType:      string(rule.ResourceType),
		DirectionID:       rule.DirectionID,
		CategoryID:        rule.CategoryID,
		ExcludeCategoryID: rule.ExcludeCategoryID,
		ProjectID:         rule.ProjectID,
		Scope:             rule.Scope,
		Action:            string(rule.Action),
	}).Error
}