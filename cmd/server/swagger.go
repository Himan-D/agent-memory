package main

import (
	"embed"
	"net/http"

	"github.com/gorilla/mux"
)

//go:embed swagger.json
var swaggerFS embed.FS

func serveSwagger(w http.ResponseWriter, r *http.Request) {
	data, err := swaggerFS.ReadFile("swagger.json")
	if err != nil {
		http.Error(w, "Swagger spec not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func serveDocsRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/swagger.json", http.StatusMovedPermanently)
}

func RegisterSwaggerRoutes(router *mux.Router) {
	router.HandleFunc("/swagger.json", serveSwagger).Methods("GET")
	router.HandleFunc("/docs", serveDocsRedirect).Methods("GET")
}
