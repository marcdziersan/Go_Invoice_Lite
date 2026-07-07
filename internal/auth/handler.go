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
		h.renderer.Render(w, http.StatusOK, "login.html", map[string]any{"Title": "Login"})
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
	if !ok || !user.Active || !VerifyPassword(password, user.PasswordHash) {
		h.renderer.Render(w, http.StatusUnauthorized, "login.html", map[string]any{
			"Title":    "Login",
			"Error":    "Benutzername oder Passwort ist falsch oder der Benutzer ist deaktiviert.",
			"Username": username,
		})
		return
	}

	if err := h.sessions.Create(w, user.ID); err != nil {
		http.Error(w, "session creation failed", http.StatusInternalServerError)
		return
	}
	_ = h.store.MarkUserLogin(user.ID)
	_ = h.store.AddAudit(user.Username, "login", "user", user.ID, "Benutzer angemeldet")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if user, ok := UserFromContext(r.Context()); ok {
		_ = h.store.AddAudit(user.Username, "logout", "user", user.ID, "Benutzer abgemeldet")
	}
	h.sessions.Destroy(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *Handler) Password(w http.ResponseWriter, r *http.Request) {
	user, _ := UserFromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		h.renderer.Render(w, http.StatusOK, "password.html", map[string]any{"Title": "Passwort ändern", "CurrentUser": user})
	case http.MethodPost:
		h.passwordPost(w, r, user.Username, user.ID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) passwordPost(w http.ResponseWriter, r *http.Request, actor string, userID int) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	current := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirm := r.FormValue("confirm_password")
	user, ok := h.store.GetUserByID(userID)
	if !ok || !VerifyPassword(current, user.PasswordHash) {
		h.renderer.Render(w, http.StatusBadRequest, "password.html", map[string]any{"Title": "Passwort ändern", "CurrentUser": user, "Error": "Das aktuelle Passwort ist falsch."})
		return
	}
	if newPassword != confirm {
		h.renderer.Render(w, http.StatusBadRequest, "password.html", map[string]any{"Title": "Passwort ändern", "CurrentUser": user, "Error": "Die neuen Passwörter stimmen nicht überein."})
		return
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		h.renderer.Render(w, http.StatusBadRequest, "password.html", map[string]any{"Title": "Passwort ändern", "CurrentUser": user, "Error": "Das neue Passwort muss mindestens 8 Zeichen haben."})
		return
	}
	if err := h.store.SetPassword(actor, userID, hash); err != nil {
		http.Error(w, "password update failed", http.StatusInternalServerError)
		return
	}
	h.renderer.Render(w, http.StatusOK, "password.html", map[string]any{"Title": "Passwort ändern", "CurrentUser": user, "Success": "Passwort wurde geändert."})
}
