package main

import (
	"bytes"
	"fmt"
	"net/smtp"
	"time"
)

func GenerateIcal(event Event, delete bool) string {
	var buf bytes.Buffer

	startTime, err := time.Parse(time.RFC3339, event.Date)
	if err != nil {
		fmt.Println("Error parsing event date:", err)
		return ""
	}

	endTime := startTime.Add(time.Duration(event.Duration) * time.Minute)

	fmt.Fprintf(&buf, "BEGIN:VCALENDAR\n")
	fmt.Fprintf(&buf, "VERSION:2.0\n")
	fmt.Fprintf(&buf, "CALSCALE:GREGORIAN\n")
	fmt.Fprintf(&buf, "BEGIN:VEVENT\n")
	fmt.Fprintf(&buf, "UID:%s@prayujt.com\n", event.Id)
	if delete {
		fmt.Fprintf(&buf, "SUMMARY:CANCELLED: %s\n", event.Title)
	} else {
		fmt.Fprintf(&buf, "SUMMARY:%s\n", event.Title)
	}
	if event.Description != nil {
		fmt.Fprintf(&buf, "DESCRIPTION:%s\n", *event.Description)
	}
	fmt.Fprintf(&buf, "DTSTART:%s\n", startTime.Format("20060102T150405Z"))
	fmt.Fprintf(&buf, "DTEND:%s\n", endTime.Format("20060102T150405Z"))

	if event.RecurrenceId != "" {
		fmt.Fprintf(&buf, "RRULE:FREQ=WEEKLY;INTERVAL=1\n")
	}

	if delete {
		fmt.Fprintf(&buf, "STATUS:CANCELLED\n")
	} else {
		fmt.Fprintf(&buf, "STATUS:CONFIRMED\n")
	}
	fmt.Fprintf(&buf, "END:VEVENT\n")
	fmt.Fprintf(&buf, "END:VCALENDAR\n")
	return buf.String()
}

func SendEvent(to []string, body string, event Event, delete bool) {
	fromEmail := "calendar@prayujt.com"
	fromName := "Prayuj Calendar"

	smtpHost := "mail.prayujt.com"
	smtpPort := "587"

	subject := fmt.Sprintf("Subject: %s\n", event.Title)
	fromHeader := fmt.Sprintf("From: %s <%s>\n", fromName, fromEmail)
	toHeader := fmt.Sprintf("To: %s\n", to[0])

	dateHeader := fmt.Sprintf("Date: %s\n", time.Now().Format(time.RFC1123Z))

	icalContent := GenerateIcal(event, delete)

	// Construct the full message with headers and body
	message := []byte(fromHeader + toHeader + subject + dateHeader + "MIME-Version: 1.0\n" +
		"Content-Type: multipart/mixed; boundary=boundary\n\n" +
		"--boundary\n" +
		"Content-Type: text/plain; charset=\"utf-8\"\n" +
		"Content-Transfer-Encoding: 7bit\n\n" +
		body + "\n\n" +
		"--boundary\n" +
		"Content-Type: text/calendar; charset=\"utf-8\"\n" +
		"Content-Disposition: attachment; filename=\"event.ics\"\n" +
		"Content-Transfer-Encoding: 7bit\n\n" +
		icalContent + "\n" +
		"--boundary--")

	auth := smtp.PlainAuth("", fromEmail, mailPassword, smtpHost)

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, fromEmail, to, message)
	if err != nil {
		fmt.Println("Error sending email:", err)
		return
	}
}
