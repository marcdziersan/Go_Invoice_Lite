package model

import "testing"

func TestNormalizeLineItemsAndTotals(t *testing.T) {
	items := NormalizeLineItems([]LineItem{
		{Description: "Entwicklung", Quantity: 2, UnitNetAmount: 10000, TaxRate: 19},
		{Description: "Hosting", Quantity: 1, UnitNetAmount: 5000, TaxRate: 7},
	})
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	net, tax, gross := Totals(items)
	if net != 25000 {
		t.Fatalf("net = %d, want 25000", net)
	}
	if tax != 4150 {
		t.Fatalf("tax = %d, want 4150", tax)
	}
	if gross != 29150 {
		t.Fatalf("gross = %d, want 29150", gross)
	}
}
