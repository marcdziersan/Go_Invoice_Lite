package invoice

import (
	"encoding/csv"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-invoice-lite/internal/auth"
	"go-invoice-lite/internal/model"
	"go-invoice-lite/internal/store"
	"go-invoice-lite/internal/web"
)

var statuses = []string{"open", "paid", "overdue", "cancelled"}

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
	if r.URL.Path == "/invoices/export.csv" && r.Method == http.MethodGet {
		h.exportCSV(w, r)
		return
	}

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if r.URL.Path == "/invoices" && r.Method == http.MethodGet {
		h.list(w, r)
		return
	}

	if len(parts) == 2 && parts[0] == "invoices" && parts[1] == "new" {
		switch r.Method {
		case http.MethodGet:
			h.form(w, r, defaultInvoice(), "")
		case http.MethodPost:
			h.create(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if len(parts) == 3 && parts[0] == "invoices" {
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
		}
	}

	http.NotFound(w, r)
}

func defaultInvoice() model.Invoice {
	now := time.Now()
	return model.Invoice{
		TaxRate:     19,
		Status:      "open",
		InvoiceDate: now.Format("2006-01-02"),
		DueDate:     now.AddDate(0, 0, 14).Format("2006-01-02"),
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	invoices := h.store.ListInvoices()
	views := make([]InvoiceView, 0, len(invoices))
	for _, inv := range invoices {
		views = append(views, InvoiceView{
			Invoice:      inv,
			CustomerName: h.store.CustomerName(inv.CustomerID),
		})
	}

	h.renderer.Render(w, http.StatusOK, "invoices.html", map[string]any{
		"Title":       "Rechnungen",
		"CurrentUser": user,
		"Invoices":    views,
		"Error":       r.URL.Query().Get("error"),
	})
}

func (h *Handler) form(w http.ResponseWriter, r *http.Request, inv model.Invoice, message string) {
	user, _ := auth.UserFromContext(r.Context())
	title := "Rechnung anlegen"
	if inv.ID > 0 {
		title = "Rechnung bearbeiten"
	}

	h.renderer.Render(w, http.StatusOK, "invoice_form.html", map[string]any{
		"Title":       title,
		"CurrentUser": user,
		"Invoice":     inv,
		"Customers":   h.store.ListCustomers(),
		"Statuses":    statuses,
		"Error":       message,
	})
}

func (h *Handler) editForm(w http.ResponseWriter, r *http.Request, id int) {
	inv, ok := h.store.GetInvoice(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	h.form(w, r, inv, "")
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	inv, err := invoiceFromRequest(r)
	if err != nil {
		h.form(w, r, inv, err.Error())
		return
	}
	if _, err := h.store.CreateInvoice(inv); err != nil {
		h.form(w, r, inv, err.Error())
		return
	}
	http.Redirect(w, r, "/invoices", http.StatusSeeOther)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, id int) {
	inv, err := invoiceFromRequest(r)
	inv.ID = id
	if err != nil {
		h.form(w, r, inv, err.Error())
		return
	}
	if err := h.store.UpdateInvoice(inv); err != nil {
		h.form(w, r, inv, err.Error())
		return
	}
	http.Redirect(w, r, "/invoices", http.StatusSeeOther)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, id int) {
	if err := h.store.DeleteInvoice(id); err != nil && !errors.Is(err, store.ErrNotFound) {
		http.Redirect(w, r, "/invoices?error=Löschen fehlgeschlagen.", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/invoices", http.StatusSeeOther)
}

func invoiceFromRequest(r *http.Request) (model.Invoice, error) {
	if err := r.ParseForm(); err != nil {
		return model.Invoice{}, err
	}

	customerID, _ := strconv.Atoi(r.FormValue("customer_id"))
	taxRate, _ := strconv.Atoi(r.FormValue("tax_rate"))
	if taxRate == 0 {
		taxRate = 19
	}
	netAmount, err := web.ParseMoney(r.FormValue("net_amount"))
	if err != nil {
		return model.Invoice{}, errors.New("Netto-Betrag ist ungültig.")
	}

	inv := model.Invoice{
		CustomerID:  customerID,
		Title:       strings.TrimSpace(r.FormValue("title")),
		Description: strings.TrimSpace(r.FormValue("description")),
		NetAmount:   netAmount,
		TaxRate:     taxRate,
		Status:      strings.TrimSpace(r.FormValue("status")),
		InvoiceDate: strings.TrimSpace(r.FormValue("invoice_date")),
		DueDate:     strings.TrimSpace(r.FormValue("due_date")),
	}
	if inv.Status == "" {
		inv.Status = "open"
	}

	if inv.CustomerID <= 0 {
		return inv, errors.New("Kunde ist Pflicht.")
	}
	if inv.Title == "" {
		return inv, errors.New("Titel ist Pflicht.")
	}
	if inv.NetAmount < 0 {
		return inv, errors.New("Netto-Betrag darf nicht negativ sein.")
	}
	if inv.InvoiceDate == "" {
		return inv, errors.New("Rechnungsdatum ist Pflicht.")
	}
	if inv.DueDate == "" {
		return inv, errors.New("Fälligkeitsdatum ist Pflicht.")
	}

	return inv, nil
}

func (h *Handler) exportCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="invoices.csv"`)

	writer := csv.NewWriter(w)
	defer writer.Flush()

	_ = writer.Write([]string{
		"Rechnungsnummer",
		"Kunde",
		"Titel",
		"Status",
		"Netto Cent",
		"MwSt",
		"Brutto Cent",
		"Rechnungsdatum",
		"Faelligkeitsdatum",
	})

	for _, inv := range h.store.ListInvoices() {
		_ = writer.Write([]string{
			inv.Number,
			h.store.CustomerName(inv.CustomerID),
			inv.Title,
			inv.Status,
			strconv.FormatInt(inv.NetAmount, 10),
			strconv.Itoa(inv.TaxRate),
			strconv.FormatInt(inv.GrossAmount, 10),
			inv.InvoiceDate,
			inv.DueDate,
		})
	}
}
