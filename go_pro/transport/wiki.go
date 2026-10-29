package transport

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"corp-portal/domain"
)

var wikiSectionNames = map[int]string{1: "Рога и копыта", 2: "Шарашкина контора"}

func (h *Handler) Wiki(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	user := GetUser(r, h.UserRepo)
	filter := h.Access.BuildDocumentFilter(user, domain.ResourceWiki, domain.ActionRead)
	if len(filter.AllowedScopes) == 0 {
		errorPage(w, http.StatusForbidden, "Нет прав на просмотр Wiki")
		return
	}

	q := r.URL.Query().Get("q")
	pages, err := h.WikiRepo.SearchPages(ctx, filter, q)
	if h.handleErr(w, err, "wiki") { return }

	h.renderTemplate(w, "wiki.html", map[string]any{
		"User": user, "Users": AllUsers(h.UserRepo), "Pages": pages, "Query": q,
	})
}

func (h *Handler) CreateWikiPage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	user := GetUser(r, h.UserRepo)
	if r.Method == http.MethodGet {
		h.renderTemplate(w, "create_wiki.html", map[string]any{
			"User": user, "Users": AllUsers(h.UserRepo), "Sections": wikiSectionNames,
		})
		return
	}
	if r.Method != http.MethodPost { errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается"); return }

	title := r.FormValue("title")
	if title == "" { errorPage(w, http.StatusBadRequest, "Заголовок обязателен"); return }
	sectionID, err := strconv.Atoi(r.FormValue("section_id"))
	if err != nil || sectionID <= 0 { errorPage(w, http.StatusBadRequest, "Некорректный раздел"); return }

	if !h.Access.CanActionWiki(user, domain.ActionCreate, sectionID) {
		errorPage(w, http.StatusForbidden, "Нет прав на создание в этом разделе")
		return
	}

	if h.handleErr(w, h.WikiRepo.CreatePage(ctx, title, sectionID), "wiki create") { return }
	http.Redirect(w, r, "/wiki", http.StatusSeeOther)
}

func (h *Handler) UpdateWikiPage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	user := GetUser(r, h.UserRepo)
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id <= 0 { errorPage(w, http.StatusBadRequest, "Некорректный ID"); return }

	page, err := h.WikiRepo.GetPage(ctx, id)
	if err != nil { errorPage(w, http.StatusNotFound, "Страница не найдена"); return }

	if !h.Access.CanActionWiki(user, domain.ActionUpdate, page.SectionID) {
		errorPage(w, http.StatusForbidden, "Нет прав на редактирование этой страницы")
		return
	}

	if r.Method == http.MethodGet {
		h.renderTemplate(w, "edit_wiki.html", map[string]any{
			"User": user, "Users": AllUsers(h.UserRepo), "Page": page, "Sections": wikiSectionNames,
		})
		return
	}
	if r.Method != http.MethodPost { errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается"); return }

	title := r.FormValue("title")
	if title == "" { errorPage(w, http.StatusBadRequest, "Заголовок обязателен"); return }
	newSectionID, err := strconv.Atoi(r.FormValue("section_id"))
	if err != nil || newSectionID <= 0 { errorPage(w, http.StatusBadRequest, "Некорректный раздел"); return }

	if newSectionID != page.SectionID {
		if !h.Access.CanActionWiki(user, domain.ActionUpdate, newSectionID) {
			errorPage(w, http.StatusForbidden, "Нет прав перенести страницу в этот раздел")
			return
		}
	}

	if h.handleErr(w, h.WikiRepo.UpdatePage(ctx, id, title, newSectionID), "wiki update") { return }
	http.Redirect(w, r, "/wiki", http.StatusSeeOther)
}

func (h *Handler) DeleteWikiPage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	user := GetUser(r, h.UserRepo)
	if r.Method != http.MethodPost { errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается"); return }

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil || id <= 0 { errorPage(w, http.StatusBadRequest, "Некорректный ID"); return }

	page, err := h.WikiRepo.GetPage(ctx, id)
	if err != nil { errorPage(w, http.StatusNotFound, "Страница не найдена"); return }

	if !h.Access.CanActionWiki(user, domain.ActionDelete, page.SectionID) {
		errorPage(w, http.StatusForbidden, "Нет прав на удаление этой страницы")
		return
	}

	if h.handleErr(w, h.WikiRepo.DeletePage(ctx, id), "wiki delete") { return }
	http.Redirect(w, r, "/wiki", http.StatusSeeOther)
}