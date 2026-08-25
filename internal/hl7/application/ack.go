package application

import "strings"

func SafeType(raw string) string {
	for _, s := range strings.Split(raw, "\r") {
		if strings.HasPrefix(s, "MSH|") {
			f := strings.Split(s, "|")
			if len(f) > 8 {
				return strings.Split(f[8], "^")[0]
			}
		}
	}
	return "UNKNOWN"
}
func SafeControl(raw string) string {
	for _, s := range strings.Split(raw, "\r") {
		if strings.HasPrefix(s, "MSH|") {
			f := strings.Split(s, "|")
			if len(f) > 9 {
				return f[9]
			}
		}
	}
	return "NA"
}
