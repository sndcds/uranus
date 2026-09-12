package app

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"golang.org/x/net/idna"
)

func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func RenderEmailTemplate(
	layoutPath string,
	contentPath string,
	data any,
) (string, string, error) {

	tmpl, err := template.ParseFiles(
		"template/email/email_base.html",
		layoutPath,
		contentPath,
	)
	if err != nil {
		return "", "", err
	}

	var subject bytes.Buffer

	if err := tmpl.ExecuteTemplate(
		&subject,
		"subject",
		data,
	); err != nil {
		return "", "", err
	}

	var body bytes.Buffer

	if err := tmpl.ExecuteTemplate(
		&body,
		"email_base",
		data,
	); err != nil {
		return "", "", err
	}

	return subject.String(), body.String(), nil
}

func SendEmailWithTimeout(
	to, subject, htmlContent string,
	timeout time.Duration,
) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- SendEmail(to, subject, htmlContent)
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case err := <-errCh:
		return err

	case <-timer.C:
		return fmt.Errorf("send email timeout after %s", timeout)
	}
}

func SendEmail(to, subject string, htmlContent string) error {
	from := UranusInstance.Config.AuthReplyEmail
	userName := UranusInstance.Config.AuthSmtpLogin
	password := UranusInstance.Config.AuthSmtpPassword
	smtpHost := UranusInstance.Config.AuthSmtpHost
	smtpPort := UranusInstance.Config.AuthSmtpPort // int

	asciiFrom, err := EncodeEmailAddress(from)
	if err != nil {
		return fmt.Errorf("unable to send email 1: %s", err.Error())
	}

	asciiTo, err := EncodeEmailAddress(to)
	if err != nil {
		return fmt.Errorf("unable to send email 2: %s", err.Error())
	}

	// Encode subject in Base64 for UTF-8
	encodedSubject := fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))

	message := []byte(
		"Subject: " + encodedSubject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"To: " + asciiTo + "\r\n" +
			"From: " + asciiFrom + "\r\n" +
			"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
			"Content-Transfer-Encoding: 8bit\r\n" +
			"\r\n" +
			htmlContent + "\r\n")

	auth := smtp.PlainAuth("", userName, password, smtpHost)
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	err = smtp.SendMail(addr, auth, userName, []string{asciiTo}, message)
	if err != nil {
		return fmt.Errorf("unable to send email: %w", err)
	}

	return nil
}

// Encode an email address for SMTP
func EncodeEmailAddress(email string) (string, error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid email: %s", email)
	}

	local := parts[0]  // user
	domain := parts[1] // domain

	asciiDomain, err := idna.ToASCII(domain)
	if err != nil {
		return "", err
	}

	return local + "@" + asciiDomain, nil
}
