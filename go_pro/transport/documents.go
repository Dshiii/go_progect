package transport

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"corp-portal/business"
	"corp-portal/domain"
)

var directionNames = map[int]string{1: "Рога и копыта", 2: "Шарашкина контора"}
var categoryNames = map[int]string{1: "Отчеты", 2: "Регламенты", 3: "Финансы"}

func dirLabel(id int) string {
	if n, ok := directionNames[id]; ok { return n }
	return "Направление " + strconv.Itoa(id)
}
func catLabel(id int) string {
	if n, ok := categoryNames[id]; ok { return n }
	return "Категория " + strconv.Itoa(id)
}

type createScopeView struct {
	DirectionID       string
	CategoryID        string
	ExcludeCategoryID string
	Label             string
}

func toCreateScopeViews(scopes []business.AllowedScope) []createScopeView {
	out := make([]createScopeView, 0, len(scopes))
	addConcrete := func(dir, cat int) {
		out = append(out, createScopeView{
			DirectionID: strconv.Itoa(dir), CategoryID: strconv.Itoa(cat),
			Label: dirLabel(dir) + " / " + catLabel(cat),
		})
	}

	for _, s := range scopes {
		if s.DirectionID == -1 {
			for d := 1; d <= 2; d++ {
				for c := 1; c <= 3; c++ { addConcrete(d, c) }
			}
			continue
		}
		if s.CategoryID == nil {
			for c := 1; c <= 3; c++ {
				if s.ExcludeCategoryID != nil && c == *s.ExcludeCategoryID { continue }
				addConcrete(s.DirectionID, c)
			}
			continue
		}
		addConcrete(s.DirectionID, *s.CategoryID)
	}

	seen := make(map[string]bool, len(out))
	uniq := make([]createScopeView, 0, len(out))
	for _, v := range out {
		k := v.DirectionID + "|" + v.CategoryID
		if !seen[k] { seen[k] = true; uniq = append(uniq, v) }
	}
	return uniq
}

func (h *Handler) handleErr(w http.ResponseWriter, err error, msg string) bool {
	if err != nil {
		log.Printf("%s: %v", msg, err)
		errorPage(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return true
	}
	return false
}

func (h *Handler) Documents(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	user := GetUser(r, h.UserRepo)
	filter := h.Access.BuildDocumentFilter(user, domain.ResourceDocuments, domain.ActionRead)
	if len(filter.AllowedScopes) == 0 {
		errorPage(w, http.StatusForbidden, "Нет прав на просмотр документов")
		return
	}

	q := r.URL.Query().Get("q")
	docs, err := h.DocsRepo.SearchDocuments(ctx, filter, q)
	if h.handleErr(w, err, "documents") { return }

	h.renderTemplate(w, "documents.html", map[string]any{
		"User": user, "Users": AllUsers(h.UserRepo), "Docs": docs, "Query": q,
	})
}

func (h *Handler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	user := GetUser(r, h.UserRepo)

	if r.Method == http.MethodGet {
		scopes := h.Access.AllowedCreateScopes(user)
		if len(scopes.AllowedScopes) == 0 {
			errorPage(w, http.StatusForbidden, "Нет прав на создание документов")
			return
		}
		h.renderTemplate(w, "create.html", map[string]any{
			"User": user, "Users": AllUsers(h.UserRepo), "Scopes": toCreateScopeViews(scopes.AllowedScopes),
		})
		return
	}
	if r.Method != http.MethodPost {
		errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	title := r.FormValue("title")
	if title == "" { errorPage(w, http.StatusBadRequest, "Название обязательно"); return }

	d, err := strconv.Atoi(r.FormValue("direction"))
	if err != nil || d <= 0 { errorPage(w, http.StatusBadRequest, "Некорректное направление"); return }

	c, _ := strconv.Atoi(r.FormValue("category"))

	if !h.Access.CanActionDoc(user, domain.ActionCreate, d, c) {
		errorPage(w, http.StatusForbidden, "Нет прав на создание в этой области")
		return
	}

	if h.handleErr(w, h.DocsRepo.CreateDocument(ctx, title, d, c), "create doc") { return }
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) SwitchUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}
	id, err := strconv.Atoi(r.FormValue("user_id"))
	if err != nil || id <= 0 {
		errorPage(w, http.StatusBadRequest, "Некорректный user_id")
		return
	}
	SetUser(w, id)
	ref := r.Referer()
	if ref == "" {
		ref = "/"
	}
	http.Redirect(w, r, ref, http.StatusSeeOther)
}

func (h *Handler) UpdateDocument(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	user := GetUser(r, h.UserRepo)
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id <= 0 { errorPage(w, http.StatusBadRequest, "Некорректный ID"); return }

	doc, err := h.DocsRepo.GetDocument(ctx, id)
	if err != nil { errorPage(w, http.StatusNotFound, "Документ не найден"); return }

	if !h.Access.CanActionDoc(user, domain.ActionUpdate, doc.DirectionID, doc.CategoryID) {
		errorPage(w, http.StatusForbidden, "Нет прав на редактирование")
		return
	}

	if r.Method == http.MethodGet {
		h.renderTemplate(w, "edit_document.html", map[string]any{
			"User": user, "Users": AllUsers(h.UserRepo), "Doc": doc,
			"Directions": directionNames, "Categories": categoryNames,
		})
		return
	}
	if r.Method != http.MethodPost { errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается"); return }

	title := r.FormValue("title")
	if title == "" { errorPage(w, http.StatusBadRequest, "Название обязательно"); return }

	newDir, err := strconv.Atoi(r.FormValue("direction_id"))
	if err != nil || newDir <= 0 { errorPage(w, http.StatusBadRequest, "Некорректное направление"); return }
	newCat, err := strconv.Atoi(r.FormValue("category_id"))
	if err != nil || newCat <= 0 { errorPage(w, http.StatusBadRequest, "Некорректная категория"); return }

	if newDir != doc.DirectionID || newCat != doc.CategoryID {
		if !h.Access.CanActionDoc(user, domain.ActionUpdate, newDir, newCat) {
			errorPage(w, http.StatusForbidden, "Нет прав перенести документ в эту область")
			return
		}
	}

	if h.handleErr(w, h.DocsRepo.UpdateDocument(ctx, id, title, newDir, newCat), "update doc") { return }
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	user := GetUser(r, h.UserRepo)
	if r.Method != http.MethodPost { errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается"); return }

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil || id <= 0 { errorPage(w, http.StatusBadRequest, "Некорректный ID"); return }

	doc, err := h.DocsRepo.GetDocument(ctx, id)
	if err != nil { errorPage(w, http.StatusNotFound, "Документ не найден"); return }

	if !h.Access.CanActionDoc(user, domain.ActionDelete, doc.DirectionID, doc.CategoryID) {
		errorPage(w, http.StatusForbidden, "Нет прав на удаление")
		return
	}

	if h.handleErr(w, h.DocsRepo.DeleteDocument(ctx, id), "delete doc") { return }
	http.Redirect(w, r, "/", http.StatusSeeOther)
}