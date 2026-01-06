package logx_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go-backend-example/pkg/logx"
)

func TestSensitiveDataMaskerMask(t *testing.T) {
	rq := require.New(t)

	masker := logx.NewSensitiveDataMasker()

	testCases := []struct {
		name   string
		input  []byte
		output []byte
	}{
		{
			name:   "Password",
			input:  []byte(`{"hello":"world","password":"abc123"}`),
			output: []byte(`{"hello":"world","password":"[MASKED]"}`),
		},
		{
			name:   "Password capital letter",
			input:  []byte(`{"hello":"world","Password":"abc123"}`),
			output: []byte(`{"hello":"world","Password":"[MASKED]"}`),
		},
		{
			name:   "Access token",
			input:  []byte(`{"accessToken":"eyJhbGciOiJFUzI1NiIsInR5cC","refreshToken":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9"}`),
			output: []byte(`{"accessToken":"[MASKED]","refreshToken":"[MASKED]"}`),
		},
		{
			name:   "First name, last name, middle name and email",
			input:  []byte(`{"profile": {"lastName": "Doe", "firstName": "John", "middleName": "Michael", "email": "john@doe.com"}, "isMarketingConsentPermitted": true}`),
			output: []byte(`{"profile": {"lastName": "[MASKED]", "firstName": "[MASKED]", "middleName": "[MASKED]", "email": "[MASKED]"}, "isMarketingConsentPermitted": true}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(*testing.T) {
			output := masker.Mask(tc.input)

			rq.Equal(tc.output, output, "%s vs %s", tc.output, output)
		})
	}
}
