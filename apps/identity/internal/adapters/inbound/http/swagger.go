package httpadapter

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"os"
	"strings"
)

const swaggerDocsHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Identity Service API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script nonce="__SWAGGER_NONCE__">
    window.onload = () => window.ui = SwaggerUIBundle({
      url: "/v1/openapi.yaml",
      dom_id: "#swagger-ui",
      deepLinking: true,
      displayRequestDuration: true,
      persistAuthorization: false,
      filter: true,
      tryItOutEnabled: true
    });
  </script>
</body>
</html>`

func (s *Server) swaggerDocs(writer http.ResponseWriter, _ *http.Request) {
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		problem(writer, http.StatusInternalServerError, "internal_error", "Unable to initialize API documentation")
		return
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Content-Security-Policy", "default-src 'none'; style-src https://unpkg.com 'unsafe-inline'; script-src https://unpkg.com 'nonce-"+nonce+"'; img-src https://unpkg.com data:; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("Referrer-Policy", "no-referrer")
	writer.Header().Set("Cache-Control", "no-store")
	_, _ = writer.Write([]byte(strings.Replace(swaggerDocsHTML, "__SWAGGER_NONCE__", nonce, 1)))
}

func (s *Server) openAPI(writer http.ResponseWriter, _ *http.Request) {
	data, err := os.ReadFile(s.openAPIPath)
	if err != nil {
		problem(writer, http.StatusServiceUnavailable, "service_unavailable", "OpenAPI contract is unavailable")
		return
	}
	writer.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("Cache-Control", "public, max-age=300")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(data)
}
