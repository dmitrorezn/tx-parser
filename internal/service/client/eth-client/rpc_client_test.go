package ethrpcclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	ethAddr = "https://ethereum-rpc.publicnode.com"
)

func TestGetBlock(t *testing.T) {
	ctx := context.Background()
	client, err := NewJsonRpcClient(ethAddr)
	require.NoError(t, err)

	number, err := client.GetBlockNumber(ctx)
	require.NoError(t, err)

	require.NotZero(t, number)
	t.Log(number)
}

func TestGetBlockTxs(t *testing.T) {
	ctx := context.Background()
	client, err := NewJsonRpcClient(ethAddr)
	require.NoError(t, err)

	number, err := client.GetBlockNumber(ctx)
	require.NoError(t, err)
	txs, err := client.GetBlockTxsByNumber(ctx, number)
	require.NoError(t, err)

	require.NotNil(t, txs)
	t.Log(len(txs))
}
