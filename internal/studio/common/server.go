package common

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/template/html/v2"
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

// NewApp creates a new Fiber app with the given template FS
func NewApp(templatesFS fs.FS) *fiber.App {
	engine := html.NewFileSystem(http.FS(templatesFS), ".html")
	return fiber.New(fiber.Config{Views: engine})
}

// SetupStaticFS mounts studio-specific and common static files on the app
func SetupStaticFS(app *fiber.App, studioStaticFS embed.FS) {
	staticFS, _ := fs.Sub(studioStaticFS, "static")
	app.Use("/static", filesystem.New(filesystem.Config{
		Root: http.FS(staticFS),
	}))

	// Serve common shared files
	app.Get("/common/static/css/base.css", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/css")
		return c.Send(baseCSS)
	})
	app.Get("/common/static/js/common.js", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "application/javascript")
		return c.Send(commonJS)
	})
}

// StartServer finds an available port, prints the URL, optionally opens a browser, and starts listening
func StartServer(app *fiber.App, port *int, name string, openBrowser bool) error {
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

	return app.Listen(fmt.Sprintf(":%d", *port))
}
