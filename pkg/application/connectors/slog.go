package connectors

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/lmittmann/tint"
)

type Slog struct {
	value   *slog.Logger
	Name    string
	Version string
	init    sync.Once
	Debug   bool
}

func (s *Slog) Logger(_ context.Context) *slog.Logger {
	s.init.Do(func() {
		// Для локальной разработки используется цветной логгер.
		if s.Debug {
			s.value = slog.New(tint.NewHandler(os.Stdout, &tint.Options{
				AddSource:   false,
				Level:       slog.LevelDebug,
				ReplaceAttr: nil,
				TimeFormat:  time.StampMilli,
				NoColor:     false,
			}))

			return
		}

		s.value = slog.New(slog.NewJSONHandler(os.Stdout, nil)).With(
			slog.Group("app",
				slog.String("name", s.Name),
				slog.String("version", s.Version),
			),
		)
	})

	return s.value
}
