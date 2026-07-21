package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"
)

const commandTimeLayout = "2006-01-02 15:04:05.000"

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

func logCommandPhaseStart(scope string, phase string, start time.Time) {
	log.Printf("[%s] %s started at %s", scope, phase, start.Format(commandTimeLayout))
}

func logCommandPhaseDone(scope string, phase string, start time.Time, end time.Time) {
	log.Printf("[%s] %s finished at %s", scope, phase, end.Format(commandTimeLayout))
	log.Printf("[%s] %s elapsed %s", scope, phase, end.Sub(start).Round(time.Millisecond))
}
