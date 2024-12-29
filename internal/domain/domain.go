package domain

import (
	"errors"
	"strings"
)

type Address string

const (
	addrPrefix = "0x"
	addrLen    = len(addrPrefix) + 40
)

func (a Address) Valid() bool {
	return len(a) == addrLen && strings.Contains(string(a), addrPrefix)
}

type Transaction struct {
	BlockHash        string  `json:"blockHash"`
	BlockNumber      string  `json:"blockNumber"`
	From             Address `json:"from"`
	Gas              string  `json:"gas"`
	GasPrice         string  `json:"gasPrice"`
	Hash             string  `json:"hash"`
	Input            string  `json:"input"`
	Nonce            string  `json:"nonce"`
	To               Address `json:"to"`
	TransactionIndex string  `json:"transactionIndex"`
	Value            string  `json:"value"`
	V                string  `json:"v"`
	R                string  `json:"r"`
	S                string  `json:"s"`
}

func (tx Transaction) BelongsToAddr(addr Address) bool {
	return tx.From == addr || tx.To == addr
}

var (
	ErrAddressNotSubscribed     = errors.New("address not subscribed")
	ErrAddressAlreadySubscribed = errors.New("address already subscribed")
	ErrNoTransactions           = errors.New("no transactions")
	ErrInvalidAddress           = errors.New("invalid address")
)
