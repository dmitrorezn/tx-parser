package ethrpcclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/dmitrorezn/tx-parser/internal/domain"
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

const (
	hexPrefix          = "0x"
	hexBase            = 16
	blockNumberBitSize = 64
)

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

var ErrInvalidBlockNumberHex = errors.New("invalid block number hex")

func ParseHexInt(str string) (int, error) {
	number, err := strconv.ParseInt(str[len(hexPrefix):], hexBase, blockNumberBitSize)
	if err != nil {
		return 0, err
	}

	return int(number), err
}

func (c *JsonRpcClient) GetBlockNumber(ctx context.Context) (int, error) {
	var numberHex string
	err := c.doRequest(ctx, "eth_blockNumber", &numberHex)
	if err != nil {
		return 0, errors.Join(err, ErrCallBlockchain)
	}
	if len(numberHex) < len(hexPrefix) {
		return 0, ErrInvalidBlockNumberHex
	}

	return ParseHexInt(numberHex)
}

func FormatIntToHex(number int) string {
	return hexPrefix + strconv.FormatInt(int64(number), hexBase)
}

func (c *JsonRpcClient) GetBlockTxsByNumber(ctx context.Context, number int) ([]domain.Transaction, error) {
	type numberAndFlag [2]any
	var (
		params = numberAndFlag{
			FormatIntToHex(number), //block number hex formated
			true,                   // return full tx data
		}
		response struct {
			Txs []domain.Transaction `json:"transactions"`
		}
	)
	err := c.doRequest(ctx, "eth_getBlockByNumber", &response, params[:]...)
	if err != nil {
		return nil, errors.Join(err, ErrCallBlockchain)
	}

	return response.Txs, nil
}
