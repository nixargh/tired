package main

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func readCalendar(path string) {
	clog.WithFields(log.Fields{"path": path}).Info("Reading calendar.")

	absPath, _ := filepath.Abs(path)
	content, err := ioutil.ReadFile(absPath)

	if err != nil {
		clog.WithFields(log.Fields{"error": err}).Fatal("Failed to read calendar.")
	}

	var calendar interface{}
	err = json.Unmarshal([]byte(content), &calendar)
	if err != nil {
		clog.WithFields(log.Fields{
			"error": err,
		}).Fatal("Error during convering JSON to data: ")
	}
}
