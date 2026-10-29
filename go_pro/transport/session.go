package transport

import (
	"html/template"
	"log"
	"net/http"
	"strconv"

	"corp-portal/domain"
)

const userCookie = "user_id"

type UserProvider interface {
	GetUserByID(int) (domain.User, error)
	ListUsers() ([]domain.User, error)
}

func SetUser(w http.ResponseWriter, userID int) {
	http.SetCookie(w, &http.Cookie{
		Name:     userCookie,
		Value:    strconv.Itoa(userID),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

const defaultUserID = 1

func defaultUser() domain.User {
	return domain.User{
		ID:    defaultUserID,
		Name:  "Анна",
		Roles: []domain.Role{{ID: 1, Name: "Аналитик"}},
	}
}

func GetUser(r *http.Request, repo UserProvider) domain.User {
	id := defaultUserID

	if c, err := r.Cookie(userCookie); err == nil {
		if v, err := strconv.Atoi(c.Value); err == nil {
			id = v
		} else {
			log.Printf("session: некорректный user_id в cookie: %q", c.Value)
		}
	}

	u, err := repo.GetUserByID(id)
	if err == nil {
		return u
	}
	log.Printf("session: GetUserByID(%d) не удалось: %v", id, err)

	if id != defaultUserID {
		if u, err := repo.GetUserByID(defaultUserID); err == nil {
			return u
		} else {
			log.Printf("session: fallback GetUserByID(%d) не удалось: %v", defaultUserID, err)
		}
	}

	return defaultUser()
}

func AllUsers(repo UserProvider) []domain.User {
	users, err := repo.ListUsers()
	if err != nil {
		log.Printf("session: ListUsers: %v", err)
		return []domain.User{}
	}
	return users
}

var errorTmpl = template.Must(template.New("error").Parse(`<!DOCTYPE html>
<html lang="ru"><head><meta charset="UTF-8"><title>Ошибка</title></head>
<body>
<h1>Ошибка {{ .Code }}</h1>
<p>{{ .Message }}</p>
<p><a href="/kanban">← На главную</a> | <a href="javascript:history.back()">← Назад</a></p>
</body></html>`))

func errorPage(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	_ = errorTmpl.Execute(w, map[string]any{
		"Code":    code,
		"Message": message,
	})
}