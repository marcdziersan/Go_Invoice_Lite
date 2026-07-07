package dashboard

import (
	"net/http"

	"go-invoice-lite/internal/auth"
	"go-invoice-lite/internal/model"
	"go-invoice-lite/internal/store"
	"go-invoice-lite/internal/web"
)

type QuoteView struct {
	model.Quote
	CustomerName string
}

type InvoiceView struct {
	model.Invoice
	CustomerName string
}

type Handler struct {
	store    *store.Store
	renderer *web.Renderer
}

func NewHandler(st *store.Store, renderer *web.Renderer) *Handler {
	return &Handler{store: st, renderer: renderer}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	user, _ := auth.UserFromContext(r.Context())
	stats := h.store.DashboardStats()

	recentQuotes := make([]QuoteView, 0, len(stats.RecentQuotes))
	for _, q := range stats.RecentQuotes {
		recentQuotes = append(recentQuotes, QuoteView{
			Quote:        q,
			CustomerName: h.store.CustomerName(q.CustomerID),
		})
	}

	recentInvoices := make([]InvoiceView, 0, len(stats.RecentInvoices))
	for _, inv := range stats.RecentInvoices {
		recentInvoices = append(recentInvoices, InvoiceView{
			Invoice:      inv,
			CustomerName: h.store.CustomerName(inv.CustomerID),
		})
	}

	h.renderer.Render(w, http.StatusOK, "dashboard.html", map[string]any{
		"Title":          "Dashboard",
		"CurrentUser":    user,
		"Stats":          stats,
		"RecentQuotes":   recentQuotes,
		"RecentInvoices": recentInvoices,
	})
}
