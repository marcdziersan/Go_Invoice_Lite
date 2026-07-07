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

const currentSchemaVersion = 2

var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("conflict")
	ErrForbidden = errors.New("forbidden")
)

type Data struct {
	SchemaVersion  int                `json:"schema_version"`
	Users          []model.User       `json:"users"`
	Customers      []model.Customer   `json:"customers"`
	Quotes         []model.Quote      `json:"quotes"`
	Invoices       []model.Invoice    `json:"invoices"`
	AuditLog       []model.AuditEvent `json:"audit_log"`
	NextUserID     int                `json:"next_user_id"`
	NextCustomerID int                `json:"next_customer_id"`
	NextQuoteID    int                `json:"next_quote_id"`
	NextInvoiceID  int                `json:"next_invoice_id"`
	NextAuditID    int                `json:"next_audit_id"`
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
			s.data = newData()
			return s, s.saveLocked()
		}
		return nil, err
	}

	if len(strings.TrimSpace(string(raw))) == 0 {
		s.data = newData()
		return s, s.saveLocked()
	}

	if err := json.Unmarshal(raw, &s.data); err != nil {
		return nil, err
	}

	s.migrateLocked()
	s.normalizeIDsLocked()
	return s, s.saveLocked()
}

func newData() Data {
	return Data{
		SchemaVersion:  currentSchemaVersion,
		NextUserID:     1,
		NextCustomerID: 1,
		NextQuoteID:    1,
		NextInvoiceID:  1,
		NextAuditID:    1,
	}
}

func (s *Store) migrateLocked() {
	if s.data.SchemaVersion == 0 {
		s.data.SchemaVersion = 1
	}

	if s.data.SchemaVersion < 2 {
		for i := range s.data.Users {
			// Legacy users had no Active field. Keep existing accounts usable.
			if !s.data.Users[i].Active {
				s.data.Users[i].Active = true
			}
			if s.data.Users[i].Role == "" {
				s.data.Users[i].Role = model.RoleUser
			}
		}
		for i := range s.data.Quotes {
			q := &s.data.Quotes[i]
			if len(q.Items) == 0 && q.NetAmount > 0 {
				q.Items = []model.LineItem{model.DefaultLineItem(q.Title, q.NetAmount, q.TaxRate)}
			}
			q.Items = model.NormalizeLineItems(q.Items)
			q.NetAmount, q.TaxAmount, q.GrossAmount = model.Totals(q.Items)
			if q.TaxRate == 0 {
				q.TaxRate = 19
			}
		}
		for i := range s.data.Invoices {
			inv := &s.data.Invoices[i]
			if len(inv.Items) == 0 && inv.NetAmount > 0 {
				inv.Items = []model.LineItem{model.DefaultLineItem(inv.Title, inv.NetAmount, inv.TaxRate)}
			}
			inv.Items = model.NormalizeLineItems(inv.Items)
			inv.NetAmount, inv.TaxAmount, inv.GrossAmount = model.Totals(inv.Items)
			if inv.TaxRate == 0 {
				inv.TaxRate = 19
			}
		}
		s.data.SchemaVersion = 2
	}

	if s.data.SchemaVersion < currentSchemaVersion {
		s.data.SchemaVersion = currentSchemaVersion
	}
}

func (s *Store) normalizeIDsLocked() {
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
	if s.data.NextAuditID < 1 {
		s.data.NextAuditID = 1
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
	for _, item := range s.data.AuditLog {
		if item.ID >= s.data.NextAuditID {
			s.data.NextAuditID = item.ID + 1
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

func (s *Store) AddAudit(actor, action, entity string, entityID int, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addAuditLocked(actor, action, entity, entityID, message)
	return s.saveLocked()
}

func (s *Store) addAuditLocked(actor, action, entity string, entityID int, message string) {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	event := model.AuditEvent{
		ID:        s.data.NextAuditID,
		Actor:     actor,
		Action:    strings.TrimSpace(action),
		Entity:    strings.TrimSpace(entity),
		EntityID:  entityID,
		Message:   strings.TrimSpace(message),
		CreatedAt: time.Now(),
	}
	s.data.NextAuditID++
	s.data.AuditLog = append(s.data.AuditLog, event)
	if len(s.data.AuditLog) > 500 {
		s.data.AuditLog = s.data.AuditLog[len(s.data.AuditLog)-500:]
	}
}

func (s *Store) ListAudit() []model.AuditEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]model.AuditEvent(nil), s.data.AuditLog...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out
}

func (s *Store) UserCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data.Users)
}

func (s *Store) CreateUser(username, passwordHash, role string) (model.User, error) {
	return s.CreateUserWithActor("system", username, passwordHash, role, true)
}

func (s *Store) CreateUserWithActor(actor, username, passwordHash, role string, active bool) (model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	username = strings.TrimSpace(username)
	if username == "" {
		return model.User{}, errors.New("username is required")
	}
	if role == "" {
		role = model.RoleUser
	}
	if role != model.RoleAdmin && role != model.RoleUser {
		return model.User{}, errors.New("invalid role")
	}
	for _, user := range s.data.Users {
		if strings.EqualFold(user.Username, username) {
			return model.User{}, ErrConflict
		}
	}
	now := time.Now()
	user := model.User{ID: s.data.NextUserID, Username: username, PasswordHash: passwordHash, Role: role, Active: active, CreatedAt: now, UpdatedAt: now}
	s.data.NextUserID++
	s.data.Users = append(s.data.Users, user)
	s.addAuditLocked(actor, "create", "user", user.ID, "Benutzer angelegt: "+user.Username)
	return user, s.saveLocked()
}

func (s *Store) UpdateUser(actor string, user model.User, newPasswordHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user.Username = strings.TrimSpace(user.Username)
	if user.Username == "" {
		return errors.New("username is required")
	}
	if user.Role != model.RoleAdmin && user.Role != model.RoleUser {
		return errors.New("invalid role")
	}
	for _, existing := range s.data.Users {
		if existing.ID != user.ID && strings.EqualFold(existing.Username, user.Username) {
			return ErrConflict
		}
	}
	for i := range s.data.Users {
		if s.data.Users[i].ID == user.ID {
			user.CreatedAt = s.data.Users[i].CreatedAt
			user.LastLoginAt = s.data.Users[i].LastLoginAt
			if newPasswordHash != "" {
				user.PasswordHash = newPasswordHash
			} else {
				user.PasswordHash = s.data.Users[i].PasswordHash
			}
			user.UpdatedAt = time.Now()
			s.data.Users[i] = user
			s.addAuditLocked(actor, "update", "user", user.ID, "Benutzer aktualisiert: "+user.Username)
			return s.saveLocked()
		}
	}
	return ErrNotFound
}

func (s *Store) DeleteUser(actor string, id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.data.Users) <= 1 {
		return fmt.Errorf("%w: last user cannot be deleted", ErrForbidden)
	}
	for _, inv := range s.data.Users {
		_ = inv
	}
	for i := range s.data.Users {
		if s.data.Users[i].ID == id {
			username := s.data.Users[i].Username
			s.data.Users = append(s.data.Users[:i], s.data.Users[i+1:]...)
			s.addAuditLocked(actor, "delete", "user", id, "Benutzer gelöscht: "+username)
			return s.saveLocked()
		}
	}
	return ErrNotFound
}

func (s *Store) ListUsers() []model.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]model.User(nil), s.data.Users...)
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Username) < strings.ToLower(out[j].Username) })
	return out
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

func (s *Store) MarkUserLogin(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for i := range s.data.Users {
		if s.data.Users[i].ID == id {
			s.data.Users[i].LastLoginAt = &now
			s.data.Users[i].UpdatedAt = now
			return s.saveLocked()
		}
	}
	return ErrNotFound
}

func (s *Store) SetPassword(actor string, userID int, passwordHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Users {
		if s.data.Users[i].ID == userID {
			s.data.Users[i].PasswordHash = passwordHash
			s.data.Users[i].UpdatedAt = time.Now()
			s.addAuditLocked(actor, "password", "user", userID, "Passwort geändert/zurückgesetzt")
			return s.saveLocked()
		}
	}
	return ErrNotFound
}

func (s *Store) ListCustomers() []model.Customer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]model.Customer(nil), s.data.Customers...)
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Company) < strings.ToLower(out[j].Company) })
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
	return s.CreateCustomerWithActor("system", c)
}
func (s *Store) CreateCustomerWithActor(actor string, c model.Customer) (model.Customer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c.Company = strings.TrimSpace(c.Company)
	if c.Company == "" {
		return model.Customer{}, errors.New("company is required")
	}
	now := time.Now()
	c.ID = s.data.NextCustomerID
	c.CreatedAt = now
	c.UpdatedAt = now
	s.data.NextCustomerID++
	s.data.Customers = append(s.data.Customers, c)
	s.addAuditLocked(actor, "create", "customer", c.ID, "Kunde angelegt: "+c.Company)
	return c, s.saveLocked()
}

func (s *Store) UpdateCustomer(c model.Customer) error { return s.UpdateCustomerWithActor("system", c) }
func (s *Store) UpdateCustomerWithActor(actor string, c model.Customer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c.Company = strings.TrimSpace(c.Company)
	if c.Company == "" {
		return errors.New("company is required")
	}
	for i := range s.data.Customers {
		if s.data.Customers[i].ID == c.ID {
			c.CreatedAt = s.data.Customers[i].CreatedAt
			c.UpdatedAt = time.Now()
			s.data.Customers[i] = c
			s.addAuditLocked(actor, "update", "customer", c.ID, "Kunde aktualisiert: "+c.Company)
			return s.saveLocked()
		}
	}
	return ErrNotFound
}

func (s *Store) DeleteCustomer(id int) error { return s.DeleteCustomerWithActor("system", id) }
func (s *Store) DeleteCustomerWithActor(actor string, id int) error {
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
			name := s.data.Customers[i].Company
			s.data.Customers = append(s.data.Customers[:i], s.data.Customers[i+1:]...)
			s.addAuditLocked(actor, "delete", "customer", id, "Kunde gelöscht: "+name)
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
	return s.CreateQuoteWithActor("system", q)
}
func (s *Store) CreateQuoteWithActor(actor string, q model.Quote) (model.Quote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.customerExistsLocked(q.CustomerID) {
		return model.Quote{}, errors.New("customer does not exist")
	}
	q.Title = strings.TrimSpace(q.Title)
	if q.Title == "" {
		return model.Quote{}, errors.New("title is required")
	}
	if q.Status == "" {
		q.Status = "draft"
	}
	q.Items = model.NormalizeLineItems(q.Items)
	if len(q.Items) == 0 {
		return model.Quote{}, errors.New("at least one line item is required")
	}
	q.NetAmount, q.TaxAmount, q.GrossAmount = model.Totals(q.Items)
	q.TaxRate = dominantTaxRate(q.Items)
	now := time.Now()
	q.ID = s.data.NextQuoteID
	q.Number = fmt.Sprintf("Q-%d-%04d", now.Year(), q.ID)
	q.CreatedAt = now
	q.UpdatedAt = now
	s.data.NextQuoteID++
	s.data.Quotes = append(s.data.Quotes, q)
	s.addAuditLocked(actor, "create", "quote", q.ID, "Angebot angelegt: "+q.Number)
	return q, s.saveLocked()
}

func (s *Store) UpdateQuote(q model.Quote) error { return s.UpdateQuoteWithActor("system", q) }
func (s *Store) UpdateQuoteWithActor(actor string, q model.Quote) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.customerExistsLocked(q.CustomerID) {
		return errors.New("customer does not exist")
	}
	q.Title = strings.TrimSpace(q.Title)
	if q.Title == "" {
		return errors.New("title is required")
	}
	q.Items = model.NormalizeLineItems(q.Items)
	if len(q.Items) == 0 {
		return errors.New("at least one line item is required")
	}
	q.NetAmount, q.TaxAmount, q.GrossAmount = model.Totals(q.Items)
	q.TaxRate = dominantTaxRate(q.Items)
	for i := range s.data.Quotes {
		if s.data.Quotes[i].ID == q.ID {
			q.Number = s.data.Quotes[i].Number
			q.CreatedAt = s.data.Quotes[i].CreatedAt
			q.UpdatedAt = time.Now()
			s.data.Quotes[i] = q
			s.addAuditLocked(actor, "update", "quote", q.ID, "Angebot aktualisiert: "+q.Number)
			return s.saveLocked()
		}
	}
	return ErrNotFound
}

func (s *Store) DeleteQuote(id int) error { return s.DeleteQuoteWithActor("system", id) }
func (s *Store) DeleteQuoteWithActor(actor string, id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, inv := range s.data.Invoices {
		if inv.QuoteID == id {
			return fmt.Errorf("%w: quote already converted to invoice", ErrForbidden)
		}
	}
	for i := range s.data.Quotes {
		if s.data.Quotes[i].ID == id {
			number := s.data.Quotes[i].Number
			s.data.Quotes = append(s.data.Quotes[:i], s.data.Quotes[i+1:]...)
			s.addAuditLocked(actor, "delete", "quote", id, "Angebot gelöscht: "+number)
			return s.saveLocked()
		}
	}
	return ErrNotFound
}

func (s *Store) ConvertQuoteToInvoice(quoteID int) (model.Invoice, error) {
	return s.ConvertQuoteToInvoiceWithActor("system", quoteID)
}
func (s *Store) ConvertQuoteToInvoiceWithActor(actor string, quoteID int) (model.Invoice, error) {
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
		Items:       append([]model.LineItem(nil), quote.Items...),
		NetAmount:   quote.NetAmount,
		TaxRate:     quote.TaxRate,
		TaxAmount:   quote.TaxAmount,
		GrossAmount: quote.GrossAmount,
		Status:      "open",
		InvoiceDate: now.Format("2006-01-02"),
		DueDate:     now.AddDate(0, 0, 14).Format("2006-01-02"),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.data.NextInvoiceID++
	s.data.Invoices = append(s.data.Invoices, invoice)
	s.addAuditLocked(actor, "convert", "invoice", invoice.ID, "Angebot "+quote.Number+" in Rechnung "+invoice.Number+" umgewandelt")
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
	return s.CreateInvoiceWithActor("system", inv)
}
func (s *Store) CreateInvoiceWithActor(actor string, inv model.Invoice) (model.Invoice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.customerExistsLocked(inv.CustomerID) {
		return model.Invoice{}, errors.New("customer does not exist")
	}
	inv.Title = strings.TrimSpace(inv.Title)
	if inv.Title == "" {
		return model.Invoice{}, errors.New("title is required")
	}
	if inv.Status == "" {
		inv.Status = "open"
	}
	inv.Items = model.NormalizeLineItems(inv.Items)
	if len(inv.Items) == 0 {
		return model.Invoice{}, errors.New("at least one line item is required")
	}
	inv.NetAmount, inv.TaxAmount, inv.GrossAmount = model.Totals(inv.Items)
	inv.TaxRate = dominantTaxRate(inv.Items)
	now := time.Now()
	inv.ID = s.data.NextInvoiceID
	inv.Number = fmt.Sprintf("INV-%d-%04d", now.Year(), inv.ID)
	inv.CreatedAt = now
	inv.UpdatedAt = now
	s.data.NextInvoiceID++
	s.data.Invoices = append(s.data.Invoices, inv)
	s.addAuditLocked(actor, "create", "invoice", inv.ID, "Rechnung angelegt: "+inv.Number)
	return inv, s.saveLocked()
}

func (s *Store) UpdateInvoice(inv model.Invoice) error {
	return s.UpdateInvoiceWithActor("system", inv)
}
func (s *Store) UpdateInvoiceWithActor(actor string, inv model.Invoice) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.customerExistsLocked(inv.CustomerID) {
		return errors.New("customer does not exist")
	}
	inv.Title = strings.TrimSpace(inv.Title)
	if inv.Title == "" {
		return errors.New("title is required")
	}
	if inv.PaymentDate != "" {
		inv.Status = "paid"
	}
	inv.Items = model.NormalizeLineItems(inv.Items)
	if len(inv.Items) == 0 {
		return errors.New("at least one line item is required")
	}
	inv.NetAmount, inv.TaxAmount, inv.GrossAmount = model.Totals(inv.Items)
	inv.TaxRate = dominantTaxRate(inv.Items)
	for i := range s.data.Invoices {
		if s.data.Invoices[i].ID == inv.ID {
			inv.Number = s.data.Invoices[i].Number
			inv.QuoteID = s.data.Invoices[i].QuoteID
			inv.CreatedAt = s.data.Invoices[i].CreatedAt
			inv.UpdatedAt = time.Now()
			s.data.Invoices[i] = inv
			s.addAuditLocked(actor, "update", "invoice", inv.ID, "Rechnung aktualisiert: "+inv.Number)
			return s.saveLocked()
		}
	}
	return ErrNotFound
}

func (s *Store) DeleteInvoice(id int) error { return s.DeleteInvoiceWithActor("system", id) }
func (s *Store) DeleteInvoiceWithActor(actor string, id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Invoices {
		if s.data.Invoices[i].ID == id {
			number := s.data.Invoices[i].Number
			s.data.Invoices = append(s.data.Invoices[:i], s.data.Invoices[i+1:]...)
			s.addAuditLocked(actor, "delete", "invoice", id, "Rechnung gelöscht: "+number)
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
	stats := DashboardStats{Customers: len(s.data.Customers), Quotes: len(s.data.Quotes), Invoices: len(s.data.Invoices)}
	for _, inv := range s.data.Invoices {
		switch inv.Status {
		case "paid":
			stats.PaidInvoices++
			stats.RevenueGross += inv.GrossAmount
		case "open", "overdue", "reminded":
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

func dominantTaxRate(items []model.LineItem) int {
	if len(items) == 0 {
		return 19
	}
	return items[0].TaxRate
}
