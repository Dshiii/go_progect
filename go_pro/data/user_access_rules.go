package data

import (
	"context"
	"time"

	"gorm.io/gorm"

	"corp-portal/domain"
)

type userAccessRuleRow struct {
	ID                int        `gorm:"column:ID;primaryKey;autoIncrement"`
	UserID            int        `gorm:"column:UserID"`
	ResourceType      string     `gorm:"column:ResourceType"`
	DirectionID       *int       `gorm:"column:DirectionID"`
	CategoryID        *int       `gorm:"column:CategoryID"`
	ExcludeCategoryID *int       `gorm:"column:ExcludeCategoryID"`
	ProjectID         *int       `gorm:"column:ProjectID"`
	Scope             string     `gorm:"column:Scope"`
	Action            string     `gorm:"column:Action"`
	ExpiresAt         *time.Time `gorm:"column:ExpiresAt"`
}

func (userAccessRuleRow) TableName() string { return "UserAccessRules" }

type UserAccessRuleRepository struct{ db *gorm.DB }

func NewUserAccessRuleRepository(db *gorm.DB) *UserAccessRuleRepository {
	return &UserAccessRuleRepository{db: db}
}

func MigrateUserAccessRules(db *gorm.DB) error {
	return db.AutoMigrate(&userAccessRuleRow{})
}

func DeleteAllUserAccessRules(tx *gorm.DB) error {
	return tx.Session(&gorm.Session{AllowGlobalUpdate: true}).
		Delete(&userAccessRuleRow{}).Error
}

func (r *UserAccessRuleRepository) Seed(rules []domain.UserAccessRule) error {
	for _, rule := range rules {
		if err := r.db.Create(&userAccessRuleRow{
			UserID:            rule.UserID,
			ResourceType:      string(rule.ResourceType),
			DirectionID:       rule.DirectionID,
			CategoryID:        rule.CategoryID,
			ExcludeCategoryID: rule.ExcludeCategoryID,
			ProjectID:         rule.ProjectID,
			Scope:             rule.Scope,
			Action:            string(rule.Action),
			ExpiresAt:         rule.ExpiresAt,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *UserAccessRuleRepository) ListAll(ctx context.Context) ([]domain.UserAccessRule, error) {
	var rows []userAccessRuleRow
	if err := r.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.UserAccessRule, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.UserAccessRule{
			ID:                row.ID,
			UserID:            row.UserID,
			ResourceType:      domain.ResourceType(row.ResourceType),
			DirectionID:       row.DirectionID,
			CategoryID:        row.CategoryID,
			ExcludeCategoryID: row.ExcludeCategoryID,
			ProjectID:         row.ProjectID,
			Scope:             row.Scope,
			Action:            domain.Action(row.Action),
			ExpiresAt:         row.ExpiresAt,
		})
	}
	return out, nil
}

func (r *UserAccessRuleRepository) Create(ctx context.Context, rule domain.UserAccessRule) error {
	return r.db.WithContext(ctx).Create(&userAccessRuleRow{
		UserID:            rule.UserID,
		ResourceType:      string(rule.ResourceType),
		DirectionID:       rule.DirectionID,
		CategoryID:        rule.CategoryID,
		ExcludeCategoryID: rule.ExcludeCategoryID,
		ProjectID:         rule.ProjectID,
		Scope:             rule.Scope,
		Action:            string(rule.Action),
		ExpiresAt:         rule.ExpiresAt,
	}).Error
}

func (r *UserAccessRuleRepository) DeleteByID(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).
		Where("ID = ?", id).
		Delete(&userAccessRuleRow{}).Error
}

func (r *UserAccessRuleRepository) DeleteByUserAndSpec(
	ctx context.Context,
	userIDs []int,
	resource domain.ResourceType,
	directionID, categoryID, excludeCategoryID *int,
	action domain.Action,
) error {
	q := r.db.WithContext(ctx).Where("UserID IN ?", userIDs).
		Where("ResourceType = ?", string(resource)).
		Where("Action = ?", string(action))

	if directionID == nil {
		q = q.Where("DirectionID IS NULL")
	} else {
		q = q.Where("DirectionID = ?", *directionID)
	}
	if categoryID == nil {
		q = q.Where("CategoryID IS NULL")
	} else {
		q = q.Where("CategoryID = ?", *categoryID)
	}
	if excludeCategoryID == nil {
		q = q.Where("ExcludeCategoryID IS NULL")
	} else {
		q = q.Where("ExcludeCategoryID = ?", *excludeCategoryID)
	}

	return q.Delete(&userAccessRuleRow{}).Error
}