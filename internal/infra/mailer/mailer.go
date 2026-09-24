package mailer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io/fs"

	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
	"go.uber.org/zap"
)

var (
	ErrMailerDisabled = errors.New("mailer is disabled")
	ErrFailedToSend   = errors.New("failed to send email")
)

type MailerService interface {
	SendTemplate(ctx context.Context, to, subject, name string, data any) error
	Ping(ctx context.Context) error
}

type smtpMailer struct {
	cfg       *config.SMTPConfig
	l         *zap.Logger
	templates *template.Template
}

func NewMailerService(cfg *config.SMTPConfig, l *zap.Logger, templatesFS fs.FS) (MailerService, error) {
	if cfg.Enabled {
		switch {
		case cfg.Host == "":
			return nil, errors.New("mailer: SMTP_HOST is required when SMTP is enabled")
		case cfg.Email == "":
			return nil, errors.New("mailer: SMTP_EMAIL is required when SMTP is enabled")
		case cfg.Password == "":
			return nil, errors.New("mailer: SMTP_PASSWORD is required when SMTP is enabled")
		}
	}

	t := template.New("").Funcs(template.FuncMap{
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
	})

	if templatesFS != nil {
		var err error
		t, err = t.ParseFS(templatesFS, "templates/*.html")
		if err != nil {
			return nil, fmt.Errorf("mailer: parse templates: %w", err)
		}
	}

	return &smtpMailer{cfg: cfg, l: l, templates: t}, nil
}

func (m *smtpMailer) SendTemplate(ctx context.Context, to, subject, name string, data any) error {
	if !m.cfg.Enabled {
		return ErrMailerDisabled
	}

	var buf bytes.Buffer

	if err := m.templates.ExecuteTemplate(&buf, name, data); err != nil {
		return fmt.Errorf("mailer: template %s: %w", name, err)
	}

	return m.send(ctx, to, m.buildMessage(to, subject, buf.String(), ""))
}

func (m *smtpMailer) Ping(ctx context.Context) error {
	if !m.cfg.Enabled {
		return ErrMailerDisabled
	}

	client, err := m.connect(ctx)
	if err != nil {
		return fmt.Errorf("mailer: ping: %w", err)
	}
	client.Close()

	return nil
}
