package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/zyedidia/eget/home"
)

// LogEntry represents a single log entry for a binary operation
type LogEntry struct {
	Timestamp time.Time
	Repo      string
	Path      string
	Action    string
}

// GetLogDir returns the appropriate log directory based on the OS
func GetLogDir() (string, error) {
	var logDir string
	
	if runtime.GOOS == "windows" {
		// Windows: use %LOCALAPPDATA%\eget\logs
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			homeDir, err := home.Home()
			if err != nil {
				return "", fmt.Errorf("could not determine home directory: %w", err)
			}
			localAppData = filepath.Join(homeDir, "AppData", "Local")
		}
		logDir = filepath.Join(localAppData, "eget", "logs")
	} else {
		// Unix-like systems: use ~/.local/share/eget/logs
		homeDir, err := home.Home()
		if err != nil {
			return "", fmt.Errorf("could not determine home directory: %w", err)
		}
		logDir = filepath.Join(homeDir, ".local", "share", "eget", "logs")
	}
	
	return logDir, nil
}

// GetLogFilePath returns the full path to the log file
func GetLogFilePath() (string, error) {
	logDir, err := GetLogDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(logDir, "eget.log"), nil
}

// ensureLogDir creates the log directory if it doesn't exist
func ensureLogDir() error {
	logDir, err := GetLogDir()
	if err != nil {
		return err
	}
	
	return os.MkdirAll(logDir, 0755)
}

// LogOperation logs a binary operation to the log file
func LogOperation(repo, path, action string) error {
	// Ensure log directory exists
	if err := ensureLogDir(); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	
	logFile, err := GetLogFilePath()
	if err != nil {
		return fmt.Errorf("failed to get log file path: %w", err)
	}
	
	// Open file in append mode, create if doesn't exist
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()
	
	// Format: timestamp\trepo\tpath\taction
	timestamp := time.Now().UTC().Format(time.RFC3339)
	logLine := fmt.Sprintf("%s\t%s\t%s\t%s\n", timestamp, repo, path, action)
	
	_, err = f.WriteString(logLine)
	if err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}
	
	return nil
}

// ReadLogs reads all log entries from the log file
func ReadLogs() ([]LogEntry, error) {
	logFile, err := GetLogFilePath()
	if err != nil {
		return nil, err
	}
	
	data, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []LogEntry{}, nil
		}
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}
	
	lines := strings.Split(string(data), "\n")
	entries := make([]LogEntry, 0, len(lines))
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		parts := strings.Split(line, "\t")
		if len(parts) != 4 {
			continue // skip malformed lines
		}
		
		timestamp, err := time.Parse(time.RFC3339, parts[0])
		if err != nil {
			continue // skip lines with invalid timestamps
		}
		
		entries = append(entries, LogEntry{
			Timestamp: timestamp,
			Repo:      parts[1],
			Path:      parts[2],
			Action:    parts[3],
		})
	}
	
	return entries, nil
}

// FormatLogEntry formats a log entry for display
func FormatLogEntry(entry LogEntry) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s",
		entry.Timestamp.Format(time.RFC3339),
		entry.Repo,
		entry.Path,
		entry.Action)
}

// PrintLogs prints all log entries
func PrintLogs() error {
	entries, err := ReadLogs()
	if err != nil {
		return err
	}
	
	for _, entry := range entries {
		fmt.Println(FormatLogEntry(entry))
	}
	
	return nil
}
