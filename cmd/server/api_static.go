package main

import (
	"embed"
	"net/http"
)

//go:embed api/llms.txt api/agents.md
var apiFS embed.FS

func serveLlmsTxt(w http.ResponseWriter, r *http.Request) {
	data, err := apiFS.ReadFile("api/llms.txt")
	if err != nil {
		jsonError(w, "llms.txt not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(data)
}

func serveAgentsMd(w http.ResponseWriter, r *http.Request) {
	data, err := apiFS.ReadFile("api/agents.md")
	if err != nil {
		jsonError(w, "agents.md not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(data)
}