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
	"github.com/microcloud/gen/go/ops/v1/opsv1connect"
	"github.com/microcloud/logger"
	"github.com/microcloud/orchestrator/server"
	"github.com/microcloud/storage"
)

func main() {
	log := logger.NewFromEnv("orchestrator")

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

	actionServer := server.NewActionServer(actionsRepo, publisher, log)
	streamHub := server.NewStreamHub(subscriber, log)

	mux := http.NewServeMux()

	// Connect-RPC handlers
	path, handler := opsv1connect.NewActionServiceHandler(actionServer,
		connect.WithInterceptors(loggingInterceptor(log)),
	)
	mux.Handle(path, handler)

	// SSE streaming endpoint
	mux.Handle("/api/stream", streamHub)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// CORS middleware
	corsHandler := corsMiddleware(mux)

	addr := getEnv("ADDR", ":8081")
	httpServer := &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(corsHandler, &http2.Server{}),
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return streamHub.Start(ctx)
	})

	g.Go(func() error {
		log.Info("orchestrator API started", "addr", addr)
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

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
