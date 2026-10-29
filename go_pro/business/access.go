package business

import (
	"context"
	"time"

	"corp-portal/domain"
)

type AllowedScope struct {
	DirectionID       int
	CategoryID        *int
	ExcludeCategoryID *int
}

const wildcardDirection = -1

type DocumentFilter struct {
	AllowedScopes []AllowedScope
}

type KanbanFilter struct {
	AllProjects bool
	ProjectIDs  []int
	UserID      int
}

type SearchQuery struct {
	Text string
}

type RoleRuleInfo struct {
	RoleID   int
	RoleName string
}

type UserRulesProvider interface {
	ListAll(ctx context.Context) ([]domain.UserAccessRule, error)
}

type RoleRulesProvider interface {
	ListAll(ctx context.Context) ([]domain.AccessRule, error)
}

type AccessService struct {
	rules     RoleRulesProvider
	userRules UserRulesProvider
}

func NewAccessService(rules RoleRulesProvider, userRules UserRulesProvider) *AccessService {
	return &AccessService{rules: rules, userRules: userRules}
}

func (s *AccessService) roleRulesList() []domain.AccessRule {
	rules, err := s.rules.ListAll(context.Background())
	if err != nil {
		return nil
	}
	return rules
}

func (s *AccessService) userRulesList() []domain.UserAccessRule {
	rules, err := s.userRules.ListAll(context.Background())
	if err != nil {
		return nil
	}
	return rules
}

func isActive(expiresAt *time.Time, now time.Time) bool {
	return expiresAt == nil || !expiresAt.Before(now)
}

func appendScope(scopes []AllowedScope, dirID, catID, exclID *int) []AllowedScope {
	if dirID == nil {
		return append(scopes, AllowedScope{DirectionID: wildcardDirection})
	}
	return append(scopes, AllowedScope{
		DirectionID:       *dirID,
		CategoryID:        catID,
		ExcludeCategoryID: exclID,
	})
}

func (s *AccessService) BuildDocumentFilter(user domain.User, resource domain.ResourceType, action domain.Action) DocumentFilter {
	if user.IsAdmin() {
		return DocumentFilter{AllowedScopes: []AllowedScope{{DirectionID: wildcardDirection}}}
	}

	now := time.Now()
	var scopes []AllowedScope
	roleIDs := make(map[int]bool, len(user.Roles))
	for _, r := range user.Roles {
		roleIDs[r.ID] = true
	}

	for _, rule := range s.roleRulesList() {
		if roleIDs[rule.RoleID] && rule.ResourceType == resource && rule.Action == action {
			scopes = appendScope(scopes, rule.DirectionID, rule.CategoryID, rule.ExcludeCategoryID)
		}
	}

	for _, rule := range s.userRulesList() {
		if rule.UserID == user.ID && rule.ResourceType == resource && rule.Action == action && isActive(rule.ExpiresAt, now) {
			scopes = appendScope(scopes, rule.DirectionID, rule.CategoryID, rule.ExcludeCategoryID)
		}
	}

	return DocumentFilter{AllowedScopes: scopes}
}

func (s *AccessService) CanActionDoc(user domain.User, action domain.Action, directionID, categoryID int) bool {
	if action == domain.ActionCreate && categoryID <= 0 {
		return false
	}
	f := s.BuildDocumentFilter(user, domain.ResourceDocuments, action)
	for _, sc := range f.AllowedScopes {
		if sc.DirectionID == wildcardDirection {
			return true
		}
		if sc.DirectionID != directionID {
			continue
		}
		if sc.ExcludeCategoryID != nil && *sc.ExcludeCategoryID == categoryID {
			continue
		}
		if sc.CategoryID == nil || *sc.CategoryID == categoryID {
			return true
		}
	}
	return false
}

func (s *AccessService) CanActionWiki(user domain.User, action domain.Action, sectionID int) bool {
	f := s.BuildDocumentFilter(user, domain.ResourceWiki, action)
	for _, sc := range f.AllowedScopes {
		if sc.DirectionID == wildcardDirection || sc.DirectionID == sectionID {
			return true
		}
	}
	return false
}

func (s *AccessService) AllowedCreateScopes(user domain.User) DocumentFilter {
	return s.BuildDocumentFilter(user, domain.ResourceDocuments, domain.ActionCreate)
}

func (s *AccessService) kanbanAccess(user domain.User) (bool, map[int]bool) {
	roleIDs := make(map[int]bool, len(user.Roles))
	for _, r := range user.Roles {
		roleIDs[r.ID] = true
	}

	now := time.Now()
	allProjects := false
	projectSet := make(map[int]bool)

	for _, rule := range s.roleRulesList() {
		if rule.ResourceType == domain.ResourceKanban && roleIDs[rule.RoleID] && rule.Action == domain.ActionRead && rule.Scope == "all" {
			if rule.ProjectID == nil {
				allProjects = true
			} else {
				projectSet[*rule.ProjectID] = true
			}
		}
	}

	for _, rule := range s.userRulesList() {
		if rule.UserID == user.ID && rule.ResourceType == domain.ResourceKanban && rule.Action == domain.ActionRead && isActive(rule.ExpiresAt, now) && rule.Scope == "all" {
			if rule.ProjectID == nil {
				allProjects = true
			} else {
				projectSet[*rule.ProjectID] = true
			}
		}
	}
	return allProjects, projectSet
}

func (s *AccessService) BuildKanbanFilter(user domain.User) KanbanFilter {
	if user.IsAdmin() {
		return KanbanFilter{AllProjects: true, UserID: user.ID}
	}
	allProjects, projectSet := s.kanbanAccess(user)
	f := KanbanFilter{UserID: user.ID, AllProjects: allProjects}
	if len(projectSet) > 0 {
		f.ProjectIDs = make([]int, 0, len(projectSet))
		for id := range projectSet {
			f.ProjectIDs = append(f.ProjectIDs, id)
		}
	}
	return f
}

func (s *AccessService) BuildSearchFilter(user domain.User, resource domain.ResourceType, text string) (DocumentFilter, SearchQuery) {
	return s.BuildDocumentFilter(user, resource, domain.ActionRead), SearchQuery{Text: text}
}

func (s *AccessService) RoleRules() []domain.AccessRule       { return s.roleRulesList() }
func (s *AccessService) UserRules() []domain.UserAccessRule  { return s.userRulesList() }

func (s *AccessService) RoleRuleInfos() []RoleRuleInfo {
	rules := s.roleRulesList()
	seen := make(map[int]bool, len(rules))
	out := make([]RoleRuleInfo, 0, len(rules))
	for _, r := range rules {
		if !seen[r.RoleID] {
			seen[r.RoleID] = true
			out = append(out, RoleRuleInfo{RoleID: r.RoleID})
		}
	}
	return out
}

func (s *AccessService) CanMutateKanban(user domain.User, task domain.KanbanTask) bool {
	if user.IsAdmin() || task.AssigneeID == user.ID {
		return true
	}
	allProjects, projectSet := s.kanbanAccess(user)
	return allProjects || projectSet[task.ProjectID]
}