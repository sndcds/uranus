package app

import (
	"bytes"
	"html/template"
	"net/mail"
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
