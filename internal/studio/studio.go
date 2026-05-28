package studio

import (
	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/studio/mongodb"
	"github.com/Lumos-Labs-HQ/flash/internal/studio/sql"
)

type Server interface {
	Start(openBrowser bool) error
}

func New(cfg *config.Config, port int, host, authToken string) Server {
	switch cfg.Database.Provider {
	case "mongodb", "mongo":
		return mongodb.NewServer(cfg, port, host, authToken)
	default:
		return sql.NewServer(cfg, port, host, authToken)
	}
}
