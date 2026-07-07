package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"go-invoice-lite/internal/auth"
	"go-invoice-lite/internal/model"
	"go-invoice-lite/internal/store"
)

type Handler struct {
	store *store.Store
}

func NewHandler(st *store.Store) *Handler {
	return &Handler{store: st}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusOK, map[string]any{"name": "GoInvoice Lite API", "version": 1})
		return
	}

	switch parts[0] {
	case "customers":
		h.customers(w, r, parts)
	case "quotes":
		h.quotes(w, r, parts)
	case "invoices":
		h.invoices(w, r, parts)
	case "audit":
		user, _ := auth.UserFromContext(r.Context())
		if !user.IsAdmin() {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		writeJSON(w, http.StatusOK, h.store.ListAudit())
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *Handler) customers(w http.ResponseWriter, r *http.Request, parts []string) {
	actor, _ := auth.UserFromContext(r.Context())
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, h.store.ListCustomers())
		case http.MethodPost:
			var c model.Customer
			if !decodeJSON(w, r, &c) {
				return
			}
			created, err := h.store.CreateCustomerWithActor(actor.Username, c)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, created)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	id, ok := parseID(w, parts[1])
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		c, ok := h.store.GetCustomer(id)
		if !ok {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSON(w, http.StatusOK, c)
	case http.MethodPut:
		var c model.Customer
		if !decodeJSON(w, r, &c) {
			return
		}
		c.ID = id
		if err := h.store.UpdateCustomerWithActor(actor.Username, c); err != nil {
			writeStoreError(w, err)
			return
		}
		updated, _ := h.store.GetCustomer(id)
		writeJSON(w, http.StatusOK, updated)
	case http.MethodDelete:
		if err := h.store.DeleteCustomerWithActor(actor.Username, id); err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) quotes(w http.ResponseWriter, r *http.Request, parts []string) {
	actor, _ := auth.UserFromContext(r.Context())
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, h.store.ListQuotes())
		case http.MethodPost:
			var q model.Quote
			if !decodeJSON(w, r, &q) {
				return
			}
			created, err := h.store.CreateQuoteWithActor(actor.Username, q)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, created)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	id, ok := parseID(w, parts[1])
	if !ok {
		return
	}
	if len(parts) == 3 && parts[2] == "convert" && r.Method == http.MethodPost {
		inv, err := h.store.ConvertQuoteToInvoiceWithActor(actor.Username, id)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, inv)
		return
	}
	switch r.Method {
	case http.MethodGet:
		q, ok := h.store.GetQuote(id)
		if !ok {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSON(w, http.StatusOK, q)
	case http.MethodPut:
		var q model.Quote
		if !decodeJSON(w, r, &q) {
			return
		}
		q.ID = id
		if err := h.store.UpdateQuoteWithActor(actor.Username, q); err != nil {
			writeStoreError(w, err)
			return
		}
		updated, _ := h.store.GetQuote(id)
		writeJSON(w, http.StatusOK, updated)
	case http.MethodDelete:
		if err := h.store.DeleteQuoteWithActor(actor.Username, id); err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) invoices(w http.ResponseWriter, r *http.Request, parts []string) {
	actor, _ := auth.UserFromContext(r.Context())
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, h.store.ListInvoices())
		case http.MethodPost:
			var inv model.Invoice
			if !decodeJSON(w, r, &inv) {
				return
			}
			created, err := h.store.CreateInvoiceWithActor(actor.Username, inv)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, created)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	id, ok := parseID(w, parts[1])
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		inv, ok := h.store.GetInvoice(id)
		if !ok {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSON(w, http.StatusOK, inv)
	case http.MethodPut:
		var inv model.Invoice
		if !decodeJSON(w, r, &inv) {
			return
		}
		inv.ID = id
		if err := h.store.UpdateInvoiceWithActor(actor.Username, inv); err != nil {
			writeStoreError(w, err)
			return
		}
		updated, _ := h.store.GetInvoice(id)
		writeJSON(w, http.StatusOK, updated)
	case http.MethodDelete:
		if err := h.store.DeleteInvoiceWithActor(actor.Username, id); err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return false
	}
	return true
}

func parseID(w http.ResponseWriter, raw string) (int, bool) {
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, store.ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, store.ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
