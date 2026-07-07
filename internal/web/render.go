package web

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

type Renderer struct {
	TemplateDir string
}

func NewRenderer(templateDir string) *Renderer {
	return &Renderer{TemplateDir: templateDir}
}

func (r *Renderer) Render(w http.ResponseWriter, status int, page string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}

	funcs := template.FuncMap{
		"money": formatMoney,
		"selected": func(current, expected string) string {
			if current == expected {
				return "selected"
			}
			return ""
		},
		"checkedID": func(current, expected int) string {
			if current == expected {
				return "selected"
			}
			return ""
		},
	}

	files := []string{
		filepath.Join(r.TemplateDir, "layout.html"),
		filepath.Join(r.TemplateDir, page),
	}

	tpl, err := template.New("layout.html").Funcs(funcs).ParseFiles(files...)
	if err != nil {
		http.Error(w, fmt.Sprintf("template parse error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	if err := tpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, fmt.Sprintf("template execute error: %v", err), http.StatusInternalServerError)
	}
}

func formatMoney(cents int64) string {
	negative := cents < 0
	if negative {
		cents = -cents
	}

	euros := cents / 100
	rest := cents % 100
	out := fmt.Sprintf("%d,%02d €", euros, rest)
	if negative {
		return "-" + out
	}
	return out
}

func ParseMoney(input string) (int64, error) {
	value := strings.TrimSpace(input)
	value = strings.ReplaceAll(value, "€", "")
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, ".", "")
	value = strings.ReplaceAll(value, ",", ".")

	if value == "" {
		return 0, nil
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid amount")
	}

	euros, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}

	var cents int64
	if len(parts) == 2 {
		decimal := parts[1]
		if len(decimal) == 1 {
			decimal += "0"
		}
		if len(decimal) > 2 {
			decimal = decimal[:2]
		}
		cents, err = strconv.ParseInt(decimal, 10, 64)
		if err != nil {
			return 0, err
		}
	}

	return euros*100 + cents, nil
}
