// Package handlers provides HTTP request handling for the mock API.
package handlers

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed *.json *.html
var swaggerFiles embed.FS

// SwaggerHandler serves Swagger UI and OpenAPI spec
type SwaggerHandler struct{}

// NewSwaggerHandler creates a new Swagger handler instance
func NewSwaggerHandler() *SwaggerHandler {
	return &SwaggerHandler{}
}

// ServeHTTP handles requests to /swagger/* paths
func (h *SwaggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Handle /swagger/openapi.json (with optional query params for cache busting)
	if path == "/swagger/openapi.json" || path == "/swagger/openapi.json/" || strings.HasPrefix(path, "/swagger/openapi.json?") {
		w.Header().Set("Content-Type", "application/json")
		data, err := fs.ReadFile(swaggerFiles, "openapi.json")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load OpenAPI spec: %v", err), http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}

	// Handle /swagger/ui.html or /swagger/
	if path == "/swagger/" || path == "/swagger/ui.html" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data, err := fs.ReadFile(swaggerFiles, "ui.html")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load Swagger UI: %v", err), http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}

	http.NotFound(w, r)
}
