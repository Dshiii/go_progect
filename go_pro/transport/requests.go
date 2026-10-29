package transport

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"corp-portal/domain"
)

func (h *Handler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r, h.UserRepo)

	if r.Method == http.MethodGet {
		users, err := h.UserRepo.ListUsers()
		if err != nil {
			log.Printf("requests: list users: %v", err)
			errorPage(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
			return
		}
		h.renderTemplate(w, "requests_create.html", map[string]any{
			"User":     user,
			"Users":    users,
			"Resource": r.URL.Query().Get("resource"), // добавляем поддержку предвыбора ресурса
		})
		return
	}

	if r.Method != http.MethodPost {
		errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	subjectID := user.ID
	// Если админ — он может выбрать другого subject_user_id из формы.
	// Если обычный пользователь — заявка всегда на себя.
	if isAdmin(user) {
		if v := r.FormValue("subject_user_id"); v != "" {
			if id, err := strconv.Atoi(v); err == nil && id > 0 {
				subjectID = id
			}
		}
	}

	resourceType := domain.ResourceType(r.FormValue("resource_type"))
	action := domain.Action(r.FormValue("action"))

	var dirID, catID *int
	if v := r.FormValue("direction_id"); v != "" && v != "0" && v != "any" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			dirID = &n
		}
	}
	if v := r.FormValue("category_id"); v != "" && v != "0" && v != "any" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			catID = &n
		}
	}

	var expiresAt *time.Time
	if v := r.FormValue("expires_hours"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			t := time.Now().Add(time.Duration(n) * time.Hour)
			expiresAt = &t
		}
	}

	req := domain.AccessRequest{
		SubjectUserID: subjectID,
		ResourceType:  resourceType,
		DirectionID:   dirID,
		CategoryID:    catID,
		Action:        action,
		ExpiresAt:     expiresAt,
		RequestedBy:   user.ID,
		Status:        domain.RequestStatusPending,
		CreatedAt:     time.Now(),
	}

	if err := h.RequestsRepo.Create(ctx, req); err != nil {
		log.Printf("requests: create: %v", err)
		errorPage(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}

	// Возвращаем пользователя туда, откуда пришёл
	ref := r.Referer()
	if ref == "" {
		ref = "/"
	}
	http.Redirect(w, r, ref, http.StatusSeeOther)
}

func (h *Handler) ApproveRequest(w http.ResponseWriter, r *http.Request) {
	h.changeRequestStatus(w, r, func(ctx context.Context, id, adminID int) error {
		return h.RequestsRepo.Approve(ctx, id, adminID)
	})
}

func (h *Handler) RejectRequest(w http.ResponseWriter, r *http.Request) {
	h.changeRequestStatus(w, r, func(ctx context.Context, id, adminID int) error {
		return h.RequestsRepo.Reject(ctx, id, adminID)
	})
}

func (h *Handler) IssueRequest(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r, h.UserRepo)
	if !isAdmin(user) {
		errorPage(w, http.StatusForbidden, "Доступ только для администратора")
		return
	}
	if r.Method != http.MethodPost {
		errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	id, err := strconv.Atoi(r.FormValue("request_id"))
	if err != nil || id <= 0 {
		errorPage(w, http.StatusBadRequest, "Некорректный request_id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	req, err := h.RequestsRepo.GetByID(ctx, id)
	if err != nil {
		log.Printf("requests: get %d: %v", id, err)
		errorPage(w, http.StatusNotFound, "Заявка не найдена")
		return
	}
	if req.Status != domain.RequestStatusApproved {
		errorPage(w, http.StatusBadRequest, "Заявку нужно сначала согласовать")
		return
	}

	rule := domain.UserAccessRule{
		UserID:       req.SubjectUserID,
		ResourceType: req.ResourceType,
		DirectionID:  req.DirectionID,
		CategoryID:   req.CategoryID,
		Action:       req.Action,
		ExpiresAt:    req.ExpiresAt,
	}
	if err := h.UserAccessRules.Create(ctx, rule); err != nil {
		log.Printf("requests: create rule: %v", err)
		errorPage(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}

	if err := h.RequestsRepo.Issue(ctx, id, user.ID); err != nil {
		log.Printf("requests: issue %d: %v", id, err)
		errorPage(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *Handler) changeRequestStatus(w http.ResponseWriter, r *http.Request, op func(context.Context, int, int) error) {
	user := GetUser(r, h.UserRepo)
	if !isAdmin(user) {
		errorPage(w, http.StatusForbidden, "Доступ только для администратора")
		return
	}
	if r.Method != http.MethodPost {
		errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}
	id, err := strconv.Atoi(r.FormValue("request_id"))
	if err != nil || id <= 0 {
		errorPage(w, http.StatusBadRequest, "Некорректный request_id")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := op(ctx, id, user.ID); err != nil {
		log.Printf("requests: change %d: %v", id, err)
		errorPage(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}