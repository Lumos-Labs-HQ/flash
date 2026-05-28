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

	// Serve common static assets (images, etc.)
	commonFS, _ := fs.Sub(CommonStaticFS, "static")
	commonFileServer := http.FileServer(http.FS(commonFS))
	mux.Handle("GET /common/static/", http.StripPrefix("/common/static/", commonFileServer))

	// Serve CDN assets locally for offline support
	cdnFS, _ := fs.Sub(CdnFS, "cdn")
	cdnServer := http.FileServer(http.FS(cdnFS))
	mux.Handle("GET /cdn/", http.StripPrefix("/cdn/", cdnServer))
}

// StartServerConfig holds configuration for starting the studio server.
type StartServerConfig struct {
	Host        string
	Port        int
	Name        string
	OpenBrowser bool
	AuthToken   string
}

// StartServer finds an available port, prints the URL, optionally opens a browser, and starts listening.
func StartServer(mux *http.ServeMux, cfg StartServerConfig) error {
	available := FindAvailablePort(cfg.Port)
	if available != cfg.Port {
		fmt.Printf("Port %d is in use, using port %d instead\n", cfg.Port, available)
		cfg.Port = available
	}

	// Wrap mux with middleware
	var handler http.Handler = mux
	handler = MaxBytesMiddleware(10 << 20)(handler) // 10 MB
	handler = CORSMiddleware(handler)
	handler = AuthMiddleware(cfg.AuthToken)(handler)

	bindAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	url := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)
	fmt.Printf("FlashORM %s starting on %s\n", cfg.Name, url)

	if cfg.OpenBrowser {
		go OpenBrowser(url)
	}

	return http.ListenAndServe(bindAddr, handler)
}
