package mongodb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Lumos-Labs-HQ/flash/internal/database/common"
	"github.com/Lumos-Labs-HQ/flash/internal/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Adapter struct {
	client   *mongo.Client
	database *mongo.Database
	dbName   string
}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Connect(ctx context.Context, url string) error {
	clientOpts := options.Client().ApplyURI(url)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	a.client = client
	dbName := a.extractDBName(url, clientOpts)

	// List all databases to help debug
	dbs, _ := client.ListDatabaseNames(ctx, bson.M{})
	fmt.Printf("[DEBUG] Available databases: %v\n", dbs)
	fmt.Printf("[DEBUG] Using database: %s\n", dbName)

	a.database = client.Database(dbName)
	a.dbName = dbName

	return nil
}

func (a *Adapter) extractDBName(url string, opts *options.ClientOptions) string {
	// Parse URL to get database name from path FIRST
	if len(url) > 0 {
		// Look for database name after last /
		parts := strings.Split(url, "/")
		if len(parts) > 3 {
			dbPart := parts[len(parts)-1]
			// Remove query parameters if any
			if idx := strings.Index(dbPart, "?"); idx > 0 {
				dbPart = dbPart[:idx]
			}
			if dbPart != "" && dbPart != "admin" {
				return dbPart
			}
		}
	}

	// Fallback to auth source
	if opts != nil && opts.Auth != nil && opts.Auth.AuthSource != "" && opts.Auth.AuthSource != "admin" {
		return opts.Auth.AuthSource
	}

	return "test"
}

func (a *Adapter) Close() error {
	if a.client != nil {
		return a.client.Disconnect(context.Background())
	}
	return nil
}

func (a *Adapter) Ping(ctx context.Context) error {
	return a.client.Ping(ctx, nil)
}

func (a *Adapter) GetAllTableNames(ctx context.Context) ([]string, error) {
	if a.database == nil {
		return nil, fmt.Errorf("database not connected")
	}

	names, err := a.database.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	var filtered []string
	for _, name := range names {
		if name != "_flash_migrations" {
			filtered = append(filtered, name)
		}
	}

	fmt.Printf("[DEBUG] MongoDB: Found %d collections (filtered: %d)\n", len(names), len(filtered))
	return filtered, nil
}

func (a *Adapter) GetTableColumns(ctx context.Context, tableName string) ([]types.SchemaColumn, error) {
	coll := a.database.Collection(tableName)

	cursor, err := coll.Find(ctx, bson.M{}, options.Find().SetLimit(100))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	fieldTypes := make(map[string]string)
	fieldNullable := make(map[string]bool)
	fieldCount := make(map[string]int)
	totalDocs := 0

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		totalDocs++

		for key, value := range doc {
			fieldCount[key]++
			inferredType := inferBSONType(value)

			if existingType, exists := fieldTypes[key]; exists {
				if existingType != inferredType {
					fieldTypes[key] = "mixed"
				}
			} else {
				fieldTypes[key] = inferredType
			}
		}
	}

	for field, count := range fieldCount {
		fieldNullable[field] = count < totalDocs
	}

	var columns []types.SchemaColumn
	columns = append(columns, types.SchemaColumn{
		Name:      "_id",
		Type:      "ObjectId",
		Nullable:  false,
		IsPrimary: true,
	})

	for field, fieldType := range fieldTypes {
		if field == "_id" {
			continue
		}
		columns = append(columns, types.SchemaColumn{
			Name:     field,
			Type:     fieldType,
			Nullable: fieldNullable[field],
		})
	}

	return columns, nil
}

func (a *Adapter) GetTableData(ctx context.Context, tableName string) ([]map[string]interface{}, error) {
	coll := a.database.Collection(tableName)
	cursor, err := coll.Find(ctx, bson.M{}, options.Find().SetLimit(1000))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		converted := make(map[string]interface{})
		for k, v := range doc {
			converted[k] = convertBSONValue(v)
		}
		results = append(results, converted)
	}
	return results, nil
}

func (a *Adapter) GetTableRowCount(ctx context.Context, tableName string) (int, error) {
	coll := a.database.Collection(tableName)
	count, err := coll.CountDocuments(ctx, bson.M{})
	return int(count), err
}

func (a *Adapter) GetAllTableRowCounts(ctx context.Context, tableNames []string) (map[string]int, error) {
	result := make(map[string]int)
	for _, name := range tableNames {
		count, _ := a.GetTableRowCount(ctx, name)
		result[name] = count
	}
	return result, nil
}

func inferBSONType(value interface{}) string {
	switch value.(type) {
	case string:
		return "string"
	case int, int32, int64:
		return "int"
	case float32, float64:
		return "double"
	case bool:
		return "bool"
	case bson.M, map[string]interface{}:
		return "object"
	case bson.A, []interface{}:
		return "array"
	case time.Time:
		return "date"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", value)
	}
}

func convertBSONValue(v interface{}) interface{} {
	switch val := v.(type) {
	case bson.M:
		result := make(map[string]interface{})
		for k, v := range val {
			result[k] = convertBSONValue(v)
		}
		return result
	case bson.A:
		result := make([]interface{}, len(val))
		for i, v := range val {
			result[i] = convertBSONValue(v)
		}
		return result
	case bson.D:
		result := make(map[string]interface{})
		for _, elem := range val {
			result[elem.Key] = convertBSONValue(elem.Value)
		}
		return result
	default:
		return v
	}
}

// Stub implementations for DatabaseAdapter interface
func (a *Adapter) CreateMigrationsTable(ctx context.Context) error             { return nil }
func (a *Adapter) EnsureMigrationTableCompatibility(ctx context.Context) error { return nil }
func (a *Adapter) CleanupBrokenMigrationRecords(ctx context.Context) error     { return nil }
func (a *Adapter) GetAppliedMigrations(ctx context.Context) (map[string]*time.Time, error) {
	return nil, nil
}
func (a *Adapter) RecordMigration(ctx context.Context, migrationID, name, checksum string) error {
	return nil
}
func (a *Adapter) ExecuteMigration(ctx context.Context, migrationSQL string) error { return nil }
func (a *Adapter) ExecuteAndRecordMigration(ctx context.Context, migrationID, name, checksum string, migrationSQL string) error {
	return nil
}
func (a *Adapter) ExecuteQuery(ctx context.Context, query string) (*common.QueryResult, error) {
	return nil, nil
}
func (a *Adapter) GetCurrentSchema(ctx context.Context) ([]types.SchemaTable, error) { return nil, nil }
func (a *Adapter) GetCurrentEnums(ctx context.Context) ([]types.SchemaEnum, error)   { return nil, nil }
func (a *Adapter) GetTableIndexes(ctx context.Context, tableName string) ([]types.SchemaIndex, error) {
	return nil, nil
}
func (a *Adapter) PullCompleteSchema(ctx context.Context) ([]types.SchemaTable, error) {
	return nil, nil
}
func (a *Adapter) CheckTableExists(ctx context.Context, tableName string) (bool, error) {
	return false, nil
}
func (a *Adapter) CheckColumnExists(ctx context.Context, tableName, columnName string) (bool, error) {
	return false, nil
}
func (a *Adapter) CheckNotNullConstraint(ctx context.Context, tableName, columnName string) (bool, error) {
	return false, nil
}
func (a *Adapter) CheckForeignKeyConstraint(ctx context.Context, tableName, constraintName string) (bool, error) {
	return false, nil
}
func (a *Adapter) CheckUniqueConstraint(ctx context.Context, tableName, constraintName string) (bool, error) {
	return false, nil
}
func (a *Adapter) DropTable(ctx context.Context, tableName string) error                   { return nil }
func (a *Adapter) DropEnum(ctx context.Context, enumName string) error                     { return nil }
func (a *Adapter) GenerateCreateTableSQL(table types.SchemaTable) string                   { return "" }
func (a *Adapter) GenerateAddColumnSQL(tableName string, column types.SchemaColumn) string { return "" }
func (a *Adapter) GenerateDropColumnSQL(tableName, columnName string) string               { return "" }
func (a *Adapter) GenerateAddIndexSQL(index types.SchemaIndex) string                      { return "" }
func (a *Adapter) GenerateDropIndexSQL(indexName string) string                            { return "" }
func (a *Adapter) MapColumnType(dbType string) string                                      { return "string" }
func (a *Adapter) FormatColumnType(column types.SchemaColumn) string                       { return column.Type }
func (a *Adapter) CreateBranchSchema(ctx context.Context, branchName string) error         { return nil }
func (a *Adapter) DropBranchSchema(ctx context.Context, branchName string) error           { return nil }
func (a *Adapter) CloneSchemaToBranch(ctx context.Context, sourceSchema, targetSchema string) error {
	return nil
}
func (a *Adapter) GetSchemaForBranch(ctx context.Context, branchSchema string) ([]types.SchemaTable, error) {
	return nil, nil
}
func (a *Adapter) SetActiveSchema(ctx context.Context, schemaName string) error { return nil }
func (a *Adapter) GetTableNamesInSchema(ctx context.Context, schemaName string) ([]string, error) {
	return nil, nil
}
func (a *Adapter) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}
func (a *Adapter) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}
func (a *Adapter) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}
func (a *Adapter) Begin(ctx context.Context) (*sql.Tx, error) { return nil, nil }

// ExecuteMongoQuery executes a MongoDB query string
func (a *Adapter) ExecuteMongoQuery(ctx context.Context, query string) ([]map[string]interface{}, error) {
	query = strings.TrimSpace(query)
	query = strings.TrimPrefix(query, "db.")

	parts := strings.SplitN(query, ".", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid query format. Use: collection.find({}) or db.collection.find({})")
	}

	collectionName := parts[0]
	operation := parts[1]

	coll := a.database.Collection(collectionName)

	if strings.HasPrefix(operation, "find(") {
		filterStr := extractBetween(operation, "find(", ")")
		if filterStr == "" {
			filterStr = "{}"
		}

		var filter bson.M
		if err := bson.UnmarshalExtJSON([]byte(filterStr), false, &filter); err != nil {
			filter = bson.M{}
		}

		cursor, err := coll.Find(ctx, filter, options.Find().SetLimit(100))
		if err != nil {
			return nil, err
		}
		defer cursor.Close(ctx)

		var results []map[string]interface{}
		for cursor.Next(ctx) {
			var doc bson.M
			if err := cursor.Decode(&doc); err != nil {
				continue
			}
			converted := make(map[string]interface{})
			for k, v := range doc {
				converted[k] = convertBSONValue(v)
			}
			results = append(results, converted)
		}
		return results, nil
	}

	if strings.HasPrefix(operation, "count(") {
		count, err := coll.CountDocuments(ctx, bson.M{})
		if err != nil {
			return nil, err
		}
		return []map[string]interface{}{{"count": count}}, nil
	}

	return nil, fmt.Errorf("unsupported operation. Supported: find({}), count()")
}

func extractBetween(str, start, end string) string {
	startIdx := strings.Index(str, start)
	if startIdx == -1 {
		return ""
	}
	startIdx += len(start)

	endIdx := strings.LastIndex(str, end)
	if endIdx == -1 || endIdx <= startIdx {
		return ""
	}

	return strings.TrimSpace(str[startIdx:endIdx])
}

// MongoDB Studio Methods

// ListCollections returns all collection names
func (a *Adapter) ListCollections(ctx context.Context) ([]string, error) {
	return a.ListCollectionsInDB(ctx, a.dbName)
}

func (a *Adapter) ListCollectionsInDB(ctx context.Context, database string) ([]string, error) {
	fmt.Printf("[DEBUG] ListCollectionsInDB called for database: %s\n", database)
	db := a.client.Database(database)
	names, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		fmt.Printf("[DEBUG] ListCollections error: %v\n", err)
		return nil, err
	}

	var filtered []string
	for _, name := range names {
		if !strings.HasPrefix(name, "system.") {
			filtered = append(filtered, name)
		}
	}
	fmt.Printf("[DEBUG] Found %d collections: %v\n", len(filtered), filtered)
	return filtered, nil
}

// GetCollectionStats returns statistics for a collection
func (a *Adapter) GetCollectionStats(ctx context.Context, collection string) (map[string]interface{}, error) {
	var result bson.M
	err := a.database.RunCommand(ctx, bson.D{{Key: "collStats", Value: collection}}).Decode(&result)
	if err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})
	for k, v := range result {
		stats[k] = v
	}
	return stats, nil
}

// FindDocuments finds documents in a collection with pagination
func (a *Adapter) FindDocuments(ctx context.Context, collection string, filter bson.M, skip, limit int64) ([]map[string]interface{}, error) {
	return a.FindDocumentsInDB(ctx, a.dbName, collection, filter, skip, limit)
}

// FindDocumentsInDB finds documents in a specific database and collection with pagination
func (a *Adapter) FindDocumentsInDB(ctx context.Context, database, collection string, filter bson.M, skip, limit int64) ([]map[string]interface{}, error) {
	fmt.Printf("[DEBUG FindDocumentsInDB] Database: %s, Collection: %s, Skip: %d, Limit: %d\n", database, collection, skip, limit)
	db := a.client.Database(database)
	coll := db.Collection(collection)
	opts := options.Find().SetSkip(skip).SetLimit(limit)

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		converted := make(map[string]interface{})
		for k, v := range doc {
			converted[k] = convertBSONValue(v)
		}
		results = append(results, converted)
	}
	return results, nil
}

// CountDocuments counts documents in a collection
func (a *Adapter) CountDocuments(ctx context.Context, collection string, filter bson.M) (int64, error) {
	return a.CountDocumentsInDB(ctx, a.dbName, collection, filter)
}

// CountDocumentsInDB counts documents in a specific database and collection
func (a *Adapter) CountDocumentsInDB(ctx context.Context, database, collection string, filter bson.M) (int64, error) {
	fmt.Printf("[DEBUG CountDocumentsInDB] Database: %s, Collection: %s\n", database, collection)
	db := a.client.Database(database)
	coll := db.Collection(collection)
	count, err := coll.CountDocuments(ctx, filter)
	fmt.Printf("[DEBUG CountDocumentsInDB] Result: %d documents, Error: %v\n", count, err)
	return count, err
}

// InsertDocument inserts a document into a collection
func (a *Adapter) InsertDocument(ctx context.Context, collection string, document interface{}) (string, error) {
	coll := a.database.Collection(collection)
	result, err := coll.InsertOne(ctx, document)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v", result.InsertedID), nil
}

// UpdateDocument updates a document in a collection
func (a *Adapter) UpdateDocument(ctx context.Context, collection string, id string, update interface{}) error {
	coll := a.database.Collection(collection)
	objectID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	_, err = coll.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

// DeleteDocument deletes a document from a collection
func (a *Adapter) DeleteDocument(ctx context.Context, collection string, id string) error {
	coll := a.database.Collection(collection)
	objectID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	_, err = coll.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

// CreateCollection creates a new collection
func (a *Adapter) CreateCollection(ctx context.Context, name string, options interface{}) error {
	return a.database.CreateCollection(ctx, name)
}

// DropCollection drops a collection
func (a *Adapter) DropCollection(ctx context.Context, name string) error {
	return a.database.Collection(name).Drop(ctx)
}

// Aggregate runs an aggregation pipeline
func (a *Adapter) Aggregate(ctx context.Context, collection string, pipeline interface{}) ([]map[string]interface{}, error) {
	coll := a.database.Collection(collection)
	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		converted := make(map[string]interface{})
		for k, v := range doc {
			converted[k] = convertBSONValue(v)
		}
		results = append(results, converted)
	}
	return results, nil
}

// ListIndexes lists all indexes for a collection
func (a *Adapter) ListIndexes(ctx context.Context, collection string) ([]map[string]interface{}, error) {
	coll := a.database.Collection(collection)
	cursor, err := coll.Indexes().List(ctx)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var indexes []map[string]interface{}
	for cursor.Next(ctx) {
		var index bson.M
		if err := cursor.Decode(&index); err != nil {
			continue
		}
		converted := make(map[string]interface{})
		for k, v := range index {
			converted[k] = convertBSONValue(v)
		}
		indexes = append(indexes, converted)
	}
	return indexes, nil
}

// CreateIndex creates a new index on a collection
func (a *Adapter) CreateIndex(ctx context.Context, collection string, keys map[string]interface{}, unique bool) error {
	coll := a.database.Collection(collection)
	indexModel := mongo.IndexModel{
		Keys:    keys,
		Options: options.Index().SetUnique(unique),
	}
	_, err := coll.Indexes().CreateOne(ctx, indexModel)
	return err
}

// DropIndex drops an index from a collection
func (a *Adapter) DropIndex(ctx context.Context, collection string, indexName string) error {
	coll := a.database.Collection(collection)
	_, err := coll.Indexes().DropOne(ctx, indexName)
	return err
}

// GetDatabaseStats returns database statistics
func (a *Adapter) GetDatabaseStats(ctx context.Context) (map[string]interface{}, error) {
	var result bson.M
	err := a.database.RunCommand(ctx, bson.D{{Key: "dbStats", Value: 1}}).Decode(&result)
	if err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})
	for k, v := range result {
		stats[k] = v
	}
	return stats, nil
}

// ListDatabases lists all databases
func (a *Adapter) ListDatabases(ctx context.Context) ([]map[string]interface{}, error) {
	databases, err := a.client.ListDatabases(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0)
	for _, db := range databases.Databases {
		result = append(result, map[string]interface{}{
			"name":       db.Name,
			"sizeOnDisk": db.SizeOnDisk,
			"empty":      db.Empty,
		})
	}
	return result, nil
}

// SwitchDatabase switches to a different database
func (a *Adapter) SwitchDatabase(dbName string) error {
	fmt.Printf("[DEBUG SwitchDatabase BEFORE] a.dbName was: '%s' (len=%d)\n", a.dbName, len(a.dbName))
	fmt.Printf("[DEBUG SwitchDatabase] Switching to database: '%s' (len=%d)\n", dbName, len(dbName))
	a.database = a.client.Database(dbName)
	a.dbName = dbName
	fmt.Printf("[DEBUG SwitchDatabase AFTER] a.dbName is now: '%s' (len=%d)\n", a.dbName, len(a.dbName))
	return nil
}

func parseObjectID(id string) (interface{}, error) {
	// Try to parse as MongoDB ObjectID
	if len(id) == 24 {
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return id, nil // Fallback to string if parsing fails
		}
		return oid, nil
	}
	return id, nil
}
