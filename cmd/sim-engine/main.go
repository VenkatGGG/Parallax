package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"

	"github.com/microcloud/bus"
	"github.com/microcloud/gen/go/sim/v1/simv1connect"
	"github.com/microcloud/logger"
	"github.com/microcloud/sim-engine/engine"
	"github.com/microcloud/sim-engine/server"
)

func main() {
	log := logger.NewFromEnv("sim-engine")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx, log); err != nil && err != context.Canceled {
		log.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger) error {
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
	eng := engine.New(publisher, log)
	controlServer := server.NewControlServer(eng, log)

	mux := http.NewServeMux()
	path, handler := simv1connect.NewSimulationControlHandler(controlServer,
		connect.WithInterceptors(loggingInterceptor(log)),
	)
	mux.Handle(path, handler)

	addr := getEnv("ADDR", ":8080")
	httpServer := &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return eng.Run(ctx)
	})

	g.Go(func() error {
		log.Info("gRPC server started", "addr", addr)
		return httpServer.ListenAndServe()
	})

	g.Go(func() error {
		<-ctx.Done()
		log.Info("shutting down...")
		return httpServer.Close()
	})

	return g.Wait()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loggingInterceptor(log *slog.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			log.Debug("rpc call", "procedure", req.Spec().Procedure)
			resp, err := next(ctx, req)
			if err != nil {
				log.Error("rpc error", "procedure", req.Spec().Procedure, "error", err)
			}
			return resp, err
		}
	}
}
