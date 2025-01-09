package service_test

import (
	"context"
	"encoding/hex"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/dmitrorezn/tx-parser/internal/domain"
	"github.com/dmitrorezn/tx-parser/internal/service"
	"github.com/dmitrorezn/tx-parser/internal/service/storage/memory"
	"github.com/dmitrorezn/tx-parser/pkg/converter"
	"github.com/dmitrorezn/tx-parser/pkg/logger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type EthRpcClient struct {
	mock.Mock
}

func (e *EthRpcClient) GetBlockNumber(_ context.Context) (int, error) {
	for _, call := range e.ExpectedCalls {
		if call.Method == "GetBlockNumber" {
			return call.ReturnArguments.Get(0).(int), call.ReturnArguments.Error(1)
		}
	}
	return 0, errors.New("not found mock GetBlockNumber")
}

func (e *EthRpcClient) GetBlockTxsByNumber(_ context.Context, _ int) ([]domain.Transaction, error) {
	for _, call := range e.ExpectedCalls {
		if call.Method == "GetBlockTxsByNumber" {
			return call.ReturnArguments.Get(0).([]domain.Transaction), call.ReturnArguments.Error(1)
		}
	}

	return nil, errors.New("not found mock GetBlockTxsByNumber")
}

var _ service.Client = (*EthRpcClient)(nil)

func setup(t *testing.T, currentBlock int) (*service.Service, service.BlocksStorage, *EthRpcClient) {
	ethClient := &EthRpcClient{}

	var (
		loggr            = logger.NewAttrLogger(logger.NewLogger())
		storage          = memory.NewStorage()
		blockNumberStore = memory.NewBlockNumberStorage()
		cfg              = service.NewConfig(100*time.Millisecond, 10)
		svc              = service.NewService(ethClient, blockNumberStore, storage, loggr, cfg)
	)
	blockNumberStore.SetCurrentBlock(currentBlock)

	return svc, blockNumberStore, ethClient
}

func TestGetCurrentBlock(t *testing.T) {
	ctx := context.Background()

	const (
		block = 100
	)

	svc, _, ethClient := setup(t, 0)

	ethClient.On("GetBlockNumber", mock.Anything).Return(block, error(nil))

	tests := map[string]struct {
		expectedBlock int
		preconditions func()
	}{
		"1. Success: get initial block head (not started svc)": {
			expectedBlock: 0,
		},
		"2. Success: current block equal to blockchain head": {
			expectedBlock: block,

			preconditions: func() {
				ethClient.On("GetBlockNumber", mock.Anything).Return(block, error(nil))
				ethClient.On("GetBlockTxsByNumber", mock.Anything, mock.Anything).Return([]domain.Transaction{
					{From: genAddress(), TransactionIndex: converter.FormatHexInt(1)},
				}, error(nil))

				// add any address to processor to start processing blocks
				require.NoError(t, svc.Subscribe(ctx, genAddress()))

				// move to blockchain head block
				processes, err := svc.ProcessTransactions(ctx)
				require.NoError(t, err)
				require.True(t, processes)
			},
		},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			if testCase.preconditions != nil {
				testCase.preconditions()
			}
			block := svc.GetCurrentBlock()
			require.Equal(t, testCase.expectedBlock, block)
		})
	}
}

func genAddress() domain.Address {
	var addr [20]byte
	_, _ = rand.Read(addr[:])

	return domain.Address("0x" + hex.EncodeToString(addr[:]))
}

func TestSubscribe(t *testing.T) {
	ctx := context.Background()

	const (
		block = 100
	)
	svc, _, ethClient := setup(t, 0)

	ethClient.On("GetBlockNumber", mock.Anything).Return(block, error(nil))

	var (
		randAddr = genAddress()
	)
	tests := map[string]struct {
		address       domain.Address
		expectedErr   error
		preconditions func()
	}{
		"1. Err address not valid": {
			expectedErr: domain.ErrInvalidAddress,
		},
		"2. Err Already Subscribed": {
			address:     randAddr,
			expectedErr: domain.ErrAddressAlreadySubscribed,
			preconditions: func() {
				require.NoError(t, svc.Subscribe(ctx, randAddr))
			},
		},
		"2. Success Subscribed": {
			address:     genAddress(),
			expectedErr: nil,
		},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			if testCase.preconditions != nil {
				testCase.preconditions()
			}
			err := svc.Subscribe(ctx, testCase.address)
			require.ErrorIs(t, err, testCase.expectedErr)
		})
	}
}

func TestGetTransactions(t *testing.T) {
	ctx := context.Background()
	const (
		block = 100
	)
	svc, blockNumberStore, ethClient := setup(t, block)

	var (
		randAddr   = genAddress()
		succesAddr = genAddress()
	)

	tests := map[string]struct {
		address       domain.Address
		txs           []domain.Transaction
		expectedErr   error
		preconditions func(addr domain.Address)
	}{
		"1. Err address not subscribed": {
			txs:         nil,
			address:     genAddress(),
			expectedErr: domain.ErrAddressNotSubscribed,
		},
		"2. Err No transaction": {
			txs:         nil,
			address:     randAddr,
			expectedErr: domain.ErrNoTransactions,
			preconditions: func(addr domain.Address) {
				require.NoError(t, svc.Subscribe(ctx, addr))
			},
		},
		"2. Success transaction": {
			txs: []domain.Transaction{
				{From: succesAddr, TransactionIndex: converter.FormatHexInt(1)},
			},
			address:     succesAddr,
			expectedErr: nil,
			preconditions: func(addr domain.Address) {
				ethClient.On("GetBlockTxsByNumber", mock.Anything, mock.Anything).Return([]domain.Transaction{
					{From: addr, TransactionIndex: converter.FormatHexInt(1)},
					// rand tx should not match to given addr
					{From: genAddress(), TransactionIndex: converter.FormatHexInt(rand.Int())},
				}, error(nil))
				ethClient.On("GetBlockNumber", mock.Anything).Return(block, error(nil))

				// subscribe to rand address from txs in current block
				require.NoError(t, svc.Subscribe(ctx, addr))

				// set service cursor to block with found address tx
				blockNumberStore.SetCurrentBlock(block)

				// process current svc block transaction to handle matches immediately
				processed, err := svc.ProcessTransactions(ctx)
				require.NoError(t, err)
				require.True(t, processed)
			},
		},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			if testCase.preconditions != nil {
				testCase.preconditions(testCase.address)
			}
			tsx, err := svc.GetTransactions(ctx, testCase.address)
			require.ErrorIs(t, err, testCase.expectedErr)
			require.Equal(t, testCase.txs, tsx)
		})
	}
}
