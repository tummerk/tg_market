package httpx

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/rs/xid"

	"go-backend-example/pkg/logx"
)

//go:generate moq -rm -out sensitive_data_masker_mock.gen.go . sensitiveDataMasker:SensitiveDataMaskerMock
type sensitiveDataMasker interface {
	Mask([]byte) []byte
}

// LoggingRoundTripper implements http.RoundTripper interface and executes HTTP
// requests with logging.
type LoggingRoundTripper struct {
	next                http.RoundTripper
	sensitiveDataMasker sensitiveDataMasker
	logFieldMaxLen      int
}

// NewLoggingRoundTripper returns a new logging RoundTripper instance.
func NewLoggingRoundTripper(
	next http.RoundTripper,
	opts ...Option,
) LoggingRoundTripper {
	rt := LoggingRoundTripper{
		next:                next,
		sensitiveDataMasker: logx.NewNopSensitiveDataMasker(),
		logFieldMaxLen:      0,
	}

	for _, opt := range opts {
		opt(&rt)
	}

	return rt
}

// RoundTrip implements http.RoundTripper interface.
func (rt LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	requestID := xid.New().String()

	reqBytes, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		logger(ctx).Error(
			"httputil.DumpRequestOut",
			slog.String(logx.FieldRequestID, requestID),
			logx.Error(err),
		)
	}

	if rt.logFieldMaxLen != 0 && len(reqBytes) > rt.logFieldMaxLen {
		reqBytes = reqBytes[:rt.logFieldMaxLen]
	}

	logger(ctx).Info(
		logx.FieldHTTPRequest,
		slog.String(logx.FieldRequestID, requestID),
		slog.String(logx.FieldRequestBody, string(rt.sensitiveDataMasker.Mask(reqBytes))),
	)

	start := time.Now()

	resp, err := rt.next.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("next.RoundTrip %w", err)
	}

	respBytes, err := httputil.DumpResponse(resp, true)
	if err != nil {
		logger(ctx).Error(
			"httputil.DumpResponse",
			slog.String(logx.FieldRequestID, requestID),
			logx.Error(err),
		)
	}

	if rt.logFieldMaxLen != 0 && len(respBytes) > rt.logFieldMaxLen {
		respBytes = respBytes[:rt.logFieldMaxLen]
	}

	logger(ctx).Info(
		logx.FieldHTTPResponse,
		slog.String(logx.FieldRequestID, requestID),
		slog.String(logx.FieldResponseBody, string(rt.sensitiveDataMasker.Mask(respBytes))),
		slog.Int64(logx.FieldDurationMs, time.Since(start).Milliseconds()),
	)

	return resp, nil
}
