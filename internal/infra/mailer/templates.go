package mailer

import "embed"

//go:embed templates/*.html
var DefaultTemplatesFS embed.FS
