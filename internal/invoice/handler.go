package invoice

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-invoice-lite/internal/auth"
	"go-invoice-lite/internal/model"
	"go-invoice-lite/internal/pdf"
	"go-invoice-lite/internal/store"
	"go-invoice-lite/internal/web"
)

var statuses = []string{"open", "paid", "overdue", "reminded", "cancelled"}

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
		case "pdf":
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			h.exportPDF(w, r, id)
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
		Items: []model.LineItem{{
			ID:       1,
			Quantity: 1,
			TaxRate:  19,
		}},
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	customerID, _ := strconv.Atoi(r.URL.Query().Get("customer_id"))
	dunningLevel, _ := strconv.Atoi(r.URL.Query().Get("dunning_level"))
	invoices := h.store.ListInvoices()
	views := make([]InvoiceView, 0, len(invoices))
	for _, inv := range invoices {
		customerName := h.store.CustomerName(inv.CustomerID)
		text := strings.ToLower(inv.Number + " " + inv.Title + " " + inv.Description + " " + customerName)
		if query != "" && !strings.Contains(text, query) {
			continue
		}
		if status != "" && inv.Status != status {
			continue
		}
		if customerID > 0 && inv.CustomerID != customerID {
			continue
		}
		if dunningLevel > 0 && inv.DunningLevel != dunningLevel {
			continue
		}
		views = append(views, InvoiceView{Invoice: inv, CustomerName: customerName})
	}

	h.renderer.Render(w, http.StatusOK, "invoices.html", map[string]any{
		"Title":              "Rechnungen",
		"CurrentUser":        user,
		"Invoices":           views,
		"Customers":          h.store.ListCustomers(),
		"Statuses":           statuses,
		"Query":              r.URL.Query().Get("q"),
		"FilterStatus":       status,
		"FilterCustomerID":   customerID,
		"FilterDunningLevel": dunningLevel,
		"Error":              r.URL.Query().Get("error"),
	})
}

func (h *Handler) form(w http.ResponseWriter, r *http.Request, inv model.Invoice, message string) {
	user, _ := auth.UserFromContext(r.Context())
	title := "Rechnung anlegen"
	if inv.ID > 0 {
		title = "Rechnung bearbeiten"
	}
	if len(inv.Items) == 0 {
		inv.Items = defaultInvoice().Items
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
	user, _ := auth.UserFromContext(r.Context())
	inv, err := invoiceFromRequest(r)
	if err != nil {
		h.form(w, r, inv, err.Error())
		return
	}
	if _, err := h.store.CreateInvoiceWithActor(user.Username, inv); err != nil {
		h.form(w, r, inv, err.Error())
		return
	}
	http.Redirect(w, r, "/invoices", http.StatusSeeOther)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, id int) {
	user, _ := auth.UserFromContext(r.Context())
	inv, err := invoiceFromRequest(r)
	inv.ID = id
	if err != nil {
		h.form(w, r, inv, err.Error())
		return
	}
	if err := h.store.UpdateInvoiceWithActor(user.Username, inv); err != nil {
		h.form(w, r, inv, err.Error())
		return
	}
	http.Redirect(w, r, "/invoices", http.StatusSeeOther)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, id int) {
	user, _ := auth.UserFromContext(r.Context())
	if err := h.store.DeleteInvoiceWithActor(user.Username, id); err != nil && !errors.Is(err, store.ErrNotFound) {
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
	dunningLevel, _ := strconv.Atoi(r.FormValue("dunning_level"))
	inv := model.Invoice{
		CustomerID:   customerID,
		Title:        strings.TrimSpace(r.FormValue("title")),
		Description:  strings.TrimSpace(r.FormValue("description")),
		Status:       strings.TrimSpace(r.FormValue("status")),
		InvoiceDate:  strings.TrimSpace(r.FormValue("invoice_date")),
		DueDate:      strings.TrimSpace(r.FormValue("due_date")),
		PaymentDate:  strings.TrimSpace(r.FormValue("payment_date")),
		DunningLevel: dunningLevel,
		Items:        lineItemsFromRequest(r),
	}
	if inv.Status == "" {
		inv.Status = "open"
	}
	if inv.PaymentDate != "" {
		inv.Status = "paid"
	}
	if inv.CustomerID <= 0 {
		return inv, errors.New("Kunde ist Pflicht.")
	}
	if inv.Title == "" {
		return inv, errors.New("Titel ist Pflicht.")
	}
	if inv.InvoiceDate == "" {
		return inv, errors.New("Rechnungsdatum ist Pflicht.")
	}
	if inv.DueDate == "" {
		return inv, errors.New("Fälligkeitsdatum ist Pflicht.")
	}
	if len(model.NormalizeLineItems(inv.Items)) == 0 {
		return inv, errors.New("Mindestens eine Position ist Pflicht.")
	}
	return inv, nil
}

func lineItemsFromRequest(r *http.Request) []model.LineItem {
	descriptions := r.Form["item_description"]
	quantities := r.Form["item_quantity"]
	amounts := r.Form["item_unit_net"]
	taxes := r.Form["item_tax_rate"]
	max := len(descriptions)
	if len(quantities) > max {
		max = len(quantities)
	}
	if len(amounts) > max {
		max = len(amounts)
	}
	if len(taxes) > max {
		max = len(taxes)
	}
	items := make([]model.LineItem, 0, max)
	for i := 0; i < max; i++ {
		desc := getFormIndex(descriptions, i)
		qty, _ := strconv.ParseInt(strings.TrimSpace(getFormIndex(quantities, i)), 10, 64)
		unit, _ := web.ParseMoney(getFormIndex(amounts, i))
		tax, _ := strconv.Atoi(strings.TrimSpace(getFormIndex(taxes, i)))
		items = append(items, model.LineItem{Description: desc, Quantity: qty, UnitNetAmount: unit, TaxRate: tax})
	}
	return items
}

func getFormIndex(values []string, index int) string {
	if index < 0 || index >= len(values) {
		return ""
	}
	return strings.TrimSpace(values[index])
}

func (h *Handler) exportCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="invoices.csv"`)

	writer := csv.NewWriter(w)
	defer writer.Flush()

	_ = writer.Write([]string{"Rechnungsnummer", "Kunde", "Titel", "Status", "Netto Cent", "MwSt Cent", "Brutto Cent", "Rechnungsdatum", "Faelligkeitsdatum", "Zahlungsdatum", "Mahnstufe"})
	for _, inv := range h.store.ListInvoices() {
		_ = writer.Write([]string{inv.Number, h.store.CustomerName(inv.CustomerID), inv.Title, inv.Status, strconv.FormatInt(inv.NetAmount, 10), strconv.FormatInt(inv.TaxAmount, 10), strconv.FormatInt(inv.GrossAmount, 10), inv.InvoiceDate, inv.DueDate, inv.PaymentDate, strconv.Itoa(inv.DunningLevel)})
	}
}

func (h *Handler) exportPDF(w http.ResponseWriter, r *http.Request, id int) {
	inv, ok := h.store.GetInvoice(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, inv.Number))
	if err := pdf.WriteInvoice(w, inv, h.store.CustomerName(inv.CustomerID)); err != nil {
		http.Error(w, "pdf export failed", http.StatusInternalServerError)
	}
}
