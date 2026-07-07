package quote

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

var statuses = []string{"draft", "sent", "accepted", "rejected"}

type QuoteView struct {
	model.Quote
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
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if r.URL.Path == "/quotes" && r.Method == http.MethodGet {
		h.list(w, r)
		return
	}

	if len(parts) == 2 && parts[0] == "quotes" && parts[1] == "new" {
		switch r.Method {
		case http.MethodGet:
			h.form(w, r, defaultQuote(), "")
		case http.MethodPost:
			h.create(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if len(parts) == 3 && parts[0] == "quotes" {
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		switch parts[2] {
		case "edit":
			switch r.Method {
			case http.MethodGet:
				h.editForm(w, r, id)
			case http.MethodPost:
				h.update(w, r, id)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		case "delete":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			h.delete(w, r, id)
			return
		case "convert":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			h.convert(w, r, id)
			return
		}
	}

	http.NotFound(w, r)
}

func defaultQuote() model.Quote {
	return model.Quote{
		TaxRate: 19,
		Status:  "draft",
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	quotes := h.store.ListQuotes()
	views := make([]QuoteView, 0, len(quotes))
	for _, q := range quotes {
		views = append(views, QuoteView{
			Quote:        q,
			CustomerName: h.store.CustomerName(q.CustomerID),
		})
	}

	h.renderer.Render(w, http.StatusOK, "quotes.html", map[string]any{
		"Title":       "Angebote",
		"CurrentUser": user,
		"Quotes":      views,
		"Error":       r.URL.Query().Get("error"),
	})
}

func (h *Handler) form(w http.ResponseWriter, r *http.Request, q model.Quote, message string) {
	user, _ := auth.UserFromContext(r.Context())
	title := "Angebot anlegen"
	if q.ID > 0 {
		title = "Angebot bearbeiten"
	}

	h.renderer.Render(w, http.StatusOK, "quote_form.html", map[string]any{
		"Title":       title,
		"CurrentUser": user,
		"Quote":       q,
		"Customers":   h.store.ListCustomers(),
		"Statuses":    statuses,
		"Error":       message,
	})
}

func (h *Handler) editForm(w http.ResponseWriter, r *http.Request, id int) {
	q, ok := h.store.GetQuote(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	h.form(w, r, q, "")
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	q, err := quoteFromRequest(r)
	if err != nil {
		h.form(w, r, q, err.Error())
		return
	}
	if _, err := h.store.CreateQuote(q); err != nil {
		h.form(w, r, q, err.Error())
		return
	}
	http.Redirect(w, r, "/quotes", http.StatusSeeOther)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, id int) {
	q, err := quoteFromRequest(r)
	q.ID = id
	if err != nil {
		h.form(w, r, q, err.Error())
		return
	}
	if err := h.store.UpdateQuote(q); err != nil {
		h.form(w, r, q, err.Error())
		return
	}
	http.Redirect(w, r, "/quotes", http.StatusSeeOther)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, id int) {
	err := h.store.DeleteQuote(id)
	if errors.Is(err, store.ErrForbidden) {
		http.Redirect(w, r, "/quotes?error=Angebot kann nicht gelöscht werden, weil daraus bereits eine Rechnung erstellt wurde.", http.StatusSeeOther)
		return
	}
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		http.Redirect(w, r, "/quotes?error=Löschen fehlgeschlagen.", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/quotes", http.StatusSeeOther)
}

func (h *Handler) convert(w http.ResponseWriter, r *http.Request, id int) {
	_, err := h.store.ConvertQuoteToInvoice(id)
	if errors.Is(err, store.ErrConflict) {
		http.Redirect(w, r, "/quotes?error=Für dieses Angebot existiert bereits eine Rechnung.", http.StatusSeeOther)
		return
	}
	if err != nil {
		http.Redirect(w, r, "/quotes?error=Umwandlung fehlgeschlagen.", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/invoices", http.StatusSeeOther)
}

func quoteFromRequest(r *http.Request) (model.Quote, error) {
	if err := r.ParseForm(); err != nil {
		return model.Quote{}, err
	}

	customerID, _ := strconv.Atoi(r.FormValue("customer_id"))
	taxRate, _ := strconv.Atoi(r.FormValue("tax_rate"))
	if taxRate == 0 {
		taxRate = 19
	}
	netAmount, err := web.ParseMoney(r.FormValue("net_amount"))
	if err != nil {
		return model.Quote{}, errors.New("Netto-Betrag ist ungültig.")
	}

	q := model.Quote{
		CustomerID:  customerID,
		Title:       strings.TrimSpace(r.FormValue("title")),
		Description: strings.TrimSpace(r.FormValue("description")),
		NetAmount:   netAmount,
		TaxRate:     taxRate,
		Status:      strings.TrimSpace(r.FormValue("status")),
		ValidUntil:  strings.TrimSpace(r.FormValue("valid_until")),
	}
	if q.Status == "" {
		q.Status = "draft"
	}

	if q.CustomerID <= 0 {
		return q, errors.New("Kunde ist Pflicht.")
	}
	if q.Title == "" {
		return q, errors.New("Titel ist Pflicht.")
	}
	if q.NetAmount < 0 {
		return q, errors.New("Netto-Betrag darf nicht negativ sein.")
	}

	return q, nil
}
