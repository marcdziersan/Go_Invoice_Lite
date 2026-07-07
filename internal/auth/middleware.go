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
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		user, ok := st.GetUserByID(userID)
		if !ok {
			sessions.Destroy(w, r)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserFromContext(ctx context.Context) (model.User, bool) {
	user, ok := ctx.Value(userContextKey).(model.User)
	return user, ok
}
