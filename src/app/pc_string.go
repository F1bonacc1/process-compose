package app

import "strings"

func isStringDefined(str string) bool {
	return len(strings.TrimSpace(str)) > 0
}
