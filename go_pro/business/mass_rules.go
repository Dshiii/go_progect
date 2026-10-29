package business

import (
	"time"

	"corp-portal/domain"
)

type MassRule struct {
	ResourceType domain.ResourceType
	DirectionID  *int
	CategoryID   *int
	ExcludeCategoryID *int
	Action       domain.Action
	UserCount    int
	UserIDs      []int
}

func (s *AccessService) FindMassRules(threshold int) []MassRule {
	type key struct {
		R domain.ResourceType
		D int
		C int
		E int
		A domain.Action
	}
	groups := make(map[key]map[int]bool)
	now := time.Now()

	for _, r := range s.userRulesList() {
		if !isActive(r.ExpiresAt, now) {
			continue
		}
		d, c, e := -1, -1, -1
		if r.DirectionID != nil {
			d = *r.DirectionID
		}
		if r.CategoryID != nil {
			c = *r.CategoryID
		}
		if r.ExcludeCategoryID != nil {
			e = *r.ExcludeCategoryID
		}
		k := key{r.ResourceType, d, c, e, r.Action}
		if groups[k] == nil {
			groups[k] = map[int]bool{}
		}
		groups[k][r.UserID] = true
	}

	var result []MassRule
	for k, users := range groups {
		if len(users) < threshold {
			continue
		}
		var dirID, catID, exclID *int
		if k.D != -1 {
			d := k.D
			dirID = &d
		}
		if k.C != -1 {
			c := k.C
			catID = &c
		}
		if k.E != -1 {
			e := k.E
			exclID = &e
		}
		ids := make([]int, 0, len(users))
		for uid := range users {
			ids = append(ids, uid)
		}
		result = append(result, MassRule{
			ResourceType:      k.R,
			DirectionID:       dirID,
			CategoryID:        catID,
			ExcludeCategoryID: exclID,
			Action:            k.A,
			UserCount:         len(users),
			UserIDs:           ids,
		})
	}
	return result
}