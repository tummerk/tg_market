package httpx

type Option func(*LoggingRoundTripper)

func WithLogFieldMaxLen(logFieldMaxLen int) Option {
	return func(rt *LoggingRoundTripper) {
		rt.logFieldMaxLen = logFieldMaxLen
	}
}

func WithSensitiveDataMasker(sensitiveDataMasker sensitiveDataMasker) Option {
	return func(rt *LoggingRoundTripper) {
		rt.sensitiveDataMasker = sensitiveDataMasker
	}
}
