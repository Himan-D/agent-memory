package main

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var landingFS http.FileSystem

func serveLanding() http.Handler {
	fs := http.FileServer(landingFS)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleanPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")

		if cleanPath == "" {
			cleanPath = "index.html"
		}

		if f, err := landingFS.Open(cleanPath); err == nil {
			f.Close()
			fs.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/" || !strings.HasPrefix(r.URL.Path, "/assets/") {
			r.URL.Path = "/"
			fs.ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
	})
}

func init() {
	candidates := []string{
		"landing/dist",
		"../landing/dist",
		filepath.Join(filepath.Dir(os.Args[0]), "landing/dist"),
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			landingFS = http.Dir(c)
			break
		}
	}
}
