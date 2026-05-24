//go:build cli || tui

package logs

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// CLI mode: log only to file, let CLI handle stdout.

func InitLogs() *os.File {

	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Error creating log directory: %v\n", err)
		os.Exit(1)
	}

	timestamp := time.Now().Format("20060102_150405")
	path := fmt.Sprintf("%s/marabu_%s.log", logDir, timestamp)

	link := fmt.Sprintf("%s/latest.log", logDir)
	if _, err := os.Lstat(link); err == nil {
		os.Remove(link)
	}

	// Create a symlink to the latest log file for easy access (used by the wrapper script)
	err := os.Symlink(filepath.Base(path), link)
	if err != nil {
		fmt.Printf("Error creating symlink for latest log: %v\n", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	log.SetOutput(file)
	log.SetFlags(log.Ltime)

	fmt.Fprintf(file, "%s%s\t--- Marabu Logs @ %s%s%s ---%s\n", BOLD, MAGENTA, BLUE, time.Now().Format(time.RFC3339), MAGENTA, RESET)

	return file
}
