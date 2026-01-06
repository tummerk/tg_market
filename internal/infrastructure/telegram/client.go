package telegram

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
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
	client   *telegram.Client
	api      *tg.Client
	Phone    string
	Password string
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
		c.Phone,
		c.Password,
		ConsoleInput{},
	)

	flow := auth.NewFlow(
		userAuth,
		auth.SendCodeOptions{},
	)

	return c.client.Auth().IfNecessary(ctx, flow)
}
