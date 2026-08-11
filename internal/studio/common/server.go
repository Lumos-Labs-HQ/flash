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

// BaseServer holds the infrastructure shared by every studio (redis, mongodb,
// sql). Each concrete studio embeds it and keeps only its own typed `service`
// field. It provides the common Start and template-render behaviour so those
// don't have to be duplicated per studio.
type BaseServer struct {
	Mux           *http.ServeMux
	Tmpl          *template.Template
	Port          int
	Host          string
	AuthToken     string
	ConnectionURL string
	Name          string // e.g. "Redis Studio", "MongoDB Studio", "Studio"
}

// Start finds an available port and starts the studio HTTP server.
func (b *BaseServer) Start(openBrowser bool) error {
	return StartServer(b.Mux, StartServerConfig{
		Host:        b.Host,
		Port:        b.Port,
		Name:        b.Name,
		OpenBrowser: openBrowser,
		AuthToken:   b.AuthToken,
	})
}

// Render executes a named template with the given data. The write error is
// intentionally ignored — the response headers are already committed by the
// time templating fails, so there is nothing actionable to return.
func (b *BaseServer) Render(w http.ResponseWriter, name string, data Map) {
	_ = b.Tmpl.ExecuteTemplate(w, name, data)
}

// SetupStaticFS mounts studio-specific and common static files on the mux
func SetupStaticFS(mux *http.ServeMux, studioStaticFS embed.FS) {
	staticFS, _ := fs.Sub(studioStaticFS, "static")
	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// Serve common shared files
	mux.HandleFunc("GET /common/static/css/base.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, _ = w.Write(baseCSS)
	})
	mux.HandleFunc("GET /common/static/js/common.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = w.Write(commonJS)
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
		go func() { _ = OpenBrowser(url) }()
	}

	return http.ListenAndServe(bindAddr, handler)
}
