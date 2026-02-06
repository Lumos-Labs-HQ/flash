package common

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
)

//go:embed static/*
var CommonStaticFS embed.FS

// Pre-load common static files at init time
var (
	baseCSS  []byte
	commonJS []byte
)

func init() {
	baseCSS, _ = CommonStaticFS.ReadFile("static/css/base.css")
	commonJS, _ = CommonStaticFS.ReadFile("static/js/common.js")
}

// ParseTemplates parses HTML templates from the given embedded FS
func ParseTemplates(templatesFS fs.FS) *template.Template {
	return template.Must(template.ParseFS(templatesFS, "templates/*.html"))
}

// SetupStaticFS mounts studio-specific and common static files on the mux
func SetupStaticFS(mux *http.ServeMux, studioStaticFS embed.FS) {
	staticFS, _ := fs.Sub(studioStaticFS, "static")
	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// Serve common shared files
	mux.HandleFunc("GET /common/static/css/base.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		w.Write(baseCSS)
	})
	mux.HandleFunc("GET /common/static/js/common.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write(commonJS)
	})
}

// StartServer finds an available port, prints the URL, optionally opens a browser, and starts listening
func StartServer(mux *http.ServeMux, port *int, name string, openBrowser bool) error {
	available := FindAvailablePort(*port)
	if available != *port {
		fmt.Printf("Port %d is in use, using port %d instead\n", *port, available)
		*port = available
	}

	url := fmt.Sprintf("http://localhost:%d", *port)
	fmt.Printf("FlashORM %s starting on %s\n", name, url)

	if openBrowser {
		go OpenBrowser(url)
	}

	return http.ListenAndServe(fmt.Sprintf(":%d", *port), mux)
}
