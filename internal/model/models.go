package model

import "time"

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
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
}

type Quote struct {
	ID          int       `json:"id"`
	Number      string    `json:"number"`
	CustomerID  int       `json:"customer_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	NetAmount   int64     `json:"net_amount"`
	TaxRate     int       `json:"tax_rate"`
	GrossAmount int64     `json:"gross_amount"`
	Status      string    `json:"status"`
	ValidUntil  string    `json:"valid_until"`
	CreatedAt   time.Time `json:"created_at"`
}

type Invoice struct {
	ID          int       `json:"id"`
	Number      string    `json:"number"`
	CustomerID  int       `json:"customer_id"`
	QuoteID     int       `json:"quote_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	NetAmount   int64     `json:"net_amount"`
	TaxRate     int       `json:"tax_rate"`
	GrossAmount int64     `json:"gross_amount"`
	Status      string    `json:"status"`
	InvoiceDate string    `json:"invoice_date"`
	DueDate     string    `json:"due_date"`
	CreatedAt   time.Time `json:"created_at"`
}

func CalculateGross(netAmount int64, taxRate int) int64 {
	return netAmount + (netAmount*int64(taxRate))/100
}
