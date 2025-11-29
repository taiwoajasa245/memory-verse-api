package mail

import (
	"bytes"
	"fmt"
	"net/smtp"
	"text/template"
)

type Mailer struct {
	FromName string
	From     string
	Password string
	Host     string
	Port     string
	auth     smtp.Auth
}

func NewMail(from, fromName, password, host, port string) *Mailer {
	auth := smtp.PlainAuth("", from, password, host)
	return &Mailer{
		FromName: fromName,
		From:     from,
		Password: password,
		Host:     host,
		Port:     port,
		auth:     auth,
	}
}

func (m *Mailer) SendHTML(to, subject, templateName string, data interface{}) error {
	// Parse your HTML template
	tmpl, err := template.ParseFiles(fmt.Sprintf("internal/mail/templates/%s", templateName))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var body bytes.Buffer
	body.WriteString("MIME-Version: 1.0\r\n")
	body.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	body.WriteString(fmt.Sprintf("From: %s <%s>\r\n", m.FromName, m.From))
	body.WriteString(fmt.Sprintf("To: %s\r\n", to))
	body.WriteString(fmt.Sprintf("Subject: %s\r\n\r\n", subject))

	// Render the HTML body
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	addr := fmt.Sprintf("%s:%s", m.Host, m.Port)
	if err := smtp.SendMail(addr, m.auth, m.From, []string{to}, body.Bytes()); err != nil {
		return fmt.Errorf("failed to send mail: %w", err)
	}

	return nil

}
