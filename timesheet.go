package main

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"slices"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var timeNow time.Time

type ActualWorkRecords struct {
	LineNumber int
	Record     string
}

type WorkRecord struct {
	LineNumber      int
	Date            string
	StartTime       string
	EndTime         string
	Issue           string
	Comment         string
	ParsedStartTime time.Time
	ParsedEndTime   time.Time
	Duration        int
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

func getActualWorkRecords(timesheetCont []string) []ActualWorkRecords {
	clog.Info("Looking for actual work records.")
	var actualWorkRecords []ActualWorkRecords

	for i := len(timesheetCont) - 1; i >= 0; i-- {
		wr := strings.TrimSpace(timesheetCont[i])
		if len(wr) == 0 || strings.HasPrefix(wr, "#") {
			continue
		}

		if wr == ">>> TIRED <<<" {
			break
		}

		awr := ActualWorkRecords{i, wr}

		clog.WithFields(log.Fields{"work record": awr}).Debug("New work record.")
		actualWorkRecords = append(actualWorkRecords, awr)
	}

	slices.Reverse(actualWorkRecords)
	return actualWorkRecords
}

func parseWorkRecords(records []ActualWorkRecords) ([]WorkRecord, int) {
	var workRecords []WorkRecord
	var errCount int

	timeNow = time.Now()

	clog.Info("Parsing actual work records.")

	// Location for time
	location := getTimeLocation()

	var wrBefore WorkRecord

	for _, record := range records {
		// Read original fileds
		fields := strings.SplitN(record.Record, ",", 5)

		var wr WorkRecord
		wr.LineNumber = record.LineNumber
		wr.Date = fields[0]
		wr.StartTime = fields[1]
		wr.EndTime = fields[2]
		wr.Issue = fields[3]
		wr.Comment = strings.ReplaceAll(fields[4], "\"", "")

		// "00:00" is a begining of a day at go time parser.
		if wr.EndTime == "00:00" {
			wr.EndTime = "23:59"
		}

		// Do not validate records with an empty EndTime as they haven't been compleated yet.
		if wr.EndTime == "" {
			clog.WithFields(log.Fields{
				"line": wr.LineNumber,
			}).Warning("End time field is empty, skipping.")

			continue
		}

		if !validateRawRecord(&wr) {
			errCount += 1
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
			clog.WithFields(log.Fields{
				"error": eerr,
			}).Fatal("Failed to parse end time.")
		}

		wr.ParsedEndTime = endDateTime

		wr.Duration = int(endDateTime.Sub(startDateTime).Seconds())

		if !validateRecord(&wr, &wrBefore) {
			errCount += 1
			continue
		}

		clog.WithFields(log.Fields{"record": wr}).Debug("Record fields.")

		workRecords = append(workRecords, wr)
		wrBefore = wr
	}

	return workRecords, errCount
}

func validateRawRecord(wr *WorkRecord) bool {
	valid := true

	if wr.Date == "" || wr.StartTime == "" || wr.EndTime == "" || wr.Issue == "" || wr.Comment == "" {
		clog.WithFields(log.Fields{
			"line": wr.LineNumber,
		}).Error("Some fields are empty, only EndTime allowed.")

		valid = false
	}

	matched, ierr := regexp.MatchString(`^[A-Z-]+-\d+$`, wr.Issue)
	if ierr != nil || !matched {
		clog.WithFields(log.Fields{
			"line":  wr.LineNumber,
			"issue": wr.Issue,
		}).Error("Bad Jira issue number.")

		valid = false
	}

	return valid
}

func validateRecord(wr *WorkRecord, wrBefore *WorkRecord) bool {
	valid := true

	if wr.Duration <= 0 {
		clog.WithFields(log.Fields{
			"line":     wr.LineNumber,
			"duration": wr.Duration,
		}).Error("Start time is after End time.")

		valid = false
	}

	if wr.ParsedStartTime.Year() != timeNow.Year() {
		clog.WithFields(log.Fields{
			"line":         wr.LineNumber,
			"year_record":  wr.ParsedStartTime.Year(),
			"year_current": timeNow.Year(),
		}).Error("Record has date from another year.")

		valid = false
	}

	if wr.ParsedStartTime.Before(wrBefore.ParsedEndTime) {
		clog.WithFields(log.Fields{
			"line":       wr.LineNumber,
			"time_start": wr.ParsedStartTime,
			"time_end":   wrBefore.ParsedEndTime,
		}).Error("Start time < than previous end time.")

		valid = false
	}

	return valid
}
