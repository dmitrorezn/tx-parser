package ethrpcclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

func (c *JsonRpcClient) doRequest(ctx context.Context, method string, result any, params ...any) error {
	payload, err := json.Marshal(newRequest(method, params...))
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.addr, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	var response rpcResponse
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}
	if response.Error != nil {
		return response.Error
	}
	if err = json.Unmarshal(response.Result, result); err != nil {
		return err
	}

	return err
}
