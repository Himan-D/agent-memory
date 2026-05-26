package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
)

//go:embed swagger.json
var swaggerFS embed.FS

func serveSwagger(w http.ResponseWriter, r *http.Request) {
	data, err := swaggerFS.ReadFile("swagger.json")
	if err != nil {
		jsonError(w, "Swagger spec not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func serveSwaggerUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("swagger").Parse(swaggerUITemplate))
	tmpl.Execute(w, struct{}{})
}

func registerSwaggerRoutes(router *mux.Router) {
	router.HandleFunc("/swagger.json", serveSwagger).Methods("GET")
	router.HandleFunc("/docs", serveSwaggerUI).Methods("GET")
	router.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		data, err := swaggerFS.ReadFile("swagger.json")
		if err != nil {
			jsonError(w, "not found", http.StatusNotFound)
			return
		}
		var spec map[string]interface{}
		if err := json.Unmarshal(data, &spec); err != nil {
			jsonError(w, "invalid spec", http.StatusInternalServerError)
			return
		}
		servers, ok := spec["servers"].([]interface{})
		if ok {
			for i, s := range servers {
				if svr, ok := s.(map[string]interface{}); ok {
					if url, ok := svr["url"].(string); ok && (url == "http://localhost:8080" || url == "") {
						svr["url"] = fmt.Sprintf("http://%s", r.Host)
						servers[i] = svr
					}
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(spec)
	}).Methods("GET")
}

const swaggerUITemplate = `<!DOCTYPE html>
<html>
<head>
    <title>Hystersis API Documentation</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin: 0; background: #fafafa; }
        .topbar { display: none; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
        const ui = SwaggerUIBundle({
            url: "/openapi.json",
            dom_id: '#swagger-ui',
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIBundle.SwaggerUIStandalonePreset
            ],
            layout: "BaseLayout",
            tryItOutEnabled: true,
            requestInterceptor: function(req) {
                const apiKey = new URLSearchParams(window.location.search).get('api_key');
                if (apiKey) {
                    req.headers['X-API-Key'] = apiKey;
                }
                return req;
            }
        });
    </script>
</body>
</html>`
