package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/microcloud/bus"
	simv1 "github.com/microcloud/gen/go/sim/v1"
	"github.com/microcloud/logger"
	"github.com/microcloud/signal-service/detector"
	"github.com/microcloud/storage"
)

func main() {
	log := logger.NewFromEnv("signal-service")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx, log); err != nil && err != context.Canceled {
		log.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger) error {
	dbCfg := storage.ConfigFromEnv()
	db, err := storage.New(ctx, dbCfg)
	if err != nil {
		return err
	}
	defer db.Close()

	log.Info("connected to database", "host", dbCfg.Host)

	if err := db.Migrate(ctx); err != nil {
		log.Warn("migration error (may be expected if tables exist)", "error", err)
	}

	busCfg := bus.DefaultConfig()
	if url := os.Getenv("NATS_URL"); url != "" {
		busCfg.URL = url
	}

	eventBus, err := bus.New(ctx, busCfg,
		bus.WithDisconnectHandler(func(err error) {
			log.Warn("NATS disconnected", "error", err)
		}),
		bus.WithReconnectHandler(func() {
			log.Info("NATS reconnected")
		}),
	)
	if err != nil {
		return err
	}
	defer eventBus.Close()

	log.Info("connected to NATS", "url", busCfg.URL)

	publisher := bus.NewPublisher(eventBus)
	subscriber := bus.NewSubscriber(eventBus)
	metricsRepo := storage.NewMetricsRepository(db)

	det := detector.New(publisher, metricsRepo, log)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Info("subscribing to metrics")
		cc, err := subscriber.SubscribeMetrics(ctx, "signal-service", func(ctx context.Context, snapshot *simv1.MetricSnapshot) error {
			return det.ProcessSnapshot(ctx, snapshot)
		})
		if err != nil {
			return err
		}
		defer cc.Stop()

		<-ctx.Done()
		return ctx.Err()
	})

	return g.Wait()
}
