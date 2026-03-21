package qq

import (
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// botGoLogger preserves useful SDK info logs while demoting noisy heartbeat
// traffic to DEBUG so long-running QQ sessions do not spam the console.
type botGoLogger struct {
	*logger.Logger
}

func newBotGoLogger(component string) *botGoLogger {
	return &botGoLogger{Logger: logger.NewLogger(component)}
}

func (b *botGoLogger) Info(v ...any) {
	message := fmt.Sprint(v...)
	if shouldDemoteBotGoInfo(message) {
		b.Logger.Debug(message)
		return
	}
	b.Logger.Info(message)
}

func (b *botGoLogger) Infof(format string, v ...any) {
	message := fmt.Sprintf(format, v...)
	if shouldDemoteBotGoInfo(message) {
		b.Logger.Debug(message)
		return
	}
	b.Logger.Info(message)
}

func shouldDemoteBotGoInfo(message string) bool {
	return strings.Contains(message, " write Heartbeat message") ||
		strings.Contains(message, " receive HeartbeatAck message")
}
