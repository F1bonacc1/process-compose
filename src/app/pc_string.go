package app

import (
	"fmt"
	"strings"
	"time"
)

func isStringDefined(str string) bool {
	return len(strings.TrimSpace(str)) > 0
}

func durationToString(dur time.Duration) string {
	if dur.Minutes() < 3 {
		return dur.Round(time.Second).String()
	} else if dur.Minutes() < 60 {
		return fmt.Sprintf("%.0fm", dur.Minutes())
	} else if dur.Hours() < 24 {
		return fmt.Sprintf("%dh%dm", int(dur.Hours()), int(dur.Minutes())%60)
	} else {
		return fmt.Sprintf("%dh", int(dur.Hours()))
	}
}
