package customer

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

	if r.URL.Path == "/customers" && r.Method == http.MethodGet {
		h.list(w, r)
		return
	}

	if len(parts) == 2 && parts[0] == "customers" && parts[1] == "new" {
		switch r.Method {
		case http.MethodGet:
			h.form(w, r, model.Customer{}, "")
		case http.MethodPost:
			h.create(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if len(parts) == 3 && parts[0] == "customers" {
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

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	h.renderer.Render(w, http.StatusOK, "customers.html", map[string]any{
		"Title":       "Kunden",
		"CurrentUser": user,
		"Customers":   h.store.ListCustomers(),
		"Error":       r.URL.Query().Get("error"),
	})
}

func (h *Handler) form(w http.ResponseWriter, r *http.Request, c model.Customer, message string) {
	user, _ := auth.UserFromContext(r.Context())
	title := "Kunde anlegen"
	if c.ID > 0 {
		title = "Kunde bearbeiten"
	}

	h.renderer.Render(w, http.StatusOK, "customer_form.html", map[string]any{
		"Title":       title,
		"CurrentUser": user,
		"Customer":    c,
		"Error":       message,
	})
}

func (h *Handler) editForm(w http.ResponseWriter, r *http.Request, id int) {
	c, ok := h.store.GetCustomer(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	h.form(w, r, c, "")
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	c, err := customerFromRequest(r)
	if err != nil {
		h.form(w, r, c, err.Error())
		return
	}

	if _, err := h.store.CreateCustomer(c); err != nil {
		h.form(w, r, c, err.Error())
		return
	}
	http.Redirect(w, r, "/customers", http.StatusSeeOther)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, id int) {
	c, err := customerFromRequest(r)
	c.ID = id
	if err != nil {
		h.form(w, r, c, err.Error())
		return
	}

	if err := h.store.UpdateCustomer(c); err != nil {
		h.form(w, r, c, err.Error())
		return
	}
	http.Redirect(w, r, "/customers", http.StatusSeeOther)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, id int) {
	err := h.store.DeleteCustomer(id)
	if errors.Is(err, store.ErrForbidden) {
		http.Redirect(w, r, "/customers?error=Kunde kann nicht gelöscht werden, weil Angebote oder Rechnungen existieren.", http.StatusSeeOther)
		return
	}
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		http.Redirect(w, r, "/customers?error=Löschen fehlgeschlagen.", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/customers", http.StatusSeeOther)
}

func customerFromRequest(r *http.Request) (model.Customer, error) {
	if err := r.ParseForm(); err != nil {
		return model.Customer{}, err
	}

	c := model.Customer{
		Company:     strings.TrimSpace(r.FormValue("company")),
		ContactName: strings.TrimSpace(r.FormValue("contact_name")),
		Email:       strings.TrimSpace(r.FormValue("email")),
		Phone:       strings.TrimSpace(r.FormValue("phone")),
		Street:      strings.TrimSpace(r.FormValue("street")),
		ZIP:         strings.TrimSpace(r.FormValue("zip")),
		City:        strings.TrimSpace(r.FormValue("city")),
		Note:        strings.TrimSpace(r.FormValue("note")),
	}

	if c.Company == "" {
		return c, errors.New("Firma ist Pflicht.")
	}

	return c, nil
}
