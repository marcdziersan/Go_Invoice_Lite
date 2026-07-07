package store

import (
	"path/filepath"
	"testing"

	"go-invoice-lite/internal/model"
)

func TestStoreBusinessFlow(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "app.json"))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	customer, err := st.CreateCustomerWithActor("tester", model.Customer{Company: "ACME GmbH"})
	if err != nil {
		t.Fatalf("CreateCustomer failed: %v", err)
	}
	quote, err := st.CreateQuoteWithActor("tester", model.Quote{
		CustomerID: customer.ID,
		Title:      "Website",
		Items:      []model.LineItem{{Description: "Entwicklung", Quantity: 2, UnitNetAmount: 10000, TaxRate: 19}},
	})
	if err != nil {
		t.Fatalf("CreateQuote failed: %v", err)
	}
	if quote.GrossAmount != 23800 {
		t.Fatalf("quote gross = %d, want 23800", quote.GrossAmount)
	}
	invoice, err := st.ConvertQuoteToInvoiceWithActor("tester", quote.ID)
	if err != nil {
		t.Fatalf("ConvertQuoteToInvoice failed: %v", err)
	}
	invoice.PaymentDate = "2026-07-07"
	if err := st.UpdateInvoiceWithActor("tester", invoice); err != nil {
		t.Fatalf("UpdateInvoice failed: %v", err)
	}
	updated, ok := st.GetInvoice(invoice.ID)
	if !ok {
		t.Fatal("invoice not found")
	}
	if updated.Status != "paid" {
		t.Fatalf("status = %q, want paid", updated.Status)
	}
	if len(st.ListAudit()) == 0 {
		t.Fatal("expected audit entries")
	}
}
