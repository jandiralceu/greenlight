package mailer

import (
	"bytes"
	"embed"
	"github.com/go-mail/mail/v2"
	"html/template"
	"time"
)

//go:embed templates/*
var templateFS embed.FS

type Mailer struct {
	dailer *mail.Dialer
	sender string
}

func New(host string, port int, username, password, sender string) Mailer {
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 10 * time.Second

	return Mailer{
		dailer: dialer,
		sender: sender,
	}
}

func (m Mailer) Send(recipient, templateFile string, data any) error {
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	subject := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(subject, "subject", data); err != nil {
		return err
	}

	plainBody := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(plainBody, "plainBody", data); err != nil {
		return err
	}

	htmlBody := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(htmlBody, "htmlBody", data); err != nil {
		return err
	}

	msg := mail.NewMessage()
	msg.SetHeader("From", m.sender)
	msg.SetHeader("To", recipient)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String())

	if err := m.dailer.DialAndSend(msg); err != nil {
		return err
	}

	return nil
}
