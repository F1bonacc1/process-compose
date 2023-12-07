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
		//seconds
		return dur.Round(time.Second).String()
	} else if dur.Minutes() < 60 {
		//minutes
		return fmt.Sprintf("%.0fm", dur.Minutes())
	} else if dur.Hours() < 24 {
		//hours and minutes
		return fmt.Sprintf("%dh%dm", int(dur.Hours()), int(dur.Minutes())%60)
	} else if dur.Hours() < 48 {
		//days and hours
		return fmt.Sprintf("%dd%dh", int(dur.Hours())/24, int(dur.Hours())%24)
	} else {
		//days
		return fmt.Sprintf("%dd", int(dur.Hours())/24)
	}
}
