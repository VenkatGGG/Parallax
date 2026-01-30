package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/microcloud/agent-service/decider"
	"github.com/microcloud/bus"
	opsv1 "github.com/microcloud/gen/go/ops/v1"
	"github.com/microcloud/logger"
	"github.com/microcloud/storage"
)

func main() {
	log := logger.NewFromEnv("agent-service")

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
	actionsRepo := storage.NewActionsRepository(db)
	incidentsRepo := storage.NewIncidentsRepository(db)

	dec := decider.New(publisher, actionsRepo, incidentsRepo, log)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Info("subscribing to incidents")
		cc, err := subscriber.SubscribeIncidents(ctx, "agent-service", func(ctx context.Context, incident *opsv1.Incident) error {
			return dec.ProcessIncident(ctx, incident)
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
