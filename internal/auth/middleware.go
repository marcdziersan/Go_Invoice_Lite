package auth

import (
	"context"
	"net/http"

	"go-invoice-lite/internal/model"
	"go-invoice-lite/internal/store"
)

type contextKey string

const userContextKey contextKey = "currentUser"

func RequireLogin(st *store.Store, sessions *SessionStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := sessions.UserID(r)
		if !ok {
			redirectLogin(w, r)
			return
		}

		user, ok := st.GetUserByID(userID)
		if !ok || !user.Active {
			sessions.Destroy(w, r)
			redirectLogin(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok || !user.IsAdmin() {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func UserFromContext(ctx context.Context) (model.User, bool) {
	user, ok := ctx.Value(userContextKey).(model.User)
	return user, ok
}

func redirectLogin(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api" || len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
