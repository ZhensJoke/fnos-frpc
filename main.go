package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	port := os.Getenv("WEB_PORT")
	if port == "" {
		port = "7500"
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/app/data"
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize managers
	configMgr := NewConfigManager(dataDir)
	processMgr := NewProcessManager(dataDir)
	versionMgr := NewVersionManager(dataDir)
	authMgr := NewAuthManager(dataDir)

	// Create handler
	handler := NewHandler(configMgr, processMgr, versionMgr, authMgr)

	// Setup routes
	mux := http.NewServeMux()

	// Auth routes (no auth middleware)
	mux.HandleFunc("GET /api/auth/status", handler.AuthStatus)
	mux.HandleFunc("POST /api/auth/setup", handler.AuthSetup)
	mux.HandleFunc("POST /api/auth/login", handler.AuthLogin)

	// Protected API routes
	mux.Handle("GET /api/servers", authMgr.Middleware(http.HandlerFunc(handler.ListServers)))
	mux.Handle("POST /api/servers", authMgr.Middleware(http.HandlerFunc(handler.CreateServer)))
	mux.Handle("PUT /api/servers/{id}", authMgr.Middleware(http.HandlerFunc(handler.UpdateServer)))
	mux.Handle("DELETE /api/servers/{id}", authMgr.Middleware(http.HandlerFunc(handler.DeleteServer)))

	mux.Handle("GET /api/servers/{id}/proxies", authMgr.Middleware(http.HandlerFunc(handler.ListProxies)))
	mux.Handle("POST /api/servers/{id}/proxies", authMgr.Middleware(http.HandlerFunc(handler.CreateProxy)))
	mux.Handle("PUT /api/servers/{id}/proxies/{pid}", authMgr.Middleware(http.HandlerFunc(handler.UpdateProxy)))
	mux.Handle("DELETE /api/servers/{id}/proxies/{pid}", authMgr.Middleware(http.HandlerFunc(handler.DeleteProxy)))

	mux.Handle("POST /api/servers/{id}/start", authMgr.Middleware(http.HandlerFunc(handler.StartServer)))
	mux.Handle("POST /api/servers/{id}/stop", authMgr.Middleware(http.HandlerFunc(handler.StopServer)))
	mux.Handle("GET /api/servers/{id}/status", authMgr.Middleware(http.HandlerFunc(handler.ServerStatus)))
	mux.Handle("GET /api/servers/{id}/logs", authMgr.Middleware(http.HandlerFunc(handler.ServerLogs)))

	mux.Handle("GET /api/frpc/version", authMgr.Middleware(http.HandlerFunc(handler.FrpcVersion)))
	mux.Handle("GET /api/frpc/latest", authMgr.Middleware(http.HandlerFunc(handler.FrpcLatest)))
	mux.Handle("POST /api/frpc/install", authMgr.Middleware(http.HandlerFunc(handler.FrpcInstall)))
	mux.Handle("POST /api/frpc/upload", authMgr.Middleware(http.HandlerFunc(handler.FrpcUpload)))

	// Static files (embedded in binary)
	staticSub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("Failed to load embedded static files: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticSub)))

	log.Printf("fnos-frpc-gui starting on port %s", port)
	log.Printf("Data directory: %s", dataDir)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
