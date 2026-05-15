package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

// SchemaSnapshot is a JSON-serialized representation of the database schema
// at a specific point in time. It lives next to the migrations folder and is
// updated every time a migration is generated. This lets the generator diff
// against the *last known schema* instead of the live DB, which solves the
// "unapplied migration" edge case cleanly.
type SchemaSnapshot struct {
	Version     string             `json:"version"`
	GeneratedAt time.Time          `json:"generated_at"`
	Tables      []types.SchemaTable `json:"tables"`
	Enums       []types.SchemaEnum  `json:"enums"`
}

const snapshotVersion = "1"
const defaultSnapshotFileName = "schema_snapshot.json"

// SnapshotPath returns the default snapshot path inside a migrations directory.
func SnapshotPath(migrationsDir string) string {
	return filepath.Join(migrationsDir, ".flash", defaultSnapshotFileName)
}

// LoadSchemaSnapshot reads a snapshot from disk. If the file does not exist
// or is unreadable, it returns (nil, nil) so the caller can fall back to the
// live database.
func LoadSchemaSnapshot(path string) (*SchemaSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read schema snapshot: %w", err)
	}

	var snap SchemaSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("failed to parse schema snapshot: %w", err)
	}

	// Basic validation
	if snap.Version == "" {
		return nil, fmt.Errorf("schema snapshot missing version")
	}

	return &snap, nil
}

// SaveSchemaSnapshot writes the given schema state to disk.
func SaveSchemaSnapshot(path string, tables []types.SchemaTable, enums []types.SchemaEnum) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	snap := SchemaSnapshot{
		Version:     snapshotVersion,
		GeneratedAt: time.Now().UTC(),
		Tables:      tables,
		Enums:       enums,
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schema snapshot: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write schema snapshot: %w", err)
	}

	return nil
}
