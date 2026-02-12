package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ServerConfig struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	ServerAddr string        `json:"serverAddr"`
	ServerPort int           `json:"serverPort"`
	AuthToken  string        `json:"authToken,omitempty"`
	AuthMethod string        `json:"authMethod,omitempty"`
	TLSEnable  bool          `json:"tlsEnable,omitempty"`
	User       string        `json:"user,omitempty"`
	Proxies    []ProxyConfig `json:"proxies"`
	CreatedAt  string        `json:"createdAt"`
	UpdatedAt  string        `json:"updatedAt"`
}

type ProxyConfig struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Type          string   `json:"type"` // tcp, udp, http, https
	LocalIP       string   `json:"localIP"`
	LocalPort     int      `json:"localPort"`
	RemotePort    int      `json:"remotePort,omitempty"`
	CustomDomains []string `json:"customDomains,omitempty"`
	Subdomain     string   `json:"subdomain,omitempty"`
}

type ConfigManager struct {
	dataDir string
	mu      sync.RWMutex
}

func NewConfigManager(dataDir string) *ConfigManager {
	return &ConfigManager{dataDir: dataDir}
}

func (cm *ConfigManager) configFilePath() string {
	return filepath.Join(cm.dataDir, "servers.json")
}

func (cm *ConfigManager) Load() ([]ServerConfig, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	b, err := os.ReadFile(cm.configFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return []ServerConfig{}, nil
		}
		return nil, err
	}

	var servers []ServerConfig
	if err := json.Unmarshal(b, &servers); err != nil {
		return nil, err
	}
	return servers, nil
}

func (cm *ConfigManager) Save(servers []ServerConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	b, err := json.MarshalIndent(servers, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cm.configFilePath(), b, 0644)
}

func (cm *ConfigManager) GetServer(id string) (*ServerConfig, error) {
	servers, err := cm.Load()
	if err != nil {
		return nil, err
	}
	for i := range servers {
		if servers[i].ID == id {
			return &servers[i], nil
		}
	}
	return nil, fmt.Errorf("server not found: %s", id)
}

func (cm *ConfigManager) CreateServer(cfg ServerConfig) error {
	servers, err := cm.Load()
	if err != nil {
		return err
	}

	cfg.ID = generateID()
	cfg.CreatedAt = time.Now().Format(time.RFC3339)
	cfg.UpdatedAt = cfg.CreatedAt
	if cfg.Proxies == nil {
		cfg.Proxies = []ProxyConfig{}
	}

	servers = append(servers, cfg)
	return cm.Save(servers)
}

func (cm *ConfigManager) UpdateServer(id string, cfg ServerConfig) error {
	servers, err := cm.Load()
	if err != nil {
		return err
	}

	for i := range servers {
		if servers[i].ID == id {
			cfg.ID = id
			cfg.CreatedAt = servers[i].CreatedAt
			cfg.UpdatedAt = time.Now().Format(time.RFC3339)
			cfg.Proxies = servers[i].Proxies
			servers[i] = cfg
			return cm.Save(servers)
		}
	}
	return fmt.Errorf("server not found: %s", id)
}

func (cm *ConfigManager) DeleteServer(id string) error {
	servers, err := cm.Load()
	if err != nil {
		return err
	}

	for i := range servers {
		if servers[i].ID == id {
			servers = append(servers[:i], servers[i+1:]...)
			return cm.Save(servers)
		}
	}
	return fmt.Errorf("server not found: %s", id)
}

func (cm *ConfigManager) AddProxy(serverID string, proxy ProxyConfig) error {
	servers, err := cm.Load()
	if err != nil {
		return err
	}

	for i := range servers {
		if servers[i].ID == serverID {
			proxy.ID = generateID()
			servers[i].Proxies = append(servers[i].Proxies, proxy)
			servers[i].UpdatedAt = time.Now().Format(time.RFC3339)
			return cm.Save(servers)
		}
	}
	return fmt.Errorf("server not found: %s", serverID)
}

func (cm *ConfigManager) UpdateProxy(serverID, proxyID string, proxy ProxyConfig) error {
	servers, err := cm.Load()
	if err != nil {
		return err
	}

	for i := range servers {
		if servers[i].ID == serverID {
			for j := range servers[i].Proxies {
				if servers[i].Proxies[j].ID == proxyID {
					proxy.ID = proxyID
					servers[i].Proxies[j] = proxy
					servers[i].UpdatedAt = time.Now().Format(time.RFC3339)
					return cm.Save(servers)
				}
			}
			return fmt.Errorf("proxy not found: %s", proxyID)
		}
	}
	return fmt.Errorf("server not found: %s", serverID)
}

func (cm *ConfigManager) DeleteProxy(serverID, proxyID string) error {
	servers, err := cm.Load()
	if err != nil {
		return err
	}

	for i := range servers {
		if servers[i].ID == serverID {
			for j := range servers[i].Proxies {
				if servers[i].Proxies[j].ID == proxyID {
					servers[i].Proxies[j] = servers[i].Proxies[len(servers[i].Proxies)-1]
					servers[i].Proxies = servers[i].Proxies[:len(servers[i].Proxies)-1]
					servers[i].UpdatedAt = time.Now().Format(time.RFC3339)
					return cm.Save(servers)
				}
			}
			return fmt.Errorf("proxy not found: %s", proxyID)
		}
	}
	return fmt.Errorf("server not found: %s", serverID)
}

// GenerateToml generates frpc.toml content for a server
func (cm *ConfigManager) GenerateToml(server *ServerConfig) string {
	var b strings.Builder

	b.WriteString("# Auto-generated by fnos-frpc-gui\n\n")

	// Global config
	b.WriteString(fmt.Sprintf("serverAddr = \"%s\"\n", server.ServerAddr))
	b.WriteString(fmt.Sprintf("serverPort = %d\n", server.ServerPort))

	if server.User != "" {
		b.WriteString(fmt.Sprintf("user = \"%s\"\n", server.User))
	}

	if server.AuthToken != "" {
		method := server.AuthMethod
		if method == "" {
			method = "token"
		}
		b.WriteString(fmt.Sprintf("\n[auth]\nmethod = \"%s\"\n", method))
		b.WriteString(fmt.Sprintf("token = \"%s\"\n", server.AuthToken))
	}

	if server.TLSEnable {
		b.WriteString("\n[transport.tls]\nenable = true\n")
	}

	// Proxies
	for _, p := range server.Proxies {
		b.WriteString("\n[[proxies]]\n")
		b.WriteString(fmt.Sprintf("name = \"%s\"\n", p.Name))
		b.WriteString(fmt.Sprintf("type = \"%s\"\n", p.Type))
		b.WriteString(fmt.Sprintf("localIP = \"%s\"\n", p.LocalIP))
		b.WriteString(fmt.Sprintf("localPort = %d\n", p.LocalPort))

		switch p.Type {
		case "tcp", "udp":
			if p.RemotePort > 0 {
				b.WriteString(fmt.Sprintf("remotePort = %d\n", p.RemotePort))
			}
		case "http", "https":
			if len(p.CustomDomains) > 0 {
				domains := make([]string, len(p.CustomDomains))
				for i, d := range p.CustomDomains {
					domains[i] = fmt.Sprintf("\"%s\"", d)
				}
				b.WriteString(fmt.Sprintf("customDomains = [%s]\n", strings.Join(domains, ", ")))
			}
			if p.Subdomain != "" {
				b.WriteString(fmt.Sprintf("subdomain = \"%s\"\n", p.Subdomain))
			}
		}
	}

	return b.String()
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
