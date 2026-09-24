package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
)

func (m *smtpMailer) connect(ctx context.Context) (*smtp.Client, error) {
	addr := net.JoinHostPort(m.cfg.Host, fmt.Sprintf("%d", m.cfg.Port))
	tlsCfg := &tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12}

	if m.cfg.ConnectionType() == config.SMTP_SSL {
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("smtp dial: %w", err)
		}
		tlsConn := tls.Client(conn, tlsCfg)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			conn.Close()
			return nil, fmt.Errorf("smtp tls handshake: %w", err)
		}
		return smtp.NewClient(tlsConn, m.cfg.Host)
	}

	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("smtp dial: %w", err)
	}
	if err := client.StartTLS(tlsCfg); err != nil {
		client.Close()
		return nil, fmt.Errorf("smtp starttls: %w", err)
	}
	return client, nil
}

func (m *smtpMailer) send(ctx context.Context, to, msg string) error {
	client, err := m.connect(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToSend, err)
	}
	defer client.Close()

	if err := client.Auth(smtp.PlainAuth("", m.cfg.Email, m.cfg.Password, m.cfg.Host)); err != nil {
		return fmt.Errorf("%w: auth: %w", ErrFailedToSend, err)
	}
	if err := client.Mail(m.cfg.Email); err != nil {
		return fmt.Errorf("%w: mail from: %w", ErrFailedToSend, err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("%w: rcpt: %w", ErrFailedToSend, err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("%w: data: %w", ErrFailedToSend, err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		w.Close()
		return fmt.Errorf("%w: write: %w", ErrFailedToSend, err)
	}
	return w.Close()
}

func (m *smtpMailer) buildHeaders(to, subject, contentType, boundary string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s <%s>\r\n", m.cfg.FromName, m.cfg.Email)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	fmt.Fprintf(&b, "MIME-Version: 1.0\r\n")

	if boundary != "" {
		fmt.Fprintf(&b, "Content-Type: %s; boundary=\"%s\"\r\n", contentType, boundary)
	} else {
		fmt.Fprintf(&b, "Content-Type: %s; charset=\"UTF-8\"\r\n", contentType)
	}
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	return b.String()
}

func (m *smtpMailer) buildMessage(to, subject, htmlBody, plainBody string) string {
	if htmlBody == "" {
		var b strings.Builder
		b.WriteString(m.buildHeaders(to, subject, "text/plain", ""))
		b.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
		b.WriteString(plainBody)
		return b.String()
	}

	boundary := fmt.Sprintf("boundary_%d", time.Now().UnixNano())
	var b strings.Builder
	b.WriteString(m.buildHeaders(to, subject, "multipart/alternative", boundary))
	b.WriteString("\r\n")
	b.WriteString("This is a multipart message in MIME format.\r\n")

	for _, part := range []struct{ ct, body string }{
		{"text/plain", "This email requires HTML support.\r\n"},
		{"text/html", htmlBody},
	} {
		fmt.Fprintf(&b, "--%s\r\n", boundary)
		fmt.Fprintf(&b, "Content-Type: %s; charset=\"UTF-8\"\r\n", part.ct)
		b.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
		b.WriteString(part.body)
		b.WriteString("\r\n")
	}
	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return b.String()
}
