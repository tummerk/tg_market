package bot

import (
	"context"
	"fmt"
	"log"
	"tg_market/internal/worker"

	"tg_market/internal/config"
	"tg_market/internal/domain/service/gift"
	"tg_market/internal/transport/bot/handler"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

// Bot представляет собой Telegram-бота
type Bot struct {
	bot        *telego.Bot
	botHandler *th.BotHandler

	handler *handler.Handler
}

// New создает новый экземпляр бота
func New(cfg config.Config,
	svc *service.GiftService,
	scanner *worker.MarketScanner,
) (*Bot, error) {
	// Создаем экземпляр бота
	bot, err := telego.NewBot(cfg.Bot.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	// Получаем обновления через long polling
	updates, err := bot.UpdatesViaLongPolling(context.Background(), &telego.GetUpdatesParams{
		Timeout: 60,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get updates: %w", err)
	}

	// Создаем BotHandler
	botHandler, err := th.NewBotHandler(bot, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot handler: %w", err)
	}

	// Создаем обработчик команд
	commandHandler := handler.New(svc, scanner) // <--- Передали сюда

	commandHandler.RegisterRoutes(botHandler, cfg.Bot.AdminID)

	return &Bot{
		bot:        bot,
		botHandler: botHandler,
		handler:    commandHandler,
	}, nil
}

// Run запускает бота
func (b *Bot) Run(ctx context.Context) error {
	// Запускаем обработку обновлений
	go func() {
		if err := b.botHandler.Start(); err != nil {
			log.Printf("Failed to start bot handler: %v", err)
		}
	}()

	// Ждем завершения
	<-ctx.Done()

	// Останавливаем обработчик
	if err := b.botHandler.Stop(); err != nil {
		log.Printf("Failed to stop bot handler: %v", err)
	}

	return ctx.Err()
}
