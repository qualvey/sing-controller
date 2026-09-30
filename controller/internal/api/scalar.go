package api

import (
	"net/http"
	"os"
)

const scalarHTML = `<!doctype html>
<html>
  <head>
    <title>sing-controller API Reference</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <link rel="icon" type="image/svg+xml" href="/logo.svg" />
    <style>
      body {
        margin: 0;
        padding: 0;
      }
    </style>
  </head>
  <body>
    <script
      id="api-reference"
      data-url="/api/swagger.json"
      data-configuration='{
        "theme": "purple",
        "layout": "modern",
        "showSidebar": true,
        "defaultHttpClient": {
          "targetKey": "shell",
          "clientKey": "curl"
        }
      }'>
    </script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`

func (h *Handler) handleSwaggerJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// 优先读取本地最新的 swagger.json（若处于开发形态）
	if data, err := os.ReadFile("swagger.json"); err == nil {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	if data, err := os.ReadFile("../docs/public/api/swagger.json"); err == nil {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	// 内存编译内嵌的 SwaggerInfo
	if doc := SwaggerInfo.ReadDoc(); doc != "" {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(doc))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"openapi":"3.0.0","info":{"title":"sing-controller API","version":"1.0"}}`))
}

func (h *Handler) handleAPIDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(scalarHTML))
}
