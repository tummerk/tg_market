package modules

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"go-backend-example/pkg/probe"
)

type ProbeServer struct {
	Name          string
	Version       string
	ListenAddress string
}

func (p ProbeServer) Run(ctx context.Context, g *errgroup.Group) {
	probeServer := probe.NewServer(
		p.ListenAddress,
		probe.Options{
			Name:    p.Name,
			Version: p.Version,
		},
	)

	g.Go(func() error {
		if err := probeServer.Run(ctx); err != nil {
			return fmt.Errorf("probeServer.Run: %w", err)
		}

		return nil
	})
}
