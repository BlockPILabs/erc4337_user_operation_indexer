package web3

import (
	"encoding/json"
	"fmt"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/utils"
	"sync/atomic"
)

var reqFmt = "{\"jsonrpc\":\"2.0\",\"id\":%d,\"method\":\"%s\",\"params\":%s}"

type Client struct {
	Provider string
	Id       int64
}

func (c *Client) Call(method string, params string, out any) error {
	id := atomic.AddInt64(&c.Id, 1)
	req := fmt.Sprintf(reqFmt, id, method, params)
	resp, err := utils.HttpPost(c.Provider, []byte(req), "application/json")
	if err != nil {
		return err
	}

	err = json.Unmarshal(resp, out)

	return err
}
