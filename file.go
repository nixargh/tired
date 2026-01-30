package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

func readTimesheet(path string) []string {
	clog.WithFields(log.Fields{"path": path}).Info("Reading timesheet.")

	absPath, _ := filepath.Abs(path)
	content, err := os.ReadFile(absPath)

	if err != nil {
		clog.WithFields(log.Fields{"error": err}).Fatal("Failed to read timesheet.")
	}

	timesheet := strings.Split(string(content), "\n")

	return timesheet
}

func writeTimesheet(path string, content []string) {
	clog.WithFields(log.Fields{"path": path}).Info("Updating timesheet file.")

	absPath, _ := filepath.Abs(path)
	bkpPath := fmt.Sprintf("%s.bak", absPath)

	// Move old file as backup
	err := os.Rename(absPath, bkpPath)
	if err != nil {
		clog.WithFields(log.Fields{"error": err}).Fatal("Failed to rename the file.")
	}

	// create the file
	f, err := os.Create(absPath)
	if err != nil {
		clog.WithFields(log.Fields{"error": err}).Fatal("Failed to create the file.")
	}

	defer func() {
		err := f.Close()
		if err == nil {
			clog.WithFields(log.Fields{"error": err}).Fatal("Failed to close the file.")
		}
	}()

	// write a string
	for i, line := range content {
		// Remove marker
		if line == defaultMarker {
			continue
		}

		_, lErr := f.WriteString(line + "\n")

		if lErr != nil {
			clog.WithFields(log.Fields{
				"number": i,
				"error":  lErr,
			}).Fatal("Failed to write the line.")
		}
	}

	// Re-add marker to the end of file
	_, lErr := f.WriteString(defaultMarker)
	if lErr != nil {
		clog.WithFields(log.Fields{"error": lErr}).Fatal("Failed to write the marker.")
	}
}
