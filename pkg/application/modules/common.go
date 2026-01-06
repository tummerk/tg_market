package modules

import "go-backend-example/pkg/contextx"

var logger = contextx.LoggerFromContextOrDefault //nolint:gochecknoglobals
