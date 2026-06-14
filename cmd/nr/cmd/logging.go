package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"
)

func logCommandStep(scope string, format string, args ...any) {
	log.Printf("[%s] %s", scope, fmt.Sprintf(format, args...))
}

func logCommandSuccess(scope string, format string, args ...any) {
	log.Printf("[%s] OK %s", scope, fmt.Sprintf(format, args...))
}

func logCommandWarn(scope string, format string, args ...any) {
	log.Printf("[%s] WARN %s", scope, fmt.Sprintf(format, args...))
}

func logCommandOutput(scope string, title string, output string) {
	output = strings.TrimSpace(output)
	if output == "" {
		return
	}
	log.Printf("[%s] %s\n%s", scope, title, output)
}

func logCommandSummary(scope string, start time.Time, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("[%s] DONE %s (%s)", scope, msg, time.Since(start).Round(time.Millisecond))
}
