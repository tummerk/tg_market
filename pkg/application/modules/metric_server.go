package modules

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"go-backend-example/pkg/metrics"
)

type MetricServer struct {
	ListenAddress string
}

func (m MetricServer) Run(ctx context.Context, g *errgroup.Group) {
	prometheusServer := metrics.NewPrometheusServer(
		m.ListenAddress,
	)

	g.Go(func() error {
		if err := prometheusServer.Run(ctx); err != nil {
			return fmt.Errorf("prometheusServer.Run: %w", err)
		}

		return nil
	})
}
