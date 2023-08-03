package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/exp/slices"
	"golang.org/x/term"

	"github.com/zalando/go-keyring"

	jira "github.com/andygrunwald/go-jira"
	log "github.com/sirupsen/logrus"
	//	"github.com/pkg/profile"
)

var version string = "1.0.2"

var clog *log.Entry

var marker string = ">>> TIRED <<<"

type WorkRecord struct {
	Date            string
	StartTime       string
	EndTime         string
	Issue           string
	Comment         string
	ParsedStartTime time.Time
	Duration        int
}

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
	flag.BoolVar(&showVersion, "version", false, "FunVPN version")

	flag.Parse()

	// Setup logging
	log.SetOutput(os.Stdout)

	if showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if debug == true {
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
	clog.WithFields(log.Fields{"number": len(timesheetCont)}).Info("Timesheet file total lines number.")

	actualWorkRecords := getActualWorkRecords(timesheetCont)
	clog.WithFields(log.Fields{"number": len(actualWorkRecords)}).Info("Actual work records number.")

	if len(actualWorkRecords) > 0 {
		workRecords := parseWorkRecords(actualWorkRecords)

		errCount := sendWorkRecords(jiraClient, workRecords, dry)

		if errCount > 0 {
			clog.WithFields(log.Fields{"number": errCount}).Fatal("Jira Time Reporter finished with errors.")
		}

		if dry == false {
			writeTimesheet(timesheet, timesheetCont)

			// Compare timesheet file size before and after changes
			tsFileStatNew, _ := os.Stat(timesheet)
			if tsFileStatNew.Size() != tsFileStat.Size() {
				clog.WithFields(log.Fields{"before": tsFileStat.Size(), "after": tsFileStatNew.Size()}).Fatal("Timesheet file size changed.")
			}
		}
	} else {
		clog.Warning("No actual work records was found.")
	}

	clog.Info("Jira Time Reporter job is finished.")
	os.Exit(0)
}

func promptForSecret(secret string) string {
	service := "tired"
	var secretValue string
	var err error

	secretValue, err = keyring.Get(service, secret)

	if err == nil && secretValue != "" {
		clog.WithFields(log.Fields{"secret": secret}).Info("Got secret value from keyring.")
		return secretValue
	}

	fmt.Printf("New '%v' value: ", secret)
	bytespw, _ := term.ReadPassword(int(syscall.Stdin))
	secretValue = string(bytespw)
	fmt.Print("\n")

	err = keyring.Set(service, secret, secretValue)

	if err != nil {
		clog.WithFields(log.Fields{"secret": secret, "error": err}).Fatal("Can't save password to keyring.")
	}

	clog.WithFields(log.Fields{"secret": secret}).Info("Secret saved to keyring.")
	return secretValue
}

func readTimesheet(path string) []string {
	clog.WithFields(log.Fields{"path": path}).Info("Reading timesheet.")

	absPath, _ := filepath.Abs(path)
	content, err := ioutil.ReadFile(absPath)

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
	os.Rename(absPath, bkpPath)

	// create the file
	f, err := os.Create(absPath)
	if err != nil {
		clog.WithFields(log.Fields{"error": err}).Fatal("Failed to create the file.")
	}

	defer f.Close()

	// write a string
	for i, line := range content {
		// Remove marker
		if line == marker {
			continue
		}

		_, lErr := f.WriteString(line + "\n")

		if lErr != nil {
			clog.WithFields(log.Fields{"number": i, "error": lErr}).Fatal("Failed to write the line.")
		}
	}

	// Re-add marker to the end of file
	_, lErr := f.WriteString(marker)
	if lErr != nil {
		clog.WithFields(log.Fields{"error": lErr}).Fatal("Failed to write the marker.")
	}
}

func getActualWorkRecords(timesheetCont []string) []string {
	clog.Info("Looking for actual work records.")
	var actualWorkRecords []string

	for i := len(timesheetCont) - 1; i >= 0; i-- {
		wr := strings.TrimSpace(timesheetCont[i])
		if len(wr) == 0 || strings.HasPrefix(wr, "#") {
			continue
		}

		if wr == ">>> TIRED <<<" {
			break
		}

		clog.WithFields(log.Fields{"work record": wr}).Debug("New work record.")
		actualWorkRecords = append(actualWorkRecords, wr)
	}

	slices.Reverse(actualWorkRecords)
	return actualWorkRecords
}

func parseWorkRecords(records []string) []WorkRecord {
	var workRecords []WorkRecord
	clog.Info("Parsing actual work records.")

	// Location for time
	location := getTimeLocation()

	for _, record := range records {
		// Read original fileds
		fields := strings.SplitN(record, ",", 5)

		var wr WorkRecord
		wr.Date = fields[0]
		wr.StartTime = fields[1]
		wr.EndTime = fields[2]
		wr.Issue = fields[3]
		wr.Comment = strings.ReplaceAll(fields[4], "\"", "")

		// Validations
		if wr.Date == "" || wr.StartTime == "" || wr.EndTime == "" || wr.Issue == "" || wr.Comment == "" {
			clog.WithFields(log.Fields{"record": wr}).Warning("Some fields are empty, skipping the record.")
			continue
		}

		// Add duration
		startDateTimeString := fmt.Sprintf("%s %s:00", wr.Date, wr.StartTime)
		startDateTime, serr := time.ParseInLocation(time.DateTime, startDateTimeString, location)
		if serr != nil {
			clog.WithFields(log.Fields{"error": serr}).Fatal("Failed to parse start time.")
		}

		// Required for record
		wr.ParsedStartTime = startDateTime

		endDateTimeString := fmt.Sprintf("%s %s:00", wr.Date, wr.EndTime)
		endDateTime, eerr := time.ParseInLocation(time.DateTime, endDateTimeString, location)
		if eerr != nil {
			clog.WithFields(log.Fields{"error": eerr}).Fatal("Failed to parse end time.")
		}

		wr.Duration = int(endDateTime.Sub(startDateTime).Seconds())

		clog.WithFields(log.Fields{"record": wr}).Debug("Record fields.")

		workRecords = append(workRecords, wr)
	}

	return workRecords
}

func getTimeLocation() *time.Location {
	timezoneRaw, err := ioutil.ReadFile("/etc/timezone")
	if err != nil {
		clog.WithFields(log.Fields{"error": err}).Fatal("Failed to read '/etc/timezone'.")
	}
	timezone := strings.Trim(string(timezoneRaw), "\n")
	clog.WithFields(log.Fields{"timezone": timezone}).Debug("Detected timezone.")

	location, err := time.LoadLocation(timezone)
	if err != nil {
		clog.WithFields(log.Fields{"error": err}).Fatal("Can't get time zone.")
	}

	return location
}

func sendWorkRecords(jiraClient *jira.Client, records []WorkRecord, dry bool) int {
	errCount := 0

	clog.Info("Sending work records to Jira.")

	for _, record := range records {
		// Check the issue exists
		issue, _, err := jiraClient.Issue.Get(record.Issue, nil)

		if err != nil {
			errShort := strings.Split(err.Error(), ":")[0]
			clog.WithFields(log.Fields{"issue": record.Issue, "error": errShort}).Error("Can't get access to the issue.")
			errCount++
			continue
		}
		clog.WithFields(log.Fields{"issue": record.Issue, "worklog_total": issue.Fields.Worklog.Total, "summary": issue.Fields.Summary}).Debug("Issue found.")

		// Send work log
		jiraStartTime := jira.Time(record.ParsedStartTime)

		workRec := jira.WorklogRecord{
			Comment:          record.Comment,
			Started:          &jiraStartTime,
			TimeSpentSeconds: record.Duration,
		}

		if dry == false {
			_, _, aErr := jiraClient.Issue.AddWorklogRecord(record.Issue, &workRec)
			if err != nil {
				clog.WithFields(log.Fields{"issue": record.Issue, "error": aErr}).Error("Failed to add Work Record to the issue.")
				errCount++
				continue
			}
		}
		clog.WithFields(log.Fields{"dry": dry, "issue": record.Issue, "worklog_total": issue.Fields.Worklog.Total, "summary": issue.Fields.Summary}).Info("Work Record added.")
	}

	return errCount
}
