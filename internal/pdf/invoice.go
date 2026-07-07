package pdf

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"go-invoice-lite/internal/model"
)

func WriteInvoice(w io.Writer, inv model.Invoice, customerName string) error {
	lines := []string{
		"GoInvoice Lite",
		"Rechnung " + inv.Number,
		"Kunde: " + customerName,
		"Titel: " + inv.Title,
		"Rechnungsdatum: " + inv.InvoiceDate,
		"Faelligkeit: " + inv.DueDate,
		"Status: " + inv.Status,
		"",
		"Positionen:",
	}
	for _, item := range inv.Items {
		lines = append(lines, fmt.Sprintf("%d x %s | Netto %s | MwSt %d%% | Brutto %s", item.Quantity, item.Description, money(item.NetAmount), item.TaxRate, money(item.GrossAmount)))
	}
	lines = append(lines,
		"",
		"Netto: "+money(inv.NetAmount),
		"MwSt: "+money(inv.TaxAmount),
		"Brutto: "+money(inv.GrossAmount),
	)
	if inv.PaymentDate != "" {
		lines = append(lines, "Bezahlt am: "+inv.PaymentDate)
	}
	if inv.DunningLevel > 0 {
		lines = append(lines, "Mahnstufe: "+strconv.Itoa(inv.DunningLevel))
	}

	content := buildContent(lines)
	return writeSimplePDF(w, content)
}

func buildContent(lines []string) string {
	var b strings.Builder
	b.WriteString("BT\n/F1 12 Tf\n50 790 Td\n")
	for i, line := range lines {
		if i == 0 {
			b.WriteString("/F1 18 Tf\n")
		} else if i == 1 {
			b.WriteString("/F1 15 Tf\n")
		} else {
			b.WriteString("/F1 10 Tf\n")
		}
		b.WriteString("(")
		b.WriteString(escapePDFText(line))
		b.WriteString(") Tj\n0 -18 Td\n")
	}
	b.WriteString("ET\n")
	return b.String()
}

func writeSimplePDF(w io.Writer, stream string) error {
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len([]byte(stream)), stream),
	}
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(objects)+1, xref)
	_, err := w.Write(buf.Bytes())
	return err
}

func escapePDFText(s string) string {
	replacements := map[string]string{
		"\\": "\\\\",
		"(":  "\\(",
		")":  "\\)",
		"€":  "EUR",
		"ä":  "ae",
		"ö":  "oe",
		"ü":  "ue",
		"Ä":  "Ae",
		"Ö":  "Oe",
		"Ü":  "Ue",
		"ß":  "ss",
	}
	for old, newValue := range replacements {
		s = strings.ReplaceAll(s, old, newValue)
	}
	return s
}

func money(cents int64) string {
	negative := cents < 0
	if negative {
		cents = -cents
	}
	euros := cents / 100
	rest := cents % 100
	value := fmt.Sprintf("%d,%02d EUR", euros, rest)
	if negative {
		return "-" + value
	}
	return value
}
