package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	jira "github.com/andygrunwald/go-jira"
	log "github.com/sirupsen/logrus"
	//	"github.com/pkg/profile"
)

var version string = "1.4.0"

var defaultMarker string = ">>> TIRED <<<"
var curTime time.Time

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
	var logReport bool
	var showVersion bool

	flag.StringVar(&url, "url", "", "Jira url")
	flag.StringVar(&username, "username", "", "Jira username")
	flag.StringVar(&password, "password", "", "Jira password")
	flag.StringVar(&timesheet, "timesheet", "", "Full path to timesheet file")
	flag.BoolVar(&debug, "debug", false, "Log debug messages")
	flag.BoolVar(&dry, "dry", false, "Do a 'dry' run, without real records sending")
	flag.BoolVar(&report, "report", false, "Show report of daily, weekly and monthly time recorded")
	flag.BoolVar(&logReport, "logReport", false, "Show log even during preparing a report (-record)")
	flag.BoolVar(&showVersion, "version", false, "TIme REcorDer version")

	flag.Parse()

	if showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	// Setup logging

	// Do not show log if doing report if not set explicitlly
	if !logReport && report {
		log.SetOutput(ioutil.Discard)
	} else {
		log.SetOutput(os.Stdout)
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

	curTime = time.Now()

	clog.WithFields(log.Fields{
		"date": curTime.Format(time.RFC3339),
	}).Info("Current date and time.")

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
		workingHours := calendar(int(curTime.Year()), int(curTime.Month()))

		daily, weekly, monthly := createReport(timesheetCont)
		fmt.Printf(
			"%.1f, %.1f, %.1f (%d)\n",
			float32(daily)/3600,
			float32(weekly)/3600,
			float32(monthly)/3600,
			workingHours,
		)
	} else {
		sendToJira(timesheetCont)
	}

	clog.Info("Jira Time Reporter job is finished.")
	os.Exit(0)
}

func createReport(timesheetCont []string) (daily int, weekly int, monthly int) {
	clog.Info("Creating report.")
	monthBefore := curTime.AddDate(0, -1, 0)

	prefix := fmt.Sprintf("%d-%02d-", monthBefore.Year(), monthBefore.Month())
	clog.WithFields(log.Fields{
		"prefix": prefix,
	}).Debug("Prefix to filter records by.")

	monthlyRecords := getRecordsUntil(timesheetCont, prefix)
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
		if record.ParsedStartTime.Year() == curTime.Year() &&
			record.ParsedStartTime.Month() == curTime.Month() &&
			record.ParsedStartTime.Day() == curTime.Day() {
			daily += record.Duration

			clog.WithFields(log.Fields{
				"record_number": record.LineNumber,
				"record_year":   record.ParsedStartTime.Year(),
				"record_month":  record.ParsedStartTime.Month(),
				"record_day":    record.ParsedStartTime.Day(),
			}).Debug("Adding to daily.")
		}

		rWeekY, rWeekN := record.ParsedStartTime.ISOWeek()
		cWeekY, cWeekN := curTime.ISOWeek()
		if rWeekY == cWeekY && rWeekN == cWeekN {
			weekly += record.Duration

			clog.WithFields(log.Fields{
				"record_number":      record.LineNumber,
				"record_week_year":   rWeekY,
				"record_week_number": rWeekN,
			}).Debug("Adding to weekly.")
		}

		if record.ParsedStartTime.Month() == curTime.Month() {
			monthly += record.Duration

			clog.WithFields(log.Fields{
				"record_number": record.LineNumber,
				"record_month":  record.ParsedStartTime.Month(),
			}).Debug("Adding to monthly.")
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
