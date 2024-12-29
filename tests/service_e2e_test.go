package tests

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dmitrorezn/tx-parser/client"
	"github.com/dmitrorezn/tx-parser/internal/domain"
	"github.com/dmitrorezn/tx-parser/internal/service"
	ethrpcclient "github.com/dmitrorezn/tx-parser/internal/service/client/eth-client"
	httpport "github.com/dmitrorezn/tx-parser/internal/service/ports/http"
	"github.com/dmitrorezn/tx-parser/internal/service/storage/memory"
	"github.com/dmitrorezn/tx-parser/pkg/logger"
	"github.com/stretchr/testify/require"
)

func setupServiceClient(t *testing.T, currentBlock int) (*service.Service, service.BlocksStorage, client.Clienter) {
	ethClient, err := ethrpcclient.NewJsonRpcClient("https://ethereum-rpc.publicnode.com")
	require.NoError(t, err)

	var (
		loggr            = logger.NewAttrLogger(logger.NewLogger())
		storage          = memory.NewStorage()
		blockNumberStore = memory.NewBlockNumberStorage()
		cfg              = service.NewConfig(100 * time.Millisecond)
		svc              = service.NewService(ethClient, blockNumberStore, storage, loggr, cfg)
		handler          = httpport.NewHandler(svc)
		srv              = httptest.NewServer(handler)
	)
	blockNumberStore.SetCurrentBlock(currentBlock)

	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})

	return svc, blockNumberStore, client.New(srv.URL)
}

func TestGetCurrentBlock(t *testing.T) {
	ctx := context.Background()
	ethClient, err := ethrpcclient.NewJsonRpcClient("https://ethereum-rpc.publicnode.com")
	require.NoError(t, err)

	headBlock, _ := ethClient.GetBlockNumber(ctx)

	prevBlock := headBlock - 1
	svc, _, svcClient := setupServiceClient(t, prevBlock)

	tests := map[string]struct {
		expectedBlock func() int
		preconditions func()
	}{
		"1. Success: get initial block head (not started svc)": {
			expectedBlock: func() int {
				return prevBlock
			},
		},
		"2. Success: current block equal to blockchain head": {
			expectedBlock: func() int {
				n, _ := ethClient.GetBlockNumber(ctx)
				return n
			},
			preconditions: func() {
				// add any address to processor to start processing blocks
				require.NoError(t, svc.Subscribe(ctx, domain.Address(genAddress())))

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
			block, err := svcClient.GetCurrentBlock(ctx)
			require.NoError(t, err)

			require.Equal(t, testCase.expectedBlock(), block)
		})
	}
}

func genAddress() string {
	var addr [20]byte
	_, _ = rand.Read(addr[:])

	return "0x" + hex.EncodeToString(addr[:])
}

func TestSubscribe(t *testing.T) {
	ctx := context.Background()
	ethClient, err := ethrpcclient.NewJsonRpcClient("https://ethereum-rpc.publicnode.com")
	require.NoError(t, err)

	headBlock, _ := ethClient.GetBlockNumber(ctx)

	_, _, svcClient := setupServiceClient(t, headBlock)

	var (
		randAddr = genAddress()
	)
	tests := map[string]struct {
		address       string
		expectedErr   error
		preconditions func()
	}{
		"1. Err address not valid": {
			expectedErr: client.NewError(
				http.StatusBadRequest, `{"error":"invalid address","msg":"invalid address"}`+"\n",
			),
		},
		"2. Err Already Subscribed": {
			address: randAddr,
			expectedErr: client.NewError(
				http.StatusConflict, `{"error":"address already subscribed","msg":"address already subscribed"}`+"\n",
			),
			preconditions: func() {
				require.NoError(t, svcClient.Subscribe(ctx, randAddr))
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
			err = svcClient.Subscribe(ctx, testCase.address)
			require.ErrorIs(t, err, testCase.expectedErr)
		})
	}
}

func TestGetTransactions(t *testing.T) {
	ctx := context.Background()
	ethClient, err := ethrpcclient.NewJsonRpcClient("https://ethereum-rpc.publicnode.com")
	require.NoError(t, err)

	headBlock, _ := ethClient.GetBlockNumber(ctx)

	svc, blockNumberStore, svcClient := setupServiceClient(t, headBlock)

	var (
		randAddr = genAddress()
	)

	tests := map[string]struct {
		address       string
		txs           []client.Transaction
		expectedErr   error
		preconditions func() (addr string, txs []client.Transaction)
	}{
		"1. Err address not subscribed": {
			txs:     nil,
			address: genAddress(),
			expectedErr: client.NewError(
				http.StatusNotFound, `{"error":"address not subscribed","msg":"not found subscriber"}`+"\n",
			),
		},
		"2. Err No transaction": {
			txs:     nil,
			address: randAddr,
			expectedErr: client.NewError(
				http.StatusNotFound, `{"error":"no transactions","msg":"not found transactions"}`+"\n",
			),
			preconditions: func() (string, []client.Transaction) {
				require.NoError(t, svcClient.Subscribe(ctx, randAddr))

				return randAddr, nil
			},
		},
		"2. Success transaction": {
			txs:         nil,
			address:     "",
			expectedErr: nil,
			preconditions: func() (string, []client.Transaction) {
				// define current height of blockchain
				block, err := ethClient.GetBlockNumber(ctx)
				require.NoError(t, err)
				// get current block txs
				txs, err := ethClient.GetBlockTxsByNumber(ctx, block)
				require.NoError(t, err)
				require.True(t, len(txs) > 0)
				var (
					tx   = txs[rand.Intn(len(txs)-1)]
					addr = string(tx.From)
				)
				// subscribe to rand address from txs in current block
				require.NoError(t, svcClient.Subscribe(ctx, addr))

				// set service cursor to block with found address tx
				blockNumberStore.SetCurrentBlock(block)

				// process current svc block transaction to handle matches immediately
				processed, err := svc.ProcessTransactions(ctx)
				require.NoError(t, err)
				require.True(t, processed)

				var transacts [1]client.Transaction
				require.NoError(t, marshalUnmarshal(tx, &transacts[0]))

				return addr, transacts[:]
			},
		},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			if testCase.preconditions != nil {
				addr, txs := testCase.preconditions()
				if addr != "" {
					testCase.address = addr
				}
				if txs != nil {
					testCase.txs = txs
				}
			}
			tsx, err := svcClient.GetTransactions(ctx, testCase.address)
			require.ErrorIs(t, err, testCase.expectedErr)
			require.Equal(t, testCase.txs, tsx)
		})
	}
}

func marshalUnmarshal(src any, dst any) error {
	p, err := json.Marshal(src)
	if err != nil {
		return err
	}

	return json.Unmarshal(p, dst)
}
