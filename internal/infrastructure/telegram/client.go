package telegram

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"

	"tg_market/internal/config"
)

// ConsoleInput реализует ввод кода с клавиатуры
type ConsoleInput struct{}

func (c ConsoleInput) Code(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	fmt.Print("Введите код из Telegram: ")
	text, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

type Client struct {
	client *telegram.Client
	api    *tg.Client // raw API
	cfg    config.Telegram
}

func NewClient(cfg config.Telegram) (*Client, error) {
	sessionDir := "storage"
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create session dir: %w", err)
	}
	sessionPath := filepath.Join(sessionDir, "session.json")

	sessionStorage := &telegram.FileSessionStorage{
		Path: sessionPath,
	}

	zapLogger, _ := zap.NewProduction()

	opts := telegram.Options{
		SessionStorage: sessionStorage,
		Logger:         zapLogger,
	}

	client := telegram.NewClient(cfg.ApiID, cfg.ApiHash, opts)

	return &Client{
		client: client,
		api:    client.API(),
		cfg:    cfg,
	}, nil
}

// Start поднимает соединение и держит его открытым.
func (c *Client) Start(ctx context.Context, onReady func() error) error {
	return c.client.Run(ctx, func(ctx context.Context) error {
		status, err := c.client.Auth().Status(ctx)
		if err != nil {
			return fmt.Errorf("auth status error: %w", err)
		}

		if !status.Authorized {
			logger(ctx).Info("User not authorized, starting login flow...")
			if err := c.authenticate(ctx); err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}
			logger(ctx).Info("Authentication successful!")
		} else {
			logger(ctx).Info("User already authorized")
		}

		// Сигнализируем наверх, что соединение установлено и авторизация прошла.
		// Сервис может начинать слать запросы.
		if onReady != nil {
			if err := onReady(); err != nil {
				return err
			}
		}

		<-ctx.Done()
		return ctx.Err()
	})
}

func (c *Client) authenticate(ctx context.Context) error {
	userAuth := auth.Constant(
		c.cfg.Phone,
		c.cfg.Password,
		ConsoleInput{},
	)

	flow := auth.NewFlow(
		userAuth,
		auth.SendCodeOptions{},
	)

	return c.client.Auth().IfNecessary(ctx, flow)
}
