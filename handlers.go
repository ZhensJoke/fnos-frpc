package main

import (
	"encoding/json"
	"io"
	"net/http"
)

type Handler struct {
	config  *ConfigManager
	process *ProcessManager
	version *VersionManager
	auth    *AuthManager
}

func NewHandler(config *ConfigManager, process *ProcessManager, version *VersionManager, auth *AuthManager) *Handler {
	return &Handler{config: config, process: process, version: version, auth: auth}
}

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	jsonResponse(w, status, map[string]string{"error": msg})
}

// --- Auth ---

func (h *Handler) AuthStatus(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, 200, map[string]interface{}{
		"needSetup":     !h.auth.IsSetup(),
		"frpcInstalled": h.version.IsInstalled(),
	})
}

func (h *Handler) AuthSetup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Password == "" {
		jsonError(w, 400, "password is required")
		return
	}

	if err := h.auth.Setup(body.Password); err != nil {
		jsonError(w, 400, err.Error())
		return
	}

	token := h.auth.CreateSession()
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
	})
	jsonResponse(w, 200, map[string]string{"token": token})
}

func (h *Handler) AuthLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Password == "" {
		jsonError(w, 400, "password is required")
		return
	}

	if !h.auth.Verify(body.Password) {
		jsonError(w, 401, "incorrect password")
		return
	}

	token := h.auth.CreateSession()
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
	})
	jsonResponse(w, 200, map[string]string{"token": token})
}

// --- Servers ---

func (h *Handler) ListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := h.config.Load()
	if err != nil {
		jsonError(w, 500, err.Error())
		return
	}

	// Attach running status
	type ServerWithStatus struct {
		ServerConfig
		Running bool `json:"running"`
		PID     int  `json:"pid"`
	}

	result := make([]ServerWithStatus, len(servers))
	for i, s := range servers {
		running, pid := h.process.Status(s.ID)
		result[i] = ServerWithStatus{ServerConfig: s, Running: running, PID: pid}
	}

	jsonResponse(w, 200, result)
}

func (h *Handler) CreateServer(w http.ResponseWriter, r *http.Request) {
	var cfg ServerConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		jsonError(w, 400, "invalid request body")
		return
	}

	if cfg.Name == "" || cfg.ServerAddr == "" || cfg.ServerPort == 0 {
		jsonError(w, 400, "name, serverAddr, and serverPort are required")
		return
	}

	if err := h.config.CreateServer(cfg); err != nil {
		jsonError(w, 500, err.Error())
		return
	}

	jsonResponse(w, 201, map[string]string{"status": "created"})
}

func (h *Handler) UpdateServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var cfg ServerConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		jsonError(w, 400, "invalid request body")
		return
	}

	if err := h.config.UpdateServer(id, cfg); err != nil {
		jsonError(w, 500, err.Error())
		return
	}

	jsonResponse(w, 200, map[string]string{"status": "updated"})
}

func (h *Handler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Stop if running
	h.process.Stop(id)

	if err := h.config.DeleteServer(id); err != nil {
		jsonError(w, 500, err.Error())
		return
	}

	jsonResponse(w, 200, map[string]string{"status": "deleted"})
}

// --- Proxies ---

func (h *Handler) ListProxies(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	server, err := h.config.GetServer(id)
	if err != nil {
		jsonError(w, 404, err.Error())
		return
	}
	jsonResponse(w, 200, server.Proxies)
}

func (h *Handler) CreateProxy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var proxy ProxyConfig
	if err := json.NewDecoder(r.Body).Decode(&proxy); err != nil {
		jsonError(w, 400, "invalid request body")
		return
	}

	if proxy.Name == "" || proxy.Type == "" {
		jsonError(w, 400, "name and type are required")
		return
	}

	if err := h.config.AddProxy(id, proxy); err != nil {
		jsonError(w, 500, err.Error())
		return
	}

	jsonResponse(w, 201, map[string]string{"status": "created"})
}

func (h *Handler) UpdateProxy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	pid := r.PathValue("pid")

	var proxy ProxyConfig
	if err := json.NewDecoder(r.Body).Decode(&proxy); err != nil {
		jsonError(w, 400, "invalid request body")
		return
	}

	if err := h.config.UpdateProxy(id, pid, proxy); err != nil {
		jsonError(w, 500, err.Error())
		return
	}

	jsonResponse(w, 200, map[string]string{"status": "updated"})
}

func (h *Handler) DeleteProxy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	pid := r.PathValue("pid")

	if err := h.config.DeleteProxy(id, pid); err != nil {
		jsonError(w, 500, err.Error())
		return
	}

	jsonResponse(w, 200, map[string]string{"status": "deleted"})
}

// --- Process Control ---

func (h *Handler) StartServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	server, err := h.config.GetServer(id)
	if err != nil {
		jsonError(w, 404, err.Error())
		return
	}

	toml := h.config.GenerateToml(server)
	if err := h.process.Start(id, toml); err != nil {
		jsonError(w, 500, err.Error())
		return
	}

	jsonResponse(w, 200, map[string]string{"status": "started"})
}

func (h *Handler) StopServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.process.Stop(id); err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	jsonResponse(w, 200, map[string]string{"status": "stopped"})
}

func (h *Handler) ServerStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	running, pid := h.process.Status(id)
	jsonResponse(w, 200, map[string]interface{}{
		"running": running,
		"pid":     pid,
	})
}

func (h *Handler) ServerLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	logs, err := h.process.GetLogs(id, 200)
	if err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	jsonResponse(w, 200, map[string]string{"logs": logs})
}

// --- FRPC Version ---

func (h *Handler) FrpcVersion(w http.ResponseWriter, r *http.Request) {
	version, err := h.version.GetCurrentVersion()
	installed := h.version.IsInstalled()
	if err != nil {
		jsonResponse(w, 200, map[string]interface{}{
			"installed": installed,
			"version":   "",
		})
		return
	}
	jsonResponse(w, 200, map[string]interface{}{
		"installed": installed,
		"version":   version,
	})
}

func (h *Handler) FrpcLatest(w http.ResponseWriter, r *http.Request) {
	release, err := h.version.GetLatestRelease()
	if err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	jsonResponse(w, 200, map[string]string{
		"version": release.TagName,
	})
}

func (h *Handler) FrpcInstall(w http.ResponseWriter, r *http.Request) {
	version, err := h.version.InstallFromGitHub()
	if err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	jsonResponse(w, 200, map[string]string{
		"status":  "installed",
		"version": version,
	})
}

func (h *Handler) FrpcUpload(w http.ResponseWriter, r *http.Request) {
	// Max 100MB upload
	r.ParseMultipartForm(100 << 20)

	file, _, err := r.FormFile("file")
	if err != nil {
		jsonError(w, 400, "file upload required")
		return
	}
	defer file.Close()

	// Read all into a temporary reader
	version, err := h.version.InstallFromUpload(io.Reader(file))
	if err != nil {
		jsonError(w, 500, err.Error())
		return
	}

	jsonResponse(w, 200, map[string]string{
		"status":  "installed",
		"version": version,
	})
}
