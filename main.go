package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	jira "github.com/andygrunwald/go-jira"
	log "github.com/sirupsen/logrus"
	//	"github.com/pkg/profile"
)

var version string = "1.1.0"

var clog *log.Entry

func main() {
	//	defer profile.Start().Stop()

	var url string
	var username string
	var password string
	var timesheet string
	var debug bool
	var dry bool
	var showVersion bool

	flag.StringVar(&url, "url", "", "Jira url")
	flag.StringVar(&username, "username", "", "Jira username")
	flag.StringVar(&password, "password", "", "Jira password")
	flag.StringVar(&timesheet, "timesheet", "", "Full path to timesheet file")
	flag.BoolVar(&debug, "debug", false, "Log debug messages")
	flag.BoolVar(&dry, "dry", false, "Do a 'dry' run, without real records sending")
	flag.BoolVar(&showVersion, "version", false, "TIme REcorDer version")

	flag.Parse()

	// Setup logging
	log.SetOutput(os.Stdout)

	if showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	clog = log.WithFields(log.Fields{
		"pid":     os.Getpid(),
		"thread":  "main",
		"version": version,
	})

	clog.Info("Starting Jira Time Reporter.")

	// Validate variables
	if timesheet == "" {
		clog.Fatal("'-timesheet' option is mandatory.")
	}

	if dry {
		clog.Warning("Running in 'dry' mode.")
	}

	if url == "" {
		url = promptForSecret("url")
	}

	if username == "" {
		username = promptForSecret("user")
	}

	if password == "" {
		password = promptForSecret("password")
	}

	curTime := time.Now().Format(time.RFC3339)
	clog.WithFields(log.Fields{"date": curTime}).Info("Current date and time.")

	tp := jira.BasicAuthTransport{
		Username: username,
		Password: password,
	}

	jiraClient, err := jira.NewClient(tp.Client(), url)
	if err != nil {
		clog.WithFields(log.Fields{"error": err}).Fatal("Jira connection failed.")
	}
	clog.Info("Connection to Jira established.")

	// Read timesheet file
	tsFileStat, _ := os.Stat(timesheet)

	timesheetCont := readTimesheet(timesheet)
	clog.WithFields(log.Fields{
		"number": len(timesheetCont),
	}).Info("Timesheet file total lines number.")

	actualWorkRecords := getActualWorkRecords(timesheetCont)
	clog.WithFields(log.Fields{
		"number": len(actualWorkRecords),
	}).Info("Actual work records number.")

	if len(actualWorkRecords) > 0 {
		workRecords, parseErrCount := parseWorkRecords(actualWorkRecords)
		if parseErrCount > 0 {
			clog.WithFields(log.Fields{
				"valid":   len(workRecords),
				"invalid": parseErrCount,
			}).Fatal("Timeshit parsing finished with errors.")
		}

		errCount := sendWorkRecords(jiraClient, workRecords, dry)

		if errCount > 0 {
			clog.WithFields(log.Fields{
				"number": errCount,
			}).Fatal("Jira Time Reporter finished with errors.")
		}

		if dry == false {
			writeTimesheet(timesheet, timesheetCont)

			// Compare timesheet file size before and after changes
			tsFileStatNew, _ := os.Stat(timesheet)
			if tsFileStatNew.Size() != tsFileStat.Size() {
				clog.WithFields(log.Fields{
					"before": tsFileStat.Size(),
					"after":  tsFileStatNew.Size(),
				}).Fatal("Timesheet file size changed.")
			}
		}
	} else {
		clog.Warning("No actual work records was found.")
	}

	clog.Info("Jira Time Reporter job is finished.")
	os.Exit(0)
}
