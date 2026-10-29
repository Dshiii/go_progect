package transport

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

func (h *Handler) Kanban(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	user := GetUser(r, h.UserRepo)
	filter := h.Access.BuildKanbanFilter(user)

	q := r.URL.Query().Get("q")
	tasks, err := h.KanbanRepo.SearchTasks(ctx, filter, q)
	if h.handleErr(w, err, "kanban") { return }

	h.renderTemplate(w, "kanban.html", map[string]any{
		"User": user, "Users": AllUsers(h.UserRepo), "Tasks": tasks, "Query": q,
	})
}

func (h *Handler) UpdateKanbanTask(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	user := GetUser(r, h.UserRepo)
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id <= 0 { errorPage(w, http.StatusBadRequest, "Некорректный ID"); return }

	task, err := h.KanbanRepo.GetTask(ctx, id)
	if err != nil { errorPage(w, http.StatusNotFound, "Задача не найдена"); return }

	if !h.Access.CanMutateKanban(user, task) {
		errorPage(w, http.StatusForbidden, "Нет прав на редактирование этой задачи")
		return
	}

	if r.Method == http.MethodGet {
		projects, err := h.KanbanRepo.ListProjects(ctx)
		if h.handleErr(w, err, "kanban list projects") { return }
		h.renderTemplate(w, "edit_kanban.html", map[string]any{
			"User": user, "Users": AllUsers(h.UserRepo), "Task": task, "Projects": projects,
		})
		return
	}
	if r.Method != http.MethodPost { errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается"); return }

	title := r.FormValue("title")
	if title == "" { errorPage(w, http.StatusBadRequest, "Название обязательно"); return }

	projectID, err := strconv.Atoi(r.FormValue("project_id"))
	if err != nil || projectID <= 0 { errorPage(w, http.StatusBadRequest, "Некорректный проект"); return }
	assigneeID, err := strconv.Atoi(r.FormValue("assignee_id"))
	if err != nil || assigneeID <= 0 { errorPage(w, http.StatusBadRequest, "Некорректный исполнитель"); return }

	if projectID != task.ProjectID || assigneeID != task.AssigneeID {
		probe := task
		probe.ProjectID = projectID
		probe.AssigneeID = assigneeID
		if !h.Access.CanMutateKanban(user, probe) {
			errorPage(w, http.StatusForbidden, "Нет прав переназначить задачу в этот проект/исполнителю")
			return
		}
	}

	if h.handleErr(w, h.KanbanRepo.UpdateTask(ctx, id, title, projectID, assigneeID), "kanban update") { return }
	http.Redirect(w, r, "/kanban", http.StatusSeeOther)
}

func (h *Handler) DeleteKanbanTask(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	user := GetUser(r, h.UserRepo)
	if r.Method != http.MethodPost { errorPage(w, http.StatusMethodNotAllowed, "Метод не поддерживается"); return }

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil || id <= 0 { errorPage(w, http.StatusBadRequest, "Некорректный ID"); return }

	task, err := h.KanbanRepo.GetTask(ctx, id)
	if err != nil { errorPage(w, http.StatusNotFound, "Задача не найдена"); return }

	if !h.Access.CanMutateKanban(user, task) {
		errorPage(w, http.StatusForbidden, "Нет прав на удаление этой задачи")
		return
	}

	if h.handleErr(w, h.KanbanRepo.DeleteTask(ctx, id), "kanban delete") { return }
	http.Redirect(w, r, "/kanban", http.StatusSeeOther)
}