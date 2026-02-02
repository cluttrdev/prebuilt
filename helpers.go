package main

import (
	"regexp"
)

// mayBeEnvVar checks if the string matches the pattern of an environment variable
// reference like ${VAR_NAME}.
// If it does, it returns the variable name and `true`.
func mayBeEnvVar(s string) (string, bool) {
	pattern := regexp.MustCompile(`\$\{(?<name>[a-zA-Z_]+[a-zA-Z0-9_]*)\}`)
	matches := pattern.FindStringSubmatch(s)
	if matches == nil {
		return "", false
	}
	return matches[1], true
}
