package application

import (
	"context"
	"fmt"
	"log/slog"
	"tg_market/internal/config"
	"tg_market/internal/domain/entity"
	service "tg_market/internal/domain/service/gift"
	"tg_market/internal/infrastructure/notifier"
	"tg_market/internal/infrastructure/persistence"
	"tg_market/internal/infrastructure/telegram"
	"tg_market/internal/worker"
	"tg_market/pkg/application/connectors"
)

func Run(ctx context.Context, log *slog.Logger, cancel context.CancelFunc) error {
	// 1. Config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}

	// 2. Database
	pg := &connectors.Postgres{
		DSN:             cfg.Postgres.DSN,
		MaxOpenConns:    cfg.Postgres.MaxOpenConns,
		MaxIdleConns:    cfg.Postgres.MaxIdleConns,
		ConnMaxLifetime: cfg.Postgres.ConnMaxLifetime,
	}
	db := pg.Client(ctx)
	defer pg.Close(ctx)

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("db ping: %w", err)
	}
	log.Info("database connection OK")

	// 3. Repositories
	giftTypeRepo := persistence.NewGiftTypeRepository(db)
	giftRepo := persistence.NewGiftRepository(db)

	// 4. Telegram Pool
	accounts, err := telegram.LoadAccounts("accounts.json")
	if err != nil {
		return fmt.Errorf("load accounts: %w", err)
	}
	log.Info("loaded accounts", "count", len(accounts))

	pool, err := telegram.NewPool(cfg.Telegram, accounts)
	if err != nil {
		return fmt.Errorf("create pool: %w", err)
	}

	go func() {
		log.Info("starting telegram pool...")
		if err := pool.Start(ctx); err != nil && ctx.Err() == nil {
			log.Error("telegram pool stopped", "error", err)
			cancel()
		}
	}()

	if err := pool.WaitReady(ctx); err != nil {
		return fmt.Errorf("wait pool ready: %w", err)
	}
	log.Info("✅ Telegram Pool Ready", "clients", pool.Size())

	dealsCh := make(chan entity.Deal, 100)

	// Notify bot

	alertBot, err := notifier.NewTelegramBot(cfg.Bot.Token, cfg.Bot.ChatID)
	if err != nil {
		return fmt.Errorf("notifier bot: %w", err)
	}
	log.Info("Testing bot notification...")
	if err := alertBot.SendText(ctx, "🚀 Bot is starting! Test message."); err != nil {
		log.Error("❌ Bot test failed! Check Token and ChatID", "err", err)
	} else {
		log.Info("✅ Bot test passed! Message sent.")
	}
	go func() {
		log.Info("notifier bot started listening")
		if err := alertBot.Run(ctx, dealsCh); err != nil {
			if ctx.Err() == nil {
				log.Error("notifier bot stopped", "error", err)
			}
		}
	}()

	svc := service.NewGiftService(giftTypeRepo, giftRepo, pool).
		WithDiscountThreshold(10)

	if err != nil {
		return fmt.Errorf("update all gift price: %w", err)
	}
	//опционально
	//svc.SyncCatalog(ctx)

	targetTypes := []int64{
		6014591077976114307, //snoop dog
		5773668482394620318,
		5935936766358847989, //snoop cigars
	}

	scanner := worker.NewMarketScanner(svc, giftTypeRepo, dealsCh).
		WithGiftTypes(targetTypes...).
		WithRateControl(cfg.Telegram.GetRatePerClient()/2, pool.Size())

	go func() {
		defer close(dealsCh)

		if err := scanner.Run(ctx); err != nil {
			if ctx.Err() == nil {
				log.Error("scanner died", "err", err)
				cancel()
			}
		}
	}()

	log.Info("scanner started", "targets", targetTypes)

	<-ctx.Done()

	log.Info("application stopping...")
	return nil
}
