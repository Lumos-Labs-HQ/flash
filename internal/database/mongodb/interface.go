package mongodb

import (
	"context"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
	"go.mongodb.org/mongo-driver/bson"
)

// This interface extends beyond the generic DatabaseAdapter to provide
type MongoDBAdapter interface {
	// Connection management
	Connect(ctx context.Context, url string) error
	Close() error
	Ping(ctx context.Context) error

	// Database operations
	ListDatabases(ctx context.Context) ([]map[string]interface{}, error)
	SwitchDatabase(dbName string) error
	DropDatabase(ctx context.Context, dbName string) error
	CreateDatabase(ctx context.Context, dbName string) error
	GetDatabaseStats(ctx context.Context) (map[string]interface{}, error)

	// Collection operations
	ListCollections(ctx context.Context) ([]string, error)
	ListCollectionsInDB(ctx context.Context, database string) ([]string, error)
	GetCollectionStats(ctx context.Context, collection string) (map[string]interface{}, error)
	CreateCollection(ctx context.Context, name string, options interface{}) error
	DropCollection(ctx context.Context, name string) error

	// Document operations
	FindDocuments(ctx context.Context, collection string, filter bson.M, skip, limit int64) ([]map[string]interface{}, error)
	FindDocumentsInDB(ctx context.Context, database, collection string, filter bson.M, skip, limit int64) ([]map[string]interface{}, error)
	CountDocuments(ctx context.Context, collection string, filter bson.M) (int64, error)
	CountDocumentsInDB(ctx context.Context, database, collection string, filter bson.M) (int64, error)
	InsertDocument(ctx context.Context, collection string, document interface{}) (string, error)
	UpdateDocument(ctx context.Context, collection string, id string, update interface{}) error
	DeleteDocument(ctx context.Context, collection string, id string) error

	// Index operations
	ListIndexes(ctx context.Context, collection string) ([]map[string]interface{}, error)
	CreateIndex(ctx context.Context, collection string, keys map[string]interface{}, unique bool) error
	DropIndex(ctx context.Context, collection string, indexName string) error

	// Aggregation
	Aggregate(ctx context.Context, collection string, pipeline interface{}) ([]map[string]interface{}, error)

	// Query execution
	ExecuteMongoQuery(ctx context.Context, query string) ([]map[string]interface{}, error)

	// Schema inference (for compatibility with generic adapter)
	GetAllTableNames(ctx context.Context) ([]string, error)
	GetTableColumns(ctx context.Context, tableName string) ([]types.SchemaColumn, error)
	GetTableData(ctx context.Context, tableName string) ([]map[string]interface{}, error)
	GetTableRowCount(ctx context.Context, tableName string) (int, error)
	GetAllTableRowCounts(ctx context.Context, tableNames []string) (map[string]int, error)
}
