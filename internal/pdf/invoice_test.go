package pdf

import (
	"bytes"
	"strings"
	"testing"

	"go-invoice-lite/internal/model"
)

func TestWriteInvoicePDF(t *testing.T) {
	inv := model.Invoice{
		Number:      "INV-2026-0001",
		Title:       "Testrechnung",
		Status:      "open",
		InvoiceDate: "2026-07-07",
		DueDate:     "2026-07-21",
		Items:       []model.LineItem{model.DefaultLineItem("Entwicklung", 10000, 19)},
		NetAmount:   10000,
		TaxAmount:   1900,
		GrossAmount: 11900,
	}
	var buf bytes.Buffer
	if err := WriteInvoice(&buf, inv, "ACME GmbH"); err != nil {
		t.Fatalf("WriteInvoice failed: %v", err)
	}
	if !strings.HasPrefix(buf.String(), "%PDF-1.4") {
		t.Fatal("expected PDF header")
	}
}
