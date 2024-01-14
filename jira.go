package main

import (
	"strings"

	jira "github.com/andygrunwald/go-jira"
	log "github.com/sirupsen/logrus"
)

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
