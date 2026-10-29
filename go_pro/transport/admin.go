package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"corp-portal/business"
	"corp-portal/domain"
)

type adminView struct {
	User        any
	Users       any
	Orphaned    []business.OrphanedRule
	MassRules   []business.MassRule
	RoleChanges []business.RoleChange
	Requests    []domain.AccessRequest
}

func (h *Handler) Admin(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r, h.UserRepo)

	if !isAdmin(user) {
		errorPage(w, http.StatusForbidden, "Доступ только для администратора")
		return
	}

	users, err := h.UserRepo.ListUsers()
	if err != nil {
		log.Printf("admin users: %v", err)
		errorPage(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}

	reqs, err := h.RequestsRepo.ListAll(r.Context())
	if err != nil {
		log.Printf("admin requests: %v", err)
		reqs = nil
	}

	view := adminView{
		User:      user,
		Users:     users,
		Orphaned:  h.Access.FindOrphanedRules(users),
		MassRules: h.Access.FindMassRules(3),
		Requests:  reqs,
	}

	roleNames := map[int]string{
		1: "Аналитик",
		2: "Аналитик закупок",
		3: "Лидер проекта",
		4: "Администратор",
	}
	for id, name := range roleNames {
		cnt, err := h.UserRepo.CountUsersByRole(id)
		if err != nil {
			log.Printf("count role %d: %v", id, err)
			continue
		}
		view.RoleChanges = append(view.RoleChanges,
			business.DescribeRoleChange(id, name, cnt))
	}

	h.renderTemplate(w, "admin.html", view)
}

var AdminRoleID = 4

func isAdmin(user domain.User) bool {
	for _, role := range user.Roles {
		if role.ID == AdminRoleID || role.Name == "Администратор" {
			return true
		}
	}
	return false
}

var (
	roleSnapshots   = map[int][]int{}
	roleSnapshotsMu sync.Mutex
)

func (h *Handler) ApplyRoleChange(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r, h.UserRepo)

	if !isAdmin(user) {
		errorPage(w, http.StatusForbidden, "Доступ только для администратора")
		return
	}

	if r.Method != http.MethodPost {
		errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	roleID, err := strconv.Atoi(r.FormValue("role_id"))
	if err != nil || roleID <= 0 {
		errorPage(w, http.StatusBadRequest, "Некорректный role_id")
		return
	}

	if roleID == AdminRoleID {
		errorPage(w, http.StatusForbidden,
			"Нельзя удалить роль администратора — вы потеряете доступ к админке")
		return
	}

	ids, err := h.UserRepo.ListUserIDsByRole(roleID)
	if err != nil {
		log.Printf("apply: list users for role %d: %v", roleID, err)
		errorPage(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}

	roleSnapshotsMu.Lock()
	roleSnapshots[roleID] = ids
	roleSnapshotsMu.Unlock()

	affected, err := h.UserRepo.DeleteRoleFromAllUsers(roleID)
	if err != nil {
		log.Printf("apply: delete role %d: %v", roleID, err)
		errorPage(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}

	log.Printf("apply role change: role %d removed from %d users by admin %d",
		roleID, affected, user.ID)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","affected":%d}`, affected)
}

func (h *Handler) CancelRoleChange(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r, h.UserRepo)

	if !isAdmin(user) {
		errorPage(w, http.StatusForbidden, "Доступ только для администратора")
		return
	}

	if r.Method != http.MethodPost {
		errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	roleID, err := strconv.Atoi(r.FormValue("role_id"))
	if err != nil || roleID <= 0 {
		errorPage(w, http.StatusBadRequest, "Некорректный role_id")
		return
	}

	roleSnapshotsMu.Lock()
	ids, ok := roleSnapshots[roleID]
	if ok {
		delete(roleSnapshots, roleID)
	}
	roleSnapshotsMu.Unlock()

	if !ok {
		errorPage(w, http.StatusBadRequest,
			"Нечего откатывать — сначала примените изменение")
		return
	}

	restored := 0
	for _, uid := range ids {
		if err := h.UserRepo.AddRoleToUser(uid, roleID); err != nil {
			log.Printf("cancel: add role %d to user %d: %v", roleID, uid, err)
			continue
		}
		restored++
	}

	log.Printf("cancel role change: role %d restored to %d users by admin %d",
		roleID, restored, user.ID)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":   "ok",
		"restored": restored,
	})
}

func (h *Handler) RemoveUserRole(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r, h.UserRepo)
	if !isAdmin(user) {
		errorPage(w, http.StatusForbidden, "Доступ только для администратора")
		return
	}
	if r.Method != http.MethodPost {
		errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	userID, err := strconv.Atoi(r.FormValue("user_id"))
	if err != nil || userID <= 0 {
		errorPage(w, http.StatusBadRequest, "Некорректный user_id")
		return
	}
	roleID, err := strconv.Atoi(r.FormValue("role_id"))
	if err != nil || roleID <= 0 {
		errorPage(w, http.StatusBadRequest, "Некорректный role_id")
		return
	}
	if roleID == AdminRoleID {
		errorPage(w, http.StatusForbidden, "Нельзя снять роль администратора")
		return
	}

	if err := h.UserRepo.RemoveRoleFromUser(userID, roleID); err != nil {
		log.Printf("remove-user-role: %v", err)
		errorPage(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *Handler) AddUserRole(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r, h.UserRepo)
	if !isAdmin(user) {
		errorPage(w, http.StatusForbidden, "Доступ только для администратора")
		return
	}
	if r.Method != http.MethodPost {
		errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	userID, err := strconv.Atoi(r.FormValue("user_id"))
	if err != nil || userID <= 0 {
		errorPage(w, http.StatusBadRequest, "Некорректный user_id")
		return
	}
	roleID, err := strconv.Atoi(r.FormValue("role_id"))
	if err != nil || roleID <= 0 {
		errorPage(w, http.StatusBadRequest, "Некорректный role_id")
		return
	}

	// Проверяем, нет ли уже такой роли (защита от дублей)
	ids, _ := h.UserRepo.ListUserIDsByRole(roleID)
	for _, id := range ids {
		if id == userID {
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}
	}

	if err := h.UserRepo.AddRoleToUser(userID, roleID); err != nil {
		log.Printf("add-user-role: %v", err)
		errorPage(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *Handler) DeleteUserRule(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r, h.UserRepo)
	if !isAdmin(user) {
		errorPage(w, http.StatusForbidden, "Доступ только для администратора")
		return
	}
	if r.Method != http.MethodPost {
		errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}
	id, err := strconv.Atoi(r.FormValue("rule_id"))
	if err != nil || id <= 0 {
		errorPage(w, http.StatusBadRequest, "Некорректный rule_id")
		return
	}
	if err := h.UserAccessRules.DeleteByID(r.Context(), id); err != nil {
		log.Printf("delete-user-rule: %v", err)
		errorPage(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *Handler) PromoteMassRule(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r, h.UserRepo)
	if !isAdmin(user) {
		errorPage(w, http.StatusForbidden, "Доступ только для администратора")
		return
	}
	if r.Method != http.MethodPost {
		errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	if err := r.ParseForm(); err != nil {
		errorPage(w, http.StatusBadRequest, "Некорректная форма")
		return
	}

	resourceType := domain.ResourceType(r.FormValue("resource_type"))
	action := domain.Action(r.FormValue("action"))

	var dirID, catID, exclID *int
	if v := r.FormValue("direction_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			dirID = &n
		}
	}
	if v := r.FormValue("category_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			catID = &n
		}
	}
	if v := r.FormValue("exclude_category_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			exclID = &n
		}
	}

	userIDs := make([]int, 0)
	for _, s := range r.Form["user_ids"] {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			userIDs = append(userIDs, n)
		}
	}
	if len(userIDs) == 0 {
		errorPage(w, http.StatusBadRequest, "Не выбрано ни одного пользователя")
		return
	}

	roleName := r.FormValue("role_name")
	if roleName == "" {
		roleName = "Роль из массового правила"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	roleID, err := h.UserRepo.CreateRole(roleName)
	if err != nil {
		log.Printf("promote: create role: %v", err)
		errorPage(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}

	newRule := domain.AccessRule{
		RoleID:            roleID,
		ResourceType:      resourceType,
		DirectionID:       dirID,
		CategoryID:        catID,
		ExcludeCategoryID: exclID,
		Action:            action,
	}
	if err := h.RoleRulesRepo.Create(ctx, newRule); err != nil {
		log.Printf("promote: create rule: %v", err)
		errorPage(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}

	for _, uid := range userIDs {
		if err := h.UserRepo.AddRoleToUser(uid, roleID); err != nil {
			log.Printf("promote: add role %d to %d: %v", roleID, uid, err)
		}
	}

	if err := h.UserAccessRules.DeleteByUserAndSpec(ctx, userIDs, resourceType, dirID, catID, exclID, action); err != nil {
		log.Printf("promote: delete user rules: %v", err)
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}