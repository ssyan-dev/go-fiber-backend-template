package config

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

func (c *SMTPConfig) ConnectionType() string {
	if c.Port == 465 {
		return SMTP_SSL
	}
	return SMTP_STARTTLS
}

func (c *OAuthConfig) IsEnabled() bool {
	return c.GitHub.Enabled || c.Google.Enabled || c.Yandex.Enabled;
}
