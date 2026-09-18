package msg

import (
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// SMTP sends email through an SMTP server
type SMTP struct {
	server   string // host:port
	username string
	password string
	from     string
}

// NewSMTP creates a new SMTP sender. server is host:port (typically port
// 587 for STARTTLS submission). If username is set, PLAIN authentication
// is used. STARTTLS is negotiated automatically when the server offers it.
func NewSMTP(server, username, password, from string) *SMTP {
	return &SMTP{
		server:   server,
		username: username,
		password: password,
		from:     from,
	}
}

// headerValue makes a string safe to place in one header line: a CR or LF
// would end the header and let the rest of the value add headers of its
// own, such as another recipient.
func headerValue(v string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return ' '
		}
		return r
	}, v)
}

// Send sends an email message
func (s *SMTP) Send(to, subject, body string) error {
	if s.server == "" {
		return fmt.Errorf("SMTP server is not set")
	}

	if s.from == "" {
		return fmt.Errorf("SMTP from address is not set")
	}

	host, _, err := net.SplitHostPort(s.server)
	if err != nil {
		return fmt.Errorf("SMTP server must be host:port: %w", err)
	}

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, host)
	}

	to = headerValue(to)

	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", headerValue(s.from))
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", headerValue(subject))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	b.WriteString("\r\n")

	return smtp.SendMail(s.server, auth, s.from, []string{to}, []byte(b.String()))
}
