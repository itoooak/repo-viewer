package main

import (
	"flag"
	"log"
	"net/http"

	"repo-viewer/internal/handler"
)

func main() {
	repoDir := flag.String("dir", "./repos", "repository directory path")
	port := flag.String("port", "8080", "server port")
	flag.Parse()

	h, err := handler.New(*repoDir)
	if err != nil {
		log.Fatalf("failed to initialize handler: %v", err)
	}

	// Page handlers
	http.HandleFunc("GET /{$}", h.HandleIndex)
	http.HandleFunc("GET /r/{repo}", h.HandleRepo)
	http.HandleFunc("GET /r/{repo}/t/{ref}/{path...}", h.HandleTree)
	http.HandleFunc("GET /r/{repo}/b/{ref}/{path...}", h.HandleBlob)

	// API handlers
	http.HandleFunc("GET /r/{repo}/api/files", h.HandleFileListAPI)

	// Fallback
	http.HandleFunc("/", h.HandleNotFound)

	addr := ":" + *port
	log.Printf("starting server: http://localhost%s", addr)
	log.Printf("repository directory: %s", *repoDir)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
