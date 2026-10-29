package transport

import (
	"bytes"
	"html/template"
	"log"
	"net/http"

	"corp-portal/business"
	"corp-portal/data"
)

type Handler struct {
	DocsRepo        *data.DocumentRepository
	WikiRepo        *data.WikiRepository
	KanbanRepo      *data.KanbanRepository
	UserRepo        *data.UserRepository
	RequestsRepo    *data.AccessRequestRepository
	UserAccessRules *data.UserAccessRuleRepository
	RoleRulesRepo   *data.RoleRulesRepository
	Access          *business.AccessService
	Tmpl            *template.Template
}

func NewHandler(
	docs *data.DocumentRepository,
	wiki *data.WikiRepository,
	kanban *data.KanbanRepository,
	users *data.UserRepository,
	requests *data.AccessRequestRepository,
	userAccessRules *data.UserAccessRuleRepository,
	roleRules *data.RoleRulesRepository,
	access *business.AccessService,
	tmpl *template.Template,
) *Handler {
	return &Handler{
		DocsRepo:        docs,
		WikiRepo:        wiki,
		KanbanRepo:      kanban,
		UserRepo:        users,
		RequestsRepo:    requests,
		UserAccessRules: userAccessRules,
		RoleRulesRepo:   roleRules,
		Access:          access,
		Tmpl:            tmpl,
	}
}

func (h *Handler) renderTemplate(w http.ResponseWriter, name string, data any) {
	var buf bytes.Buffer
	if err := h.Tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
		errorPage(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("write response %s: %v", name, err)
	}
}