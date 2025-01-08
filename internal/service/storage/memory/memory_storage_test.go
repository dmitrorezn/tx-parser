package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"slices"
	"testing"

	"github.com/dmitrorezn/tx-parser/internal/domain"
	"github.com/stretchr/testify/require"
)

func genAddress() string {
	var addr [20]byte
	_, _ = rand.Read(addr[:])

	return hex.EncodeToString(addr[:])
}

func TestExistsSubscriber(t *testing.T) {
	storage := NewStorage()

	ctx := context.Background()

	addr := domain.Address(genAddress())
	err := storage.AddSubscriber(ctx, addr)
	require.NoError(t, err)

	err = storage.AddSubscriber(ctx, addr)
	require.Error(t, domain.ErrAddressAlreadySubscribed)

	ex, err := storage.ExistsSubscriber(ctx, addr)
	require.NoError(t, err)

	require.True(t, ex)
}

func TestGetTransactions(t *testing.T) {
	storage := NewStorage()

	ctx := context.Background()

	addr := domain.Address(genAddress())
	err := storage.AddTx(ctx, addr, domain.Transaction{
		From: addr,
	})
	require.NoError(t, err)

	txs, err := storage.GetTransactions(ctx, addr)
	require.NoError(t, err)

	require.Len(t, txs, 1)
	require.True(t, slices.ContainsFunc(txs, func(transaction domain.Transaction) bool {
		return transaction.From == addr
	}))

	txs, err = storage.GetTransactions(ctx, addr)
	require.Error(t, domain.ErrNoTransactions)
}
