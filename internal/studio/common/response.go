package common

import (
	"encoding/json"
	"net/http"
)

// Map replaces fiber.Map
type Map = map[string]any

// writeJSON sets headers, status, and encodes JSON to the response writer
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// JSON sends a success response with data
func JSON(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, Response{Success: true, Data: data})
}

// JSONMessage sends a success response with message
func JSONMessage(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusOK, Response{Success: true, Message: message})
}

// JSONError sends an error response
func JSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, Response{Success: false, Message: message})
}

// Result writes data as a success response, or a 500 error response if err is
// non-nil. It collapses the ubiquitous studio handler pattern:
//
//	x, err := s.service.Foo()
//	if err != nil { JSONError(w, 500, err.Error()); return }
//	JSON(w, x)
func Result(w http.ResponseWriter, data any, err error) {
	if err != nil {
		JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	JSON(w, data)
}

// ResultMessage writes a success message, or a 500 error response if err is
// non-nil. It is the message-returning counterpart to Result for handlers whose
// success path carries no data.
func ResultMessage(w http.ResponseWriter, message string, err error) {
	if err != nil {
		JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	JSONMessage(w, message)
}

// JSONMap sends an arbitrary map as JSON (replaces JSONFiberMap)
func JSONMap(w http.ResponseWriter, data Map) {
	writeJSON(w, http.StatusOK, data)
}

// JSONRaw sends any data as JSON without wrapping in Response
func JSONRaw(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, data)
}

// ParseJSON decodes the JSON request body into target
func ParseJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

// Query returns the query parameter value, or defaultValue if empty/missing
func Query(r *http.Request, key, defaultValue string) string {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultValue
	}
	return v
}
