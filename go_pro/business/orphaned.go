package business

import (
	"time"

	"corp-portal/domain"
)

type OrphanedRule struct {
	UserID      int
	UserName    string
	RuleID      int
	Resource    domain.ResourceType
	Description string
}

func (s *AccessService) FindOrphanedRules(users []domain.User) []OrphanedRule {
	var result []OrphanedRule
	byID := make(map[int]domain.User)
	for _, u := range users {
		byID[u.ID] = u
	}

	now := time.Now()
	roleRules := s.roleRulesList()

	for _, rule := range s.userRulesList() {
		if !isActive(rule.ExpiresAt, now) {
			continue
		}
		u, ok := byID[rule.UserID]
		if !ok {
			continue
		}

		covered := false
		for _, rr := range roleRules {
			if rr.ResourceType != rule.ResourceType || rr.Action != rule.Action {
				continue
			}
			roleOwned := false
			for _, r := range u.Roles {
				if r.ID == rr.RoleID {
					roleOwned = true
					break
				}
			}
			if !roleOwned {
				continue
			}
			if rr.DirectionID == nil || rule.DirectionID == nil {
				if rr.DirectionID == nil && rule.DirectionID == nil {
					covered = true
					break
				}
				continue
			}
			if *rr.DirectionID != *rule.DirectionID {
				continue
			}
			if rr.CategoryID == nil && rr.ExcludeCategoryID == nil {
				covered = true
				break
			}
			if rule.CategoryID != nil && rr.CategoryID != nil && *rr.CategoryID == *rule.CategoryID {
				covered = true
				break
			}
			if rule.CategoryID != nil && rr.ExcludeCategoryID != nil && *rr.ExcludeCategoryID != *rule.CategoryID {
				covered = true
				break
			}
		}

		if covered {
			continue
		}

		result = append(result, OrphanedRule{
			UserID:      u.ID,
			UserName:    u.Name,
			RuleID:      rule.ID,
			Resource:    rule.ResourceType,
			Description: "Правило прямого доступа не покрыто ни одной ролью пользователя",
		})
	}
	return result
}