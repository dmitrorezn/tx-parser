package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dmitrorezn/tx-parser/internal/service"
	ethrpcclient "github.com/dmitrorezn/tx-parser/internal/service/client/eth-client"
	"github.com/dmitrorezn/tx-parser/internal/service/ports/http"
	"github.com/dmitrorezn/tx-parser/internal/service/storage/memory"
	"github.com/dmitrorezn/tx-parser/pkg/logger"
)

var (
	addr             = flag.String("addr", "localhost:80", "http server address")
	ethAddr          = flag.String("eth_addr", "https://ethereum-rpc.publicnode.com", "http server address")
	fetchTxsInterval = flag.Duration("interval", 10*time.Second, "fetch transactions interval")
	blockStart       = flag.Int("blockStart", 0, "block from where to start")
)

func main() {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	loggr := logger.NewAttrLogger(logger.NewLogger(
		logger.WithHandlerFactory(logger.JSONFactory),
		logger.WithWriter(os.Stdout),
		logger.WithLevel(slog.LevelDebug),
	))
	client, err := ethrpcclient.NewJsonRpcClient(*ethAddr)
	if err != nil {
		loggr.Panic(ctx, "NewJsonRpcClient", slog.Any("error", err))
	}
	var (
		storage          = memory.NewStorage()
		blockNumberStore = memory.NewBlockNumberStorage()
		cfg              = service.NewConfig(*fetchTxsInterval)
		svc              = service.NewService(client, blockNumberStore, storage, loggr, cfg)
		handler          = httpport.NewHandler(svc)
	)
	if *blockStart != 0 {
		blockNumberStore.SetCurrentBlock(*blockStart)
	}
	httpServer := &http.Server{
		Addr:    *addr,
		Handler: handler,
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		svc.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			loggr.Error(ctx, "ListenAndServe", slog.Any("error", err))

			cancel()
		}
	}()
	loggr.Info(ctx, "Server started")

	<-ctx.Done()
	ctx = context.Background()
	if err = httpServer.Shutdown(ctx); err != nil {
		// force close server connections
		err = errors.Join(err, httpServer.Close())
		loggr.Error(ctx, "Shutdown", slog.Any("error", err))
	}

	wg.Wait()
	loggr.Info(ctx, "Server gracefully stopped")
	loggr.Info(ctx, "LAST_PROCESSED_BLOCK", slog.Int("NUMBER", blockNumberStore.GetCurrentBlock()))
}
