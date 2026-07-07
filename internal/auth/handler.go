package auth

import (
	"net/http"
	"strings"

	"go-invoice-lite/internal/store"
	"go-invoice-lite/internal/web"
)

type Handler struct {
	store    *store.Store
	sessions *SessionStore
	renderer *web.Renderer
}

func NewHandler(st *store.Store, sessions *SessionStore, renderer *web.Renderer) *Handler {
	return &Handler{store: st, sessions: sessions, renderer: renderer}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.renderer.Render(w, http.StatusOK, "login.html", map[string]any{
			"Title": "Login",
		})
	case http.MethodPost:
		h.loginPost(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) loginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	user, ok := h.store.FindUserByUsername(username)
	if !ok || !VerifyPassword(password, user.PasswordHash) {
		h.renderer.Render(w, http.StatusUnauthorized, "login.html", map[string]any{
			"Title":    "Login",
			"Error":    "Benutzername oder Passwort ist falsch.",
			"Username": username,
		})
		return
	}

	if err := h.sessions.Create(w, user.ID); err != nil {
		http.Error(w, "session creation failed", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.sessions.Destroy(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
