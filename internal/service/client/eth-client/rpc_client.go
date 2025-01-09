package ethrpcclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"

	"github.com/dmitrorezn/tx-parser/internal/domain"
	"github.com/dmitrorezn/tx-parser/pkg/converter"
)

type JsonRpcClient struct {
	httpClient *http.Client
	addr       string
}

func NewJsonRpcClient(addr string) (*JsonRpcClient, error) {
	return &JsonRpcClient{
		addr:       addr,
		httpClient: http.DefaultClient,
	}, nil
}

var (
	ErrCallBlockchain = errors.New("err call blockchain")
)

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e rpcError) EthError() {}
func (e rpcError) ErrorCode() int {
	return e.Code
}
func (e rpcError) Error() string {
	return e.Message
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError
}

type Request struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
	Id      int    `json:"id"`
}

const (
	jsonRpcVersion = "2.0"
)

var id atomic.Int32

func newRequest(method string, params ...any) Request {
	return Request{
		Jsonrpc: jsonRpcVersion,
		Id:      int(id.Add(1)),
		Method:  method,
		Params:  params,
	}
}

func (c *JsonRpcClient) GetBlockNumber(ctx context.Context) (int, error) {
	var numberHex string
	err := c.doRequest(ctx, "eth_blockNumber", &numberHex)
	if err != nil {
		return 0, errors.Join(err, ErrCallBlockchain)
	}

	return converter.ParseHexInt(numberHex)
}

type numberAndFullTxFlag [2]any

func (c *JsonRpcClient) GetBlockTxsByNumber(ctx context.Context, number int) ([]domain.Transaction, error) {
	var (
		params = numberAndFullTxFlag{
			converter.FormatHexInt(number), //block number hex formatted
			true,                           // return full tx data
		}
		response struct {
			Txs []domain.Transaction `json:"transactions"`
		}
	)
	if err := c.doRequest(ctx, "eth_getBlockByNumber", &response, params[:]...); err != nil {
		return nil, errors.Join(err, ErrCallBlockchain)
	}

	return response.Txs, nil
}
