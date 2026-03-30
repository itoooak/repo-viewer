package view

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
)

type Renderer struct {
	templates *template.Template
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		if bytes == 1 {
			return "1 byte"
		}
		return fmt.Sprintf("%d bytes", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

func NewRenderer(templateDir string) (*Renderer, error) {
	funcMap := template.FuncMap{
		"formatFileSize": formatFileSize,
	}

	templates, err := template.New("").Funcs(funcMap).ParseGlob(filepath.Join(templateDir, "*.html"))
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return &Renderer{templates: templates}, nil
}

func (r *Renderer) Render(w http.ResponseWriter, name string, data interface{}) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return r.templates.ExecuteTemplate(w, name, data)
}

func (r *Renderer) RenderError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	data := map[string]interface{}{
		"Title":   fmt.Sprintf("Error %d - repo-viewer", code),
		"Code":    code,
		"Message": message,
	}
	if err := r.Render(w, "error", data); err != nil {
		http.Error(w, message, code)
	}
}
