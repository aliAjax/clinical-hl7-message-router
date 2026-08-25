package infrastructure

import "strings"

func Normalize(raw string) string { return strings.ReplaceAll(strings.TrimSpace(raw), "\n", "\r") }
