package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	addr   string
	client *http.Client
}

// New - creates new parser client
// todo add client options, timeouts, idle timeout and round tripper for traces and baggage propagation
func New(addr string) *Client {
	return &Client{
		addr:   addr,
		client: http.DefaultClient,
	}
}

type Clienter interface {
	// GetCurrentBlock - last parsed block
	GetCurrentBlock(ctx context.Context) (int, error)
	// Subscribe - add address to observer
	Subscribe(ctx context.Context, address string) error
	// GetTransactions -  list of inbound or outbound transactions for an address
	GetTransactions(ctx context.Context, address string) ([]Transaction, error)
}

var _ Clienter = (*Client)(nil)

type blockResponse struct {
	CurrentBlockHeight int `json:"currentBlockHeight"`
}

func (c *Client) GetCurrentBlock(ctx context.Context) (int, error) {
	body, err := c.doGET(ctx, "getCurrentBlock")
	if err != nil {
		return 0, err
	}
	defer func() {
		err = errors.Join(err, body.Close())
	}()
	var resp blockResponse
	if err = json.NewDecoder(body).Decode(&resp); err != nil {
		return 0, err
	}

	return resp.CurrentBlockHeight, err
}

type subscribeRequest struct {
	Address string `json:"address"`
}

func (c *Client) Subscribe(ctx context.Context, address string) error {
	body, err := c.doPOST(ctx, "subscribe", subscribeRequest{Address: address})
	if err != nil {
		return err
	}
	if err = body.Close(); err != nil {
		return err
	}

	return nil
}

type Transaction struct {
	BlockHash        string `json:"blockHash"`
	BlockNumber      string `json:"blockNumber"`
	From             string `json:"from"`
	Gas              string `json:"gas"`
	GasPrice         string `json:"gasPrice"`
	Hash             string `json:"hash"`
	Input            string `json:"input"`
	Nonce            string `json:"nonce"`
	To               string `json:"to"`
	TransactionIndex string `json:"transactionIndex"`
	Value            string `json:"value"`
	V                string `json:"v"`
	R                string `json:"r"`
	S                string `json:"s"`
}

func (c *Client) GetTransactions(ctx context.Context, address string) ([]Transaction, error) {
	path, err := url.JoinPath("getTransactions", address)
	if err != nil {
		return nil, err
	}
	body, err := c.doGET(ctx, path)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, body.Close())
	}()
	var txs []Transaction
	if err = json.NewDecoder(body).Decode(&txs); err != nil {
		return nil, err
	}

	return txs, nil
}

func (c *Client) doGET(ctx context.Context, path string) (io.ReadCloser, error) {
	requestURL, err := url.JoinPath(c.addr, path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, handleError(resp)
	}

	return resp.Body, nil
}
func handleError(resp *http.Response) (err error) {
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	p, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return NewError(
		resp.StatusCode,
		string(p),
	)
}
func (c *Client) doPOST(ctx context.Context, path string, body any) (io.ReadCloser, error) {
	requestURL, err := url.JoinPath(c.addr, path)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	if err = json.NewEncoder(buf).Encode(body); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, buf)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, handleError(resp)
	}

	return resp.Body, nil
}
