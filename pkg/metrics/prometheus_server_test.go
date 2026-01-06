package metrics_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"go-backend-example/pkg/metrics"
)

func TestPrometheusServer(t *testing.T) {
	rq := require.New(t)

	testCases := []struct {
		name          string
		listenAddress string
		endpoint      string
		statusCode    int
	}{
		{
			name:          "Metrics handler",
			listenAddress: ":10010",
			endpoint:      "http://:10010/metrics",
			statusCode:    http.StatusOK,
		},
		{
			name:          "Invalid endpoint",
			listenAddress: ":10020",
			endpoint:      "http://:10020/invalid",
			statusCode:    http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(*testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			prometheusServer := metrics.NewPrometheusServer(tc.listenAddress)

			g, ctx := errgroup.WithContext(ctx)

			g.Go(func() error {
				return prometheusServer.Run(ctx)
			})

			// Wait for server to start.
			time.Sleep(time.Second)

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, tc.endpoint, http.NoBody)
			rq.NoError(err)

			resp, err := http.DefaultClient.Do(req)
			rq.NoError(err)

			defer resp.Body.Close()

			rq.Equal(tc.statusCode, resp.StatusCode)

			cancel()

			rq.NoError(g.Wait())
		})
	}
}
