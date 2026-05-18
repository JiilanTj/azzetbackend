package smtp

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/smtp"
	"strings"
)

type Mailer struct {
	host     string
	port     string
	username string
	password string
	from     string
	addr     string
	auth     smtp.Auth
}

func (m *Mailer) HealthCheck() error {
	conn, err := net.Dial("tcp", m.addr)
	if err != nil {
		return fmt.Errorf("smtp connection error: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("smtp client error: %w", err)
	}
	defer client.Quit()

	if err := client.StartTLS(&tls.Config{ServerName: m.host}); err != nil {
		return fmt.Errorf("smtp starttls error: %w", err)
	}

	if m.auth != nil {
		if err := client.Auth(m.auth); err != nil {
			return fmt.Errorf("smtp auth error: %w", err)
		}
	}

	return nil
}

type Email struct {
	From        string
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	Body        string
	HTMLBody    string
	ReplyTo     string
	Attachments []Attachment
}

type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

func New(host, port, username, password, from string) *Mailer {
	addr := host + ":" + port

	var auth smtp.Auth
	if username != "" && password != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}

	return &Mailer{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		addr:     addr,
		auth:     auth,
	}
}

func (m *Mailer) Send(email Email) error {
	if email.From == "" {
		email.From = m.from
	}

	msg, err := m.buildMessage(email)
	if err != nil {
		return fmt.Errorf("failed to build message: %w", err)
	}

	if err := m.sendWithTLS(email.To, msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	slog.Info("email sent",
		"from", email.From,
		"to", strings.Join(email.To, ","),
		"subject", email.Subject,
	)

	return nil
}

func (m *Mailer) sendWithTLS(to []string, msg []byte) error {
	conn, err := net.Dial("tcp", m.addr)
	if err != nil {
		return fmt.Errorf("dial error: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("smtp client error: %w", err)
	}
	defer client.Quit()

	if err := client.StartTLS(&tls.Config{ServerName: m.host}); err != nil {
		return fmt.Errorf("starttls error: %w", err)
	}

	if m.auth != nil {
		if err := client.Auth(m.auth); err != nil {
			return fmt.Errorf("auth error: %w", err)
		}
	}

	if err := client.Mail(m.from); err != nil {
		return fmt.Errorf("mail from error: %w", err)
	}

	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("rcpt to error: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data error: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("close error: %w", err)
	}

	return nil
}

func (m *Mailer) buildMessage(email Email) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("From: %s\r\n", email.From))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(email.To, ",")))

	if len(email.Cc) > 0 {
		buf.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(email.Cc, ",")))
	}

	if email.ReplyTo != "" {
		buf.WriteString(fmt.Sprintf("Reply-To: %s\r\n", email.ReplyTo))
	}

	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", mime.QEncoding.Encode("UTF-8", email.Subject)))
	buf.WriteString("MIME-Version: 1.0\r\n")

	hasAttachments := len(email.Attachments) > 0
	hasBoth := email.Body != "" && email.HTMLBody != ""

	if hasAttachments {
		boundary := generateBoundary()
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
		buf.WriteString("\r\n")

		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))

		if hasBoth {
			altBoundary := generateBoundary()
			buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", altBoundary))
			buf.WriteString("\r\n")

			buf.WriteString(fmt.Sprintf("--%s\r\n", altBoundary))
			buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
			buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
			buf.WriteString("\r\n")
			buf.WriteString(email.Body)
			buf.WriteString("\r\n\r\n")

			buf.WriteString(fmt.Sprintf("--%s\r\n", altBoundary))
			buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
			buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
			buf.WriteString("\r\n")
			buf.WriteString(email.HTMLBody)
			buf.WriteString("\r\n\r\n")

			buf.WriteString(fmt.Sprintf("--%s--\r\n", altBoundary))
		} else if email.HTMLBody != "" {
			buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
			buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
			buf.WriteString("\r\n")
			buf.WriteString(email.HTMLBody)
			buf.WriteString("\r\n\r\n")
		} else {
			buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
			buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
			buf.WriteString("\r\n")
			buf.WriteString(email.Body)
			buf.WriteString("\r\n\r\n")
		}

		for _, att := range email.Attachments {
			buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			buf.WriteString(fmt.Sprintf("Content-Type: %s\r\n", att.ContentType))
			buf.WriteString("Content-Transfer-Encoding: base64\r\n")
			buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", mime.QEncoding.Encode("UTF-8", att.Filename)))
			buf.WriteString("\r\n")
			buf.WriteString(base64Encode(att.Data))
			buf.WriteString("\r\n\r\n")
		}

		buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if hasBoth {
		altBoundary := generateBoundary()
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", altBoundary))
		buf.WriteString("\r\n")

		buf.WriteString(fmt.Sprintf("--%s\r\n", altBoundary))
		buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(email.Body)
		buf.WriteString("\r\n\r\n")

		buf.WriteString(fmt.Sprintf("--%s\r\n", altBoundary))
		buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(email.HTMLBody)
		buf.WriteString("\r\n\r\n")

		buf.WriteString(fmt.Sprintf("--%s--\r\n", altBoundary))
	} else if email.HTMLBody != "" {
		buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(email.HTMLBody)
		buf.WriteString("\r\n")
	} else {
		buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(email.Body)
		buf.WriteString("\r\n")
	}

	return buf.Bytes(), nil
}

func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func generateBoundary() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("boundary_%x", b)
}
