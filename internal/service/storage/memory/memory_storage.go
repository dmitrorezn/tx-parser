package memory

import (
	"context"
	"sync"

	"github.com/dmitrorezn/tx-parser/internal/domain"
)

type Storage struct {
	subsMu sync.RWMutex
	subs   map[domain.Address]struct{}

	txMu sync.RWMutex
	txs  map[domain.Address][]domain.Transaction
}

func NewStorage() *Storage {
	return &Storage{
		subs: make(map[domain.Address]struct{}),
		txs:  make(map[domain.Address][]domain.Transaction),
	}
}

func (s *Storage) AddSubscriber(_ context.Context, addr domain.Address) error {
	s.subsMu.Lock()
	defer s.subsMu.Unlock()
	if _, ok := s.subs[addr]; ok {
		return domain.ErrAddressAlreadySubscribed
	}
	s.subs[addr] = struct{}{}

	return nil
}
func (s *Storage) ExistsSubscriber(_ context.Context, addr domain.Address) (bool, error) {
	s.subsMu.RLock()
	_, ok := s.subs[addr]
	s.subsMu.RUnlock()

	return ok, nil
}

func (s *Storage) GetSubscribers(_ context.Context) ([]domain.Address, error) {
	s.subsMu.RLock()
	addrs := make([]domain.Address, 0, len(s.subs))
	for addr := range s.subs {
		addrs = append(addrs, addr)
	}
	s.subsMu.RUnlock()

	return addrs, nil
}

func (s *Storage) AddTx(_ context.Context, addr domain.Address, tx domain.Transaction) error {
	s.txMu.Lock()
	defer s.txMu.Unlock()

	s.txs[addr] = append(s.txs[addr], tx)

	return nil
}

func (s *Storage) GetTransactions(_ context.Context, addr domain.Address) ([]domain.Transaction, error) {
	s.txMu.Lock()
	txs, ok := s.txs[addr]
	if !ok {
		s.txMu.Unlock()

		return nil, domain.ErrNoTransactions
	}
	delete(s.txs, addr)
	s.txMu.Unlock()

	transactions := make([]domain.Transaction, len(txs))
	copy(transactions, txs)

	return txs, nil
}
