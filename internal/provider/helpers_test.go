package provider

import (
	"github.com/thanhpk/randstr"
	"strings"
	"testing"
)

// GenerateServiceName a random service name
func GenerateServiceName(t *testing.T) string {

	t.Helper()

	const maxAttempts = 10

	attempt := 0

GENERATE:
	serviceName := randstr.String(24)

	runes := []rune(serviceName)
	if len(runes) >= 1 {
		runes[0] = 's'
	} else {
		if attempt > maxAttempts {
			panic("can not generate service name")
		}
		attempt++

		goto GENERATE
	}

	return strings.ToLower(string(runes))
}
