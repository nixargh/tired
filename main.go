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

var version string = "1.2.0"

var defaultMarker string = ">>> TIRED <<<"

var clog *log.Entry

var timesheet string
var url string
var username string
var password string
var dry bool

func main() {
	//	defer profile.Start().Stop()

	var debug bool
	var report bool
	var showVersion bool

	flag.StringVar(&url, "url", "", "Jira url")
	flag.StringVar(&username, "username", "", "Jira username")
	flag.StringVar(&password, "password", "", "Jira password")
	flag.StringVar(&timesheet, "timesheet", "", "Full path to timesheet file")
	flag.BoolVar(&debug, "debug", false, "Log debug messages")
	flag.BoolVar(&dry, "dry", false, "Do a 'dry' run, without real records sending")
	flag.BoolVar(&report, "report", false, "Show report of daily, weekly and monthly time recorded")
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

	curTime := time.Now().Format(time.RFC3339)
	clog.WithFields(log.Fields{"date": curTime}).Info("Current date and time.")

	// Validate variables
	if timesheet == "" {
		clog.Fatal("'-timesheet' option is mandatory.")
	}

	if dry {
		clog.Warning("Running in 'dry' mode.")
	}

	// Read timesheet file
	timesheetCont := readTimesheet(timesheet)
	clog.WithFields(log.Fields{
		"number": len(timesheetCont),
	}).Info("Timesheet file total lines number.")

	if report {
		daily, weekly, monthly := createReport(timesheetCont)
		fmt.Printf(
			"%.1f, %.1f, %.1f\n",
			float32(daily)/3600,
			float32(weekly)/3600,
			float32(monthly)/3600,
		)
	} else {
		sendToJira(timesheetCont)
	}

	clog.Info("Jira Time Reporter job is finished.")
	os.Exit(0)
}

func createReport(timesheetCont []string) (daily int, weekly int, monthly int) {
	curTime := time.Now()
	monthBefore := curTime.AddDate(0, -1, 0)

	lastMonth := fmt.Sprintf("%d-%d-", monthBefore.Year(), monthBefore.Month())
	clog.WithFields(log.Fields{
		"marker": lastMonth,
	}).Debug("Last month line prefix.")

	monthlyRecords := getRecordsUntil(timesheetCont, lastMonth)
	clog.WithFields(log.Fields{
		"number": len(monthlyRecords),
	}).Info("Monthly records number for a report.")

	records, parseErrCount := parseWorkRecords(monthlyRecords)
	if parseErrCount > 0 {
		clog.WithFields(log.Fields{
			"valid":   len(records),
			"invalid": parseErrCount,
		}).Fatal("Timeshit parsing finished with errors.")
	}

	for _, record := range records {
		if record.ParsedStartTime.Day() == curTime.Day() {
			daily += record.Duration
		}

		rWeekY, rWeekN := record.ParsedStartTime.ISOWeek()
		cWeekY, cWeekN := curTime.ISOWeek()
		if rWeekY == cWeekY && rWeekN == cWeekN {
			weekly += record.Duration
		}

		if record.ParsedStartTime.Month() == curTime.Month() {
			monthly += record.Duration
		}
	}

	return
}

func sendToJira(timesheetCont []string) {
	if url == "" {
		url = promptForSecret("url")
	}

	if username == "" {
		username = promptForSecret("user")
	}

	if password == "" {
		password = promptForSecret("password")
	}

	tp := jira.BasicAuthTransport{
		Username: username,
		Password: password,
	}

	jiraClient, err := jira.NewClient(tp.Client(), url)
	if err != nil {
		clog.WithFields(log.Fields{"error": err}).Fatal("Jira connection failed.")
	}
	clog.Info("Connection to Jira established.")

	// Save timesheet file size for later comparison
	tsFileStat, _ := os.Stat(timesheet)

	uncommittedRecords := getRecordsUntil(timesheetCont, defaultMarker)
	clog.WithFields(log.Fields{
		"number": len(uncommittedRecords),
	}).Info("Uncommitted records number.")

	if len(uncommittedRecords) > 0 {
		workRecords, parseErrCount := parseWorkRecords(uncommittedRecords)
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
}
