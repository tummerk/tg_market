package logx

import (
	"regexp"
)

type SensitiveDataMaskerInterface interface {
	Mask(input []byte) []byte
}

//nolint:gochecknoglobals
var sensitiveDataPatterns = []*regexp.Regexp{
	// JSON fields.
	regexp.MustCompile("(?s)(Authorization: Bearer ).+?(\r)"),
	regexp.MustCompile(`(?s)("[Pp]assword":\s?").+?(")`),
	regexp.MustCompile(`(?s)("accessToken":\s?").+?(")`),
	regexp.MustCompile(`(?s)("refreshToken":\s?").+?(")`),
	regexp.MustCompile(`(?s)("firstName":\s?").+?(")`),
	regexp.MustCompile(`(?s)("middleName":\s?").+?(")`),
	regexp.MustCompile(`(?s)("lastName":\s?").+?(")`),
	regexp.MustCompile(`(?s)("email":\s?").+?(")`),
}

type SensitiveDataMasker struct{}

func NewSensitiveDataMasker() SensitiveDataMasker {
	return SensitiveDataMasker{}
}

func (s SensitiveDataMasker) Mask(input []byte) []byte {
	for _, pattern := range sensitiveDataPatterns {
		input = pattern.ReplaceAll(input, []byte("${1}[MASKED]${2}"))
	}

	return input
}
