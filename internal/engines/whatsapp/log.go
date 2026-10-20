package whatsapp

import (
	"fmt"
	"log/slog"
	"regexp"

	waLog "go.mau.fi/whatsmeow/util/log"
)

// longNumber matches phone numbers and the numeric part of JIDs, which must never reach a log.
var longNumber = regexp.MustCompile(`\d{6,}`)

// redact removes phone numbers and other long numeric identifiers from a log line.
func redact(s string) string {
	return longNumber.ReplaceAllString(s, "[redacted]")
}

// slogAdapter routes whatsmeow's logs to the session's slog.Logger. Only warnings and
// errors are kept, and every line is redacted.
type slogAdapter struct {
	log *slog.Logger
}

var _ waLog.Logger = slogAdapter{}

func newLogger(l *slog.Logger) waLog.Logger { return slogAdapter{log: l} }

func (a slogAdapter) Errorf(msg string, args ...any) { a.log.Error(redact(fmt.Sprintf(msg, args...))) }
func (a slogAdapter) Warnf(msg string, args ...any)  { a.log.Warn(redact(fmt.Sprintf(msg, args...))) }
func (a slogAdapter) Infof(string, ...any)           {}
func (a slogAdapter) Debugf(string, ...any)          {}
func (a slogAdapter) Sub(module string) waLog.Logger {
	return slogAdapter{log: a.log.With("module", module)}
}
