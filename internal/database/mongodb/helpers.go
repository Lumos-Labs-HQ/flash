package mongodb

import (
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// inferBSONType infers the MongoDB type from a Go value
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

// convertBSONValue converts BSON values to standard Go types
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

// extractBetween extracts a substring between two delimiters
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

// parseObjectID parses a string ID to ObjectID or returns the string as-is
func parseObjectID(id string) (interface{}, error) {
	if len(id) == 24 {
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return id, nil
		}
		return oid, nil
	}
	return id, nil
}
