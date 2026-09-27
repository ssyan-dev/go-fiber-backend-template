package config

import "fmt"

var (
	IsProduction = true
)

var (
	SMTP_SSL      = "ssl"
	SMTP_STARTTLS = "starttls"
)

func (c *AppConfig) IsDevelopment() bool {
	return c.Env == "development"
}

func (c *AppConfig) IsProduction() bool {
	return c.Env == "production"
}

func (c *AppConfig) AppURL() string {
	if c.IsProduction() {
		return "https://" + c.URL
	}
	return fmt.Sprintf("http://%s:%d", c.URL, c.Port)
}

func (c *AppConfig) BaseURL() string {
	return c.AppURL() + c.GlobalPrefix
}

func (c *SMTPConfig) ConnectionType() string {
	if c.Port == 465 {
		return SMTP_SSL
	}
	return SMTP_STARTTLS
}

func (c *OAuthConfig) IsEnabled() bool {
	return c.GitHub.Enabled || c.Google.Enabled || c.Yandex.Enabled
}
