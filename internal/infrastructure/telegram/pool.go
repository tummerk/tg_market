package telegram

import (
	"context"
	"fmt"
	"github.com/gotd/td/telegram"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"tg_market/internal/config"
	"tg_market/internal/domain/entity"
)

type clientWrapper struct {
	client *Client
	index  int
}

type ClientPool struct {
	clients []*clientWrapper
	index   atomic.Uint64
	ready   chan struct{}
	mu      sync.Mutex
}

func NewPool(cfg config.Telegram, accounts []Account) (*ClientPool, error) {
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no accounts provided")
	}

	pool := &ClientPool{
		clients: make([]*clientWrapper, 0, len(accounts)),
		ready:   make(chan struct{}),
	}

	for i, acc := range accounts {
		client, err := newClientWithSession(cfg, acc, fmt.Sprintf("session_%d", i))
		if err != nil {
			return nil, fmt.Errorf("create client %d (%s): %w", i, acc.Phone, err)
		}

		pool.clients = append(pool.clients, &clientWrapper{
			client: client,
			index:  i,
		})
	}

	return pool, nil
}

func newClientWithSession(cfg config.Telegram, acc Account, sessionName string) (*Client, error) {
	sessionDir := "storage/sessions"
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, fmt.Errorf("create session dir: %w", err)
	}

	sessionPath := filepath.Join(sessionDir, sessionName+".json")
	sessionStorage := &telegram.FileSessionStorage{Path: sessionPath}

	zapLogger := zap.NewNop()

	opts := telegram.Options{
		SessionStorage: sessionStorage,
		Logger:         zapLogger,
	}

	client := telegram.NewClient(cfg.ApiID, cfg.ApiHash, opts)

	return &Client{
		client:   client,
		api:      client.API(),
		Phone:    acc.Phone,
		Password: acc.Password,
	}, nil
}

func (p *ClientPool) Start(ctx context.Context) error {
	var wg sync.WaitGroup
	readyCount := atomic.Int32{}
	errCh := make(chan error, len(p.clients))

	for i, cw := range p.clients {
		wg.Add(1)

		go func(idx int, c *clientWrapper) {
			defer wg.Done()

			err := c.client.Start(ctx, func() error {
				count := readyCount.Add(1)
				fmt.Printf("✅ Client %d (%s) ready [%d/%d]\n", idx, c.client.Phone, count, len(p.clients))

				if int(count) == len(p.clients) {
					close(p.ready)
				}
				return nil
			})

			if err != nil && ctx.Err() == nil {
				errCh <- fmt.Errorf("client %d: %w", idx, err)
			}
		}(i, cw)
	}

	select {
	case <-p.ready:
		fmt.Println("🚀 All clients ready!")
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}

	<-ctx.Done()
	wg.Wait()
	return ctx.Err()
}

func (p *ClientPool) WaitReady(ctx context.Context) error {
	select {
	case <-p.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// next — просто round-robin, без задержек
func (p *ClientPool) next() *Client {
	p.mu.Lock()
	defer p.mu.Unlock()

	idx := p.index.Add(1) % uint64(len(p.clients))
	cw := p.clients[idx]

	return cw.client
}

func (p *ClientPool) Size() int {
	return len(p.clients)
}

// TgClient interface
func (p *ClientPool) GetGiftTypes(ctx context.Context, hash int) ([]entity.GiftType, error) {
	return p.next().GetGiftTypes(ctx, hash)
}

func (p *ClientPool) GetMarketDeals(ctx context.Context, giftTypeID int64, limit int) ([]entity.Deal, error) {
	return p.next().GetMarketDeals(ctx, giftTypeID, limit)
}

func (p *ClientPool) GetLastPrices(ctx context.Context, giftTypeID int, limit int) ([]int, error) {
	return p.next().GetLastPrices(ctx, giftTypeID, limit)
}

func (p *ClientPool) GetGiftsPage(ctx context.Context, giftID int64, offset string, limit int) ([]entity.Gift, string, error) {
	return p.next().GetGiftsPage(ctx, giftID, offset, limit)
}
