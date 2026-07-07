package model

import (
	"strings"
	"time"
)

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

type User struct {
	ID           int        `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"password_hash"`
	Role         string     `json:"role"`
	Active       bool       `json:"active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

func (u User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

type Customer struct {
	ID          int       `json:"id"`
	Company     string    `json:"company"`
	ContactName string    `json:"contact_name"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	Street      string    `json:"street"`
	ZIP         string    `json:"zip"`
	City        string    `json:"city"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type LineItem struct {
	ID            int    `json:"id"`
	Description   string `json:"description"`
	Quantity      int64  `json:"quantity"`
	UnitNetAmount int64  `json:"unit_net_amount"`
	TaxRate       int    `json:"tax_rate"`
	NetAmount     int64  `json:"net_amount"`
	TaxAmount     int64  `json:"tax_amount"`
	GrossAmount   int64  `json:"gross_amount"`
}

type Quote struct {
	ID          int        `json:"id"`
	Number      string     `json:"number"`
	CustomerID  int        `json:"customer_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Items       []LineItem `json:"items"`
	NetAmount   int64      `json:"net_amount"`
	TaxRate     int        `json:"tax_rate"`
	TaxAmount   int64      `json:"tax_amount"`
	GrossAmount int64      `json:"gross_amount"`
	Status      string     `json:"status"`
	ValidUntil  string     `json:"valid_until"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Invoice struct {
	ID           int        `json:"id"`
	Number       string     `json:"number"`
	CustomerID   int        `json:"customer_id"`
	QuoteID      int        `json:"quote_id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Items        []LineItem `json:"items"`
	NetAmount    int64      `json:"net_amount"`
	TaxRate      int        `json:"tax_rate"`
	TaxAmount    int64      `json:"tax_amount"`
	GrossAmount  int64      `json:"gross_amount"`
	Status       string     `json:"status"`
	InvoiceDate  string     `json:"invoice_date"`
	DueDate      string     `json:"due_date"`
	PaymentDate  string     `json:"payment_date"`
	DunningLevel int        `json:"dunning_level"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type AuditEvent struct {
	ID        int       `json:"id"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Entity    string    `json:"entity"`
	EntityID  int       `json:"entity_id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

func CalculateGross(netAmount int64, taxRate int) int64 {
	return netAmount + (netAmount*int64(taxRate))/100
}

func NormalizeLineItems(items []LineItem) []LineItem {
	out := make([]LineItem, 0, len(items))
	for _, item := range items {
		item.Description = strings.TrimSpace(item.Description)
		if item.Description == "" && item.UnitNetAmount == 0 {
			continue
		}
		if item.Quantity <= 0 {
			item.Quantity = 1
		}
		if item.TaxRate < 0 {
			item.TaxRate = 0
		}
		if item.TaxRate == 0 {
			item.TaxRate = 19
		}
		item.ID = len(out) + 1
		item.NetAmount = item.UnitNetAmount * item.Quantity
		item.TaxAmount = (item.NetAmount * int64(item.TaxRate)) / 100
		item.GrossAmount = item.NetAmount + item.TaxAmount
		out = append(out, item)
	}
	return out
}

func Totals(items []LineItem) (net int64, tax int64, gross int64) {
	for _, item := range items {
		net += item.NetAmount
		tax += item.TaxAmount
		gross += item.GrossAmount
	}
	return net, tax, gross
}

func DefaultLineItem(description string, netAmount int64, taxRate int) LineItem {
	if taxRate == 0 {
		taxRate = 19
	}
	items := NormalizeLineItems([]LineItem{{
		Description:   description,
		Quantity:      1,
		UnitNetAmount: netAmount,
		TaxRate:       taxRate,
	}})
	if len(items) == 0 {
		return LineItem{ID: 1, Quantity: 1, TaxRate: taxRate}
	}
	return items[0]
}
