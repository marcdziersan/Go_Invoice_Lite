package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"go-invoice-lite/internal/auth"
	"go-invoice-lite/internal/model"
	"go-invoice-lite/internal/store"
	"go-invoice-lite/internal/web"
)

type Handler struct {
	store    *store.Store
	renderer *web.Renderer
}

func NewHandler(st *store.Store, renderer *web.Renderer) *Handler {
	return &Handler{store: st, renderer: renderer}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if r.URL.Path == "/users" && r.Method == http.MethodGet {
		h.listUsers(w, r)
		return
	}
	if len(parts) == 2 && parts[0] == "users" && parts[1] == "new" {
		switch r.Method {
		case http.MethodGet:
			h.userForm(w, r, model.User{Role: model.RoleUser, Active: true}, "", true)
		case http.MethodPost:
			h.createUser(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	if len(parts) == 3 && parts[0] == "users" {
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		switch parts[2] {
		case "edit":
			switch r.Method {
			case http.MethodGet:
				h.editUserForm(w, r, id)
			case http.MethodPost:
				h.updateUser(w, r, id)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		case "delete":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			h.deleteUser(w, r, id)
			return
		}
	}

	if r.URL.Path == "/audit" && r.Method == http.MethodGet {
		h.audit(w, r)
		return
	}

	http.NotFound(w, r)
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	h.renderer.Render(w, http.StatusOK, "users.html", map[string]any{
		"Title":       "Benutzerverwaltung",
		"CurrentUser": user,
		"Users":       h.store.ListUsers(),
		"Error":       r.URL.Query().Get("error"),
	})
}

func (h *Handler) userForm(w http.ResponseWriter, r *http.Request, u model.User, message string, isNew bool) {
	user, _ := auth.UserFromContext(r.Context())
	title := "Benutzer anlegen"
	if !isNew {
		title = "Benutzer bearbeiten"
	}
	h.renderer.Render(w, http.StatusOK, "user_form.html", map[string]any{
		"Title":       title,
		"CurrentUser": user,
		"UserForm":    u,
		"Roles":       []string{model.RoleAdmin, model.RoleUser},
		"Error":       message,
		"IsNew":       isNew,
	})
}

func (h *Handler) editUserForm(w http.ResponseWriter, r *http.Request, id int) {
	u, ok := h.store.GetUserByID(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	h.userForm(w, r, u, "", false)
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	actor, _ := auth.UserFromContext(r.Context())
	u, password, err := userFromRequest(r, true)
	if err != nil {
		h.userForm(w, r, u, err.Error(), true)
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		h.userForm(w, r, u, "Passwort muss mindestens 8 Zeichen haben.", true)
		return
	}
	if _, err := h.store.CreateUserWithActor(actor.Username, u.Username, hash, u.Role, u.Active); err != nil {
		if errors.Is(err, store.ErrConflict) {
			h.userForm(w, r, u, "Benutzername existiert bereits.", true)
			return
		}
		h.userForm(w, r, u, err.Error(), true)
		return
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request, id int) {
	actor, _ := auth.UserFromContext(r.Context())
	u, password, err := userFromRequest(r, false)
	u.ID = id
	if err != nil {
		h.userForm(w, r, u, err.Error(), false)
		return
	}
	var hash string
	if strings.TrimSpace(password) != "" {
		var hashErr error
		hash, hashErr = auth.HashPassword(password)
		if hashErr != nil {
			h.userForm(w, r, u, "Neues Passwort muss mindestens 8 Zeichen haben.", false)
			return
		}
	}
	if err := h.store.UpdateUser(actor.Username, u, hash); err != nil {
		if errors.Is(err, store.ErrConflict) {
			h.userForm(w, r, u, "Benutzername existiert bereits.", false)
			return
		}
		h.userForm(w, r, u, err.Error(), false)
		return
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request, id int) {
	actor, _ := auth.UserFromContext(r.Context())
	if actor.ID == id {
		http.Redirect(w, r, "/users?error=Der eigene Benutzer kann nicht gelöscht werden.", http.StatusSeeOther)
		return
	}
	if err := h.store.DeleteUser(actor.Username, id); err != nil {
		http.Redirect(w, r, "/users?error=Benutzer konnte nicht gelöscht werden.", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func userFromRequest(r *http.Request, needPassword bool) (model.User, string, error) {
	if err := r.ParseForm(); err != nil {
		return model.User{}, "", err
	}
	u := model.User{
		Username: strings.TrimSpace(r.FormValue("username")),
		Role:     strings.TrimSpace(r.FormValue("role")),
		Active:   r.FormValue("active") == "1",
	}
	password := r.FormValue("password")
	if u.Username == "" {
		return u, password, errors.New("Benutzername ist Pflicht.")
	}
	if u.Role == "" {
		u.Role = model.RoleUser
	}
	if needPassword && strings.TrimSpace(password) == "" {
		return u, password, errors.New("Passwort ist Pflicht.")
	}
	return u, password, nil
}

func (h *Handler) audit(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	events := h.store.ListAudit()
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	if query != "" {
		filtered := events[:0]
		for _, ev := range events {
			text := strings.ToLower(ev.Actor + " " + ev.Action + " " + ev.Entity + " " + ev.Message)
			if strings.Contains(text, query) {
				filtered = append(filtered, ev)
			}
		}
		events = filtered
	}
	h.renderer.Render(w, http.StatusOK, "audit.html", map[string]any{
		"Title":       "Audit-Log",
		"CurrentUser": user,
		"Events":      events,
		"Query":       r.URL.Query().Get("q"),
	})
}
