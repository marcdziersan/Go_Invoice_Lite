package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"go-invoice-lite/internal/model"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("conflict")
	ErrForbidden = errors.New("forbidden")
)

type Data struct {
	Users          []model.User     `json:"users"`
	Customers      []model.Customer `json:"customers"`
	Quotes         []model.Quote    `json:"quotes"`
	Invoices       []model.Invoice  `json:"invoices"`
	NextUserID     int              `json:"next_user_id"`
	NextCustomerID int              `json:"next_customer_id"`
	NextQuoteID    int              `json:"next_quote_id"`
	NextInvoiceID  int              `json:"next_invoice_id"`
}

type Store struct {
	mu   sync.RWMutex
	path string
	data Data
}

func New(path string) (*Store, error) {
	s := &Store{path: path}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			s.data = Data{
				NextUserID:     1,
				NextCustomerID: 1,
				NextQuoteID:    1,
				NextInvoiceID:  1,
			}
			return s, s.saveLocked()
		}
		return nil, err
	}

	if len(raw) == 0 {
		return s, nil
	}

	if err := json.Unmarshal(raw, &s.data); err != nil {
		return nil, err
	}

	s.normalizeIDs()
	return s, nil
}

func (s *Store) normalizeIDs() {
	if s.data.NextUserID < 1 {
		s.data.NextUserID = 1
	}
	if s.data.NextCustomerID < 1 {
		s.data.NextCustomerID = 1
	}
	if s.data.NextQuoteID < 1 {
		s.data.NextQuoteID = 1
	}
	if s.data.NextInvoiceID < 1 {
		s.data.NextInvoiceID = 1
	}

	for _, item := range s.data.Users {
		if item.ID >= s.data.NextUserID {
			s.data.NextUserID = item.ID + 1
		}
	}
	for _, item := range s.data.Customers {
		if item.ID >= s.data.NextCustomerID {
			s.data.NextCustomerID = item.ID + 1
		}
	}
	for _, item := range s.data.Quotes {
		if item.ID >= s.data.NextQuoteID {
			s.data.NextQuoteID = item.ID + 1
		}
	}
	for _, item := range s.data.Invoices {
		if item.ID >= s.data.NextInvoiceID {
			s.data.NextInvoiceID = item.ID + 1
		}
	}
}

func (s *Store) saveLocked() error {
	tmp := s.path + ".tmp"
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, raw, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) UserCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data.Users)
}

func (s *Store) CreateUser(username, passwordHash, role string) (model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	username = strings.TrimSpace(username)
	if username == "" {
		return model.User{}, errors.New("username is required")
	}
	for _, user := range s.data.Users {
		if strings.EqualFold(user.Username, username) {
			return model.User{}, ErrConflict
		}
	}

	user := model.User{
		ID:           s.data.NextUserID,
		Username:     username,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    time.Now(),
	}
	s.data.NextUserID++
	s.data.Users = append(s.data.Users, user)
	return user, s.saveLocked()
}

func (s *Store) FindUserByUsername(username string) (model.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.data.Users {
		if strings.EqualFold(user.Username, username) {
			return user, true
		}
	}
	return model.User{}, false
}

func (s *Store) GetUserByID(id int) (model.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.data.Users {
		if user.ID == id {
			return user, true
		}
	}
	return model.User{}, false
}

func (s *Store) ListCustomers() []model.Customer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := append([]model.Customer(nil), s.data.Customers...)
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Company) < strings.ToLower(out[j].Company)
	})
	return out
}

func (s *Store) GetCustomer(id int) (model.Customer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range s.data.Customers {
		if item.ID == id {
			return item, true
		}
	}
	return model.Customer{}, false
}

func (s *Store) CreateCustomer(c model.Customer) (model.Customer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c.Company = strings.TrimSpace(c.Company)
	if c.Company == "" {
		return model.Customer{}, errors.New("company is required")
	}
	c.ID = s.data.NextCustomerID
	c.CreatedAt = time.Now()
	s.data.NextCustomerID++
	s.data.Customers = append(s.data.Customers, c)
	return c, s.saveLocked()
}

func (s *Store) UpdateCustomer(c model.Customer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	c.Company = strings.TrimSpace(c.Company)
	if c.Company == "" {
		return errors.New("company is required")
	}

	for i := range s.data.Customers {
		if s.data.Customers[i].ID == c.ID {
			c.CreatedAt = s.data.Customers[i].CreatedAt
			s.data.Customers[i] = c
			return s.saveLocked()
		}
	}
	return ErrNotFound
}

func (s *Store) DeleteCustomer(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, q := range s.data.Quotes {
		if q.CustomerID == id {
			return fmt.Errorf("%w: customer has quotes", ErrForbidden)
		}
	}
	for _, inv := range s.data.Invoices {
		if inv.CustomerID == id {
			return fmt.Errorf("%w: customer has invoices", ErrForbidden)
		}
	}

	for i := range s.data.Customers {
		if s.data.Customers[i].ID == id {
			s.data.Customers = append(s.data.Customers[:i], s.data.Customers[i+1:]...)
			return s.saveLocked()
		}
	}
	return ErrNotFound
}

func (s *Store) ListQuotes() []model.Quote {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := append([]model.Quote(nil), s.data.Quotes...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out
}

func (s *Store) GetQuote(id int) (model.Quote, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range s.data.Quotes {
		if item.ID == id {
			return item, true
		}
	}
	return model.Quote{}, false
}

func (s *Store) CreateQuote(q model.Quote) (model.Quote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.customerExistsLocked(q.CustomerID) {
		return model.Quote{}, errors.New("customer does not exist")
	}
	q.Title = strings.TrimSpace(q.Title)
	if q.Title == "" {
		return model.Quote{}, errors.New("title is required")
	}
	if q.TaxRate == 0 {
		q.TaxRate = 19
	}
	if q.Status == "" {
		q.Status = "draft"
	}
	q.ID = s.data.NextQuoteID
	q.Number = fmt.Sprintf("Q-%d-%04d", time.Now().Year(), q.ID)
	q.GrossAmount = model.CalculateGross(q.NetAmount, q.TaxRate)
	q.CreatedAt = time.Now()

	s.data.NextQuoteID++
	s.data.Quotes = append(s.data.Quotes, q)
	return q, s.saveLocked()
}

func (s *Store) UpdateQuote(q model.Quote) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.customerExistsLocked(q.CustomerID) {
		return errors.New("customer does not exist")
	}
	q.Title = strings.TrimSpace(q.Title)
	if q.Title == "" {
		return errors.New("title is required")
	}
	if q.TaxRate == 0 {
		q.TaxRate = 19
	}
	q.GrossAmount = model.CalculateGross(q.NetAmount, q.TaxRate)

	for i := range s.data.Quotes {
		if s.data.Quotes[i].ID == q.ID {
			q.Number = s.data.Quotes[i].Number
			q.CreatedAt = s.data.Quotes[i].CreatedAt
			s.data.Quotes[i] = q
			return s.saveLocked()
		}
	}
	return ErrNotFound
}

func (s *Store) DeleteQuote(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, inv := range s.data.Invoices {
		if inv.QuoteID == id {
			return fmt.Errorf("%w: quote already converted to invoice", ErrForbidden)
		}
	}

	for i := range s.data.Quotes {
		if s.data.Quotes[i].ID == id {
			s.data.Quotes = append(s.data.Quotes[:i], s.data.Quotes[i+1:]...)
			return s.saveLocked()
		}
	}
	return ErrNotFound
}

func (s *Store) ConvertQuoteToInvoice(quoteID int) (model.Invoice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var quote model.Quote
	found := false
	for _, q := range s.data.Quotes {
		if q.ID == quoteID {
			quote = q
			found = true
			break
		}
	}
	if !found {
		return model.Invoice{}, ErrNotFound
	}
	for _, inv := range s.data.Invoices {
		if inv.QuoteID == quoteID {
			return model.Invoice{}, fmt.Errorf("%w: invoice already exists for quote", ErrConflict)
		}
	}

	now := time.Now()
	invoice := model.Invoice{
		ID:          s.data.NextInvoiceID,
		Number:      fmt.Sprintf("INV-%d-%04d", now.Year(), s.data.NextInvoiceID),
		CustomerID:  quote.CustomerID,
		QuoteID:     quote.ID,
		Title:       quote.Title,
		Description: quote.Description,
		NetAmount:   quote.NetAmount,
		TaxRate:     quote.TaxRate,
		GrossAmount: quote.GrossAmount,
		Status:      "open",
		InvoiceDate: now.Format("2006-01-02"),
		DueDate:     now.AddDate(0, 0, 14).Format("2006-01-02"),
		CreatedAt:   now,
	}

	s.data.NextInvoiceID++
	s.data.Invoices = append(s.data.Invoices, invoice)
	return invoice, s.saveLocked()
}

func (s *Store) ListInvoices() []model.Invoice {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := append([]model.Invoice(nil), s.data.Invoices...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out
}

func (s *Store) GetInvoice(id int) (model.Invoice, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range s.data.Invoices {
		if item.ID == id {
			return item, true
		}
	}
	return model.Invoice{}, false
}

func (s *Store) CreateInvoice(inv model.Invoice) (model.Invoice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.customerExistsLocked(inv.CustomerID) {
		return model.Invoice{}, errors.New("customer does not exist")
	}
	inv.Title = strings.TrimSpace(inv.Title)
	if inv.Title == "" {
		return model.Invoice{}, errors.New("title is required")
	}
	if inv.TaxRate == 0 {
		inv.TaxRate = 19
	}
	if inv.Status == "" {
		inv.Status = "open"
	}
	inv.ID = s.data.NextInvoiceID
	inv.Number = fmt.Sprintf("INV-%d-%04d", time.Now().Year(), inv.ID)
	inv.GrossAmount = model.CalculateGross(inv.NetAmount, inv.TaxRate)
	inv.CreatedAt = time.Now()

	s.data.NextInvoiceID++
	s.data.Invoices = append(s.data.Invoices, inv)
	return inv, s.saveLocked()
}

func (s *Store) UpdateInvoice(inv model.Invoice) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.customerExistsLocked(inv.CustomerID) {
		return errors.New("customer does not exist")
	}
	inv.Title = strings.TrimSpace(inv.Title)
	if inv.Title == "" {
		return errors.New("title is required")
	}
	if inv.TaxRate == 0 {
		inv.TaxRate = 19
	}
	inv.GrossAmount = model.CalculateGross(inv.NetAmount, inv.TaxRate)

	for i := range s.data.Invoices {
		if s.data.Invoices[i].ID == inv.ID {
			inv.Number = s.data.Invoices[i].Number
			inv.QuoteID = s.data.Invoices[i].QuoteID
			inv.CreatedAt = s.data.Invoices[i].CreatedAt
			s.data.Invoices[i] = inv
			return s.saveLocked()
		}
	}
	return ErrNotFound
}

func (s *Store) DeleteInvoice(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.Invoices {
		if s.data.Invoices[i].ID == id {
			s.data.Invoices = append(s.data.Invoices[:i], s.data.Invoices[i+1:]...)
			return s.saveLocked()
		}
	}
	return ErrNotFound
}

type DashboardStats struct {
	Customers        int
	Quotes           int
	Invoices         int
	OpenInvoices     int
	PaidInvoices     int
	RevenueGross     int64
	OutstandingGross int64
	RecentQuotes     []model.Quote
	RecentInvoices   []model.Invoice
}

func (s *Store) DashboardStats() DashboardStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := DashboardStats{
		Customers: len(s.data.Customers),
		Quotes:    len(s.data.Quotes),
		Invoices:  len(s.data.Invoices),
	}
	for _, inv := range s.data.Invoices {
		switch inv.Status {
		case "paid":
			stats.PaidInvoices++
			stats.RevenueGross += inv.GrossAmount
		case "open", "overdue":
			stats.OpenInvoices++
			stats.OutstandingGross += inv.GrossAmount
		}
	}

	quotes := append([]model.Quote(nil), s.data.Quotes...)
	sort.Slice(quotes, func(i, j int) bool { return quotes[i].ID > quotes[j].ID })
	if len(quotes) > 5 {
		quotes = quotes[:5]
	}
	stats.RecentQuotes = quotes

	invoices := append([]model.Invoice(nil), s.data.Invoices...)
	sort.Slice(invoices, func(i, j int) bool { return invoices[i].ID > invoices[j].ID })
	if len(invoices) > 5 {
		invoices = invoices[:5]
	}
	stats.RecentInvoices = invoices

	return stats
}

func (s *Store) CustomerName(id int) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, c := range s.data.Customers {
		if c.ID == id {
			return c.Company
		}
	}
	return "Unbekannter Kunde"
}

func (s *Store) customerExistsLocked(id int) bool {
	for _, c := range s.data.Customers {
		if c.ID == id {
			return true
		}
	}
	return false
}
