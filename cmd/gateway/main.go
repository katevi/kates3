package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func postFileHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON body
	var data map[string]string
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Set response type
	w.Header().Set("Content-Type", "application/json")

	// Send response
	json.NewEncoder(w).Encode(map[string]string{
		"status": "received",
		"data":   data["message"],
	})
}

func getFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filename := r.URL.Query().Get("filename")
	if filename == "" {
		http.Error(w, `{"error": "Filename parameter is required"}`, http.StatusBadRequest)
		return
	}
	fmt.Println("filename = ", filename)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"filename": filename,
	})
}

// Main handler that routes requests
func apiHandler(w http.ResponseWriter, r *http.Request) {
	// Only handle /api/file path
	if !strings.HasPrefix(r.URL.Path, "/api/file") {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case "POST":
		postFileHandler(w, r)
	case "GET":
		getFileHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/file", apiHandler)

	log.Println("Server starting on :8080")
	log.Println("POST /api/file - Upload file")
	log.Println("GET  /api/file?filename=test.txt - Get file")

	http.ListenAndServe(":8080", mux)
}
