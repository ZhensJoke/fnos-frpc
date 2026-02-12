package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type ProcessInfo struct {
	ServerID string
	Cmd      *exec.Cmd
	LogFile  string
	Running  bool
}

type ProcessManager struct {
	dataDir   string
	processes map[string]*ProcessInfo
	mu        sync.RWMutex
}

func NewProcessManager(dataDir string) *ProcessManager {
	logsDir := filepath.Join(dataDir, "logs")
	os.MkdirAll(logsDir, 0755)
	confDir := filepath.Join(dataDir, "conf")
	os.MkdirAll(confDir, 0755)

	return &ProcessManager{
		dataDir:   dataDir,
		processes: make(map[string]*ProcessInfo),
	}
}

func (pm *ProcessManager) frpcPath() string {
	name := "frpc"
	if runtime.GOOS == "windows" {
		name = "frpc.exe"
	}
	return filepath.Join(pm.dataDir, "frpc", name)
}

func (pm *ProcessManager) confPath(serverID string) string {
	return filepath.Join(pm.dataDir, "conf", serverID+".toml")
}

func (pm *ProcessManager) logPath(serverID string) string {
	return filepath.Join(pm.dataDir, "logs", serverID+".log")
}

func (pm *ProcessManager) Start(serverID string, tomlContent string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if already running
	if info, ok := pm.processes[serverID]; ok && info.Running {
		return fmt.Errorf("server %s is already running", serverID)
	}

	// Check frpc binary
	frpcPath := pm.frpcPath()
	if _, err := os.Stat(frpcPath); err != nil {
		return fmt.Errorf("frpc binary not found, please install frpc first")
	}

	// Write config file
	confFile := pm.confPath(serverID)
	if err := os.WriteFile(confFile, []byte(tomlContent), 0644); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}

	// Open log file
	logFile := pm.logPath(serverID)
	lf, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %v", err)
	}

	// Start frpc
	cmd := exec.Command(frpcPath, "-c", confFile)
	cmd.Stdout = lf
	cmd.Stderr = lf

	if err := cmd.Start(); err != nil {
		lf.Close()
		return fmt.Errorf("failed to start frpc: %v", err)
	}

	info := &ProcessInfo{
		ServerID: serverID,
		Cmd:      cmd,
		LogFile:  logFile,
		Running:  true,
	}
	pm.processes[serverID] = info

	// Monitor process in background
	go func() {
		cmd.Wait()
		lf.Close()
		pm.mu.Lock()
		if p, ok := pm.processes[serverID]; ok && p.Cmd == cmd {
			p.Running = false
		}
		pm.mu.Unlock()
		log.Printf("frpc process for server %s exited", serverID)
	}()

	log.Printf("frpc started for server %s (PID: %d)", serverID, cmd.Process.Pid)
	return nil
}

func (pm *ProcessManager) Stop(serverID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	info, ok := pm.processes[serverID]
	if !ok || !info.Running {
		return fmt.Errorf("server %s is not running", serverID)
	}

	if err := info.Cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to stop frpc: %v", err)
	}

	info.Running = false
	log.Printf("frpc stopped for server %s", serverID)
	return nil
}

func (pm *ProcessManager) Status(serverID string) (bool, int) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	info, ok := pm.processes[serverID]
	if !ok {
		return false, 0
	}

	pid := 0
	if info.Cmd != nil && info.Cmd.Process != nil {
		pid = info.Cmd.Process.Pid
	}
	return info.Running, pid
}

func (pm *ProcessManager) GetLogs(serverID string, lines int) (string, error) {
	logFile := pm.logPath(serverID)
	b, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	content := string(b)
	if lines > 0 {
		allLines := strings.Split(content, "\n")
		if len(allLines) > lines {
			allLines = allLines[len(allLines)-lines:]
		}
		content = strings.Join(allLines, "\n")
	}

	return content, nil
}

func (pm *ProcessManager) StopAll() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for id, info := range pm.processes {
		if info.Running {
			info.Cmd.Process.Kill()
			info.Running = false
			log.Printf("frpc stopped for server %s (shutdown)", id)
		}
	}
}
