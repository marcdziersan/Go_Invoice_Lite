package app

import (
	"net/http"

	"go-invoice-lite/internal/auth"
	"go-invoice-lite/internal/customer"
	"go-invoice-lite/internal/dashboard"
	"go-invoice-lite/internal/invoice"
	"go-invoice-lite/internal/quote"
	"go-invoice-lite/internal/store"
	"go-invoice-lite/internal/web"
)

func New(st *store.Store) http.Handler {
	renderer := web.NewRenderer("web/templates")
	sessions := auth.NewSessionStore()

	authHandler := auth.NewHandler(st, sessions, renderer)
	dashboardHandler := dashboard.NewHandler(st, renderer)
	customerHandler := customer.NewHandler(st, renderer)
	quoteHandler := quote.NewHandler(st, renderer)
	invoiceHandler := invoice.NewHandler(st, renderer)

	mux := http.NewServeMux()

	static := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", static))

	mux.HandleFunc("/login", authHandler.Login)
	mux.HandleFunc("/logout", authHandler.Logout)

	mux.Handle("/customers", auth.RequireLogin(st, sessions, customerHandler))
	mux.Handle("/customers/", auth.RequireLogin(st, sessions, customerHandler))

	mux.Handle("/quotes", auth.RequireLogin(st, sessions, quoteHandler))
	mux.Handle("/quotes/", auth.RequireLogin(st, sessions, quoteHandler))

	mux.Handle("/invoices", auth.RequireLogin(st, sessions, invoiceHandler))
	mux.Handle("/invoices/", auth.RequireLogin(st, sessions, invoiceHandler))

	mux.Handle("/", auth.RequireLogin(st, sessions, dashboardHandler))

	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
