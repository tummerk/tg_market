package logx

import (
	"fmt"
	"log/slog"

	"github.com/lmittmann/tint"
)

var Error = tint.Err //nolint:gochecknoglobals

func Stringer(name string, value fmt.Stringer) slog.Attr {
	return slog.String(name, value.String())
}
