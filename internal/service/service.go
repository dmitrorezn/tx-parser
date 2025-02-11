package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dmitrorezn/tx-parser/internal/domain"
	"github.com/dmitrorezn/tx-parser/pkg/converter"
	"github.com/dmitrorezn/tx-parser/pkg/logger"
)

type Servicer interface {
	// GetCurrentBlock - last parsed block
	GetCurrentBlock() int
	// Subscribe - add address to observer
	Subscribe(ctx context.Context, address domain.Address) error
	// GetTransactions -  list of inbound or outbound transactions for an address
	GetTransactions(ctx context.Context, address domain.Address) ([]domain.Transaction, error)
	// ProcessTransactions - defines current blockchain height and starting processing transactions in range
	// prevBlockNumber from last processed block and skips processing if all txs from block are already processed
	ProcessTransactions(ctx context.Context) (bool, error)
}

type Client interface {
	GetBlockNumber(ctx context.Context) (int, error)
	GetBlockTxsByNumber(ctx context.Context, number int) ([]domain.Transaction, error)
}

type BlocksStorage interface {
	GetCurrentBlock() int
	SetCurrentBlock(currBlock int)
	DelLastProcessedTxIndex(blockNumber int)
	GetLastProcessedTxIndex(block int) (int, bool)
	SetLastProcessedTxIndex(block int, idx int)
}

type Storage interface {
	AddSubscriber(ctx context.Context, addr domain.Address) error
	ExistsSubscriber(ctx context.Context, addr domain.Address) (bool, error)
	AddTx(ctx context.Context, addr domain.Address, tx domain.Transaction) error
	GetTransactions(ctx context.Context, addr domain.Address) ([]domain.Transaction, error)
}

type Service struct {
	cfg          Config
	client       Client
	blockStorage BlocksStorage
	storage      Storage
	logger       Logger
}

func NewConfig(
	txFetchInterval time.Duration,
	matcherWorkers int,
) Config {
	return Config{
		txFetchInterval: txFetchInterval,
		matcherWorkers:  matcherWorkers,
	}
}

type Config struct {
	txFetchInterval time.Duration
	matcherWorkers  int
}

type Logger interface {
	Error(ctx context.Context, msg string, args ...any)
	Info(ctx context.Context, msg string, args ...any)
}

func NewService(client Client, blockStorage BlocksStorage, storage Storage, logger Logger, cfg Config) *Service {
	return &Service{
		cfg:          cfg,
		client:       client,
		blockStorage: blockStorage,
		storage:      storage,
		logger:       logger,
	}
}

func (s *Service) Run(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	ctx = logger.NewAttrContext(ctx) // to handle attributes from upstream calls in logs
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		start := time.Now()
		if processed, err := s.ProcessTransactions(ctx); err != nil {
			s.logger.Error(ctx, "processTransactions",
				slog.Any("error", err),
				slog.String("process_time", time.Since(start).String()),
			)
		} else if processed {
			s.logger.Info(ctx, "processTransactions processed",
				slog.String("process_time", time.Since(start).String()),
			)
		}
		timer.Reset(s.cfg.txFetchInterval)
	}
}

func (s *Service) ProcessTransactions(ctx context.Context) (bool, error) {
	currentBlockNumber, err := s.client.GetBlockNumber(ctx)
	if err != nil {
		return false, err
	}
	var (
		prevBlockNumber = s.blockStorage.GetCurrentBlock()
		nextBlockNumber = prevBlockNumber + 1
	)
	if prevBlockNumber != 0 {
		currentBlockNumber = min(currentBlockNumber, nextBlockNumber)
	}

	// define if we already started processing current block
	// and define last processed transaction to avoid duplicated transactions
	var prevLastProcessedIndex int
	if prevBlockNumber == currentBlockNumber {
		prevLastProcessedIndex, _ = s.blockStorage.GetLastProcessedTxIndex(currentBlockNumber)
	}
	logger.AttrsFromCtx(ctx).PutAttrs(
		slog.Int("prevBlockNumber", prevBlockNumber),
		slog.Int("currentBlockNumber", currentBlockNumber),
		slog.Int("prevLastProcessedIndex", prevLastProcessedIndex),
	)
	txs, err := s.client.GetBlockTxsByNumber(ctx, currentBlockNumber)
	if err != nil {
		return false, err
	}
	if err = s.handleTransactionsMatching(ctx, currentBlockNumber, prevLastProcessedIndex, txs); err != nil {
		return true, err
	}

	return true, nil
}

type Stat struct {
	Processed atomic.Int32
	Skipped   atomic.Int32
	Matched   atomic.Int32
}

func (s *Stat) String() string {
	return fmt.Sprint(
		" Skipped: ", s.Skipped.Load(),
		" Processed: ", s.Processed.Load(),
		" Matched: ", s.Matched.Load(),
	)
}

func (s *Service) handleTransactions(
	ctx context.Context,
	stat *Stat,
	lastProcessedIndex int,
	txStream chan domain.Transaction,
	errsStream chan error,
) {
	var exist bool
	for tx := range txStream {
		txIdx, err := converter.ParseHexInt(tx.TransactionIndex)
		if err != nil {
			errsStream <- err

			continue
		}
		if lastProcessedIndex != 0 && txIdx <= lastProcessedIndex {
			stat.Skipped.Add(1)

			continue
		}
		stat.Processed.Add(1)

		for _, addr := range []domain.Address{tx.From, tx.To} {
			if exist, err = s.storage.ExistsSubscriber(ctx, addr); err != nil {
				errsStream <- err

				continue
			}
			if !exist {
				continue
			}
			stat.Matched.Add(1)
			if err = s.storage.AddTx(ctx, addr, tx); err != nil {
				errsStream <- err
			}
		}
	}
}

func (s *Service) handleTransactionsMatching(
	ctx context.Context,
	blockNumber int,
	lastProcessedIndex int,
	txs []domain.Transaction,
) (joinedErr error) {
	var (
		stat      = new(Stat)
		txStream  = make(chan domain.Transaction)
		errStream = make(chan error)
	)
	wg := sync.WaitGroup{}
	for i := 0; i < s.cfg.matcherWorkers; i++ {
		wg.Add(1)
		go func() {
			s.handleTransactions(ctx, stat, lastProcessedIndex, txStream, errStream)
			wg.Done()
		}()
	}
	go func() {
		for _, tx := range txs {
			txStream <- tx
		}
		close(txStream)
	}()
	go func() {
		wg.Wait()
		close(errStream)
	}()
	for err := range errStream {
		joinedErr = errors.Join(joinedErr, err)
	}
	s.blockStorage.SetCurrentBlock(blockNumber)

	lastProcessedTxIndex, err := converter.ParseHexInt(txs[len(txs)-1].TransactionIndex)
	if err != nil {
		return errors.Join(joinedErr, err)
	}
	s.blockStorage.SetLastProcessedTxIndex(blockNumber, lastProcessedTxIndex)

	logger.AttrsFromCtx(ctx).PutAttrs(
		slog.Int("tx_len", len(txs)),
		slog.Int("last_tx_idx", lastProcessedTxIndex),
		slog.Any("stat", stat.String()),
	)

	return joinedErr
}

func (s *Service) GetCurrentBlock() int {
	return s.blockStorage.GetCurrentBlock()
}

func (s *Service) Subscribe(ctx context.Context, address domain.Address) error {
	if !address.Valid() {
		return domain.ErrInvalidAddress
	}

	return s.storage.AddSubscriber(ctx, address)
}

func (s *Service) GetTransactions(ctx context.Context, address domain.Address) ([]domain.Transaction, error) {
	if !address.Valid() {
		return nil, domain.ErrInvalidAddress
	}
	exist, err := s.storage.ExistsSubscriber(ctx, address)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, domain.ErrAddressNotSubscribed
	}

	return s.storage.GetTransactions(ctx, address)
}
