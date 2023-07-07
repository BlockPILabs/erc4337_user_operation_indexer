package indexer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/rpc"
	"strings"
)

type Rpc interface {
	Db() database.KVStore
	EntryPoint() string
}

func eth_getLogsByUserOperation(s Rpc, req *rpc.JsonRpcMessage) *rpc.JsonRpcMessage {
	var params []string
	err := json.Unmarshal(req.Params, &params)
	if err != nil || len(params) == 0 {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32000, string(invalidRequest))
	}

	var logs = make([][]byte, len(params))
	for i, hash := range params {
		data, _ := s.Db().Get(DbKeyUserOp(hash))
		if data == nil {
			data = []byte("null")
		}
		logs[i] = data
	}

	result := bytes.Join([][]byte{[]byte("["), bytes.Join(logs, []byte(",")), []byte("]")}, []byte(""))

	resp := rpc.NewJsonRpcMessage(req.ID)
	resp.Result = result
	return resp
}

func eth_getLogs(s Rpc, req *rpc.JsonRpcMessage) *rpc.JsonRpcMessage {
	var params []map[string]any
	err := json.Unmarshal(req.Params, &params)
	if err != nil {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32000, "invalid json")
	}

	if len(params) != 1 {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32602, "too many arguments, want at most 1")
	}

	address := ""
	v, ok := params[0]["address"]
	if !ok {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32602, "address wanted")
	}
	switch v.(type) {
	case string:
		address = v.(string)
	case []interface{}:
		arr := v.([]interface{})
		if len(arr) != 1 {
			return rpc.NewJsonRpcMessageWithError(req.ID, -32000, "most 1 address")
		}
		address = fmt.Sprintf("%v", arr[0])
	}
	address = strings.ToLower(address)
	if address != s.EntryPoint() {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32000, "address wanted")
	}

	v, ok = params[0]["topics"]
	if !ok {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32000, "topics wanted")
	}

	var topics []any
	switch v.(type) {
	case []any:
		topics = v.([]any)
	case [][]any:
		arr := v.([][]any)
		if len(arr) != 1 {
			return rpc.NewJsonRpcMessageWithError(req.ID, -32000, "most 1 group topics")
		}
		topics = arr[0]
	}

	if len(topics) < 2 {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32000, "require at least 2 topic descriptors")
	}

	descriptor := strings.ToLower(fmt.Sprintf("%v", topics[0]))
	if descriptor != LogDescriptor {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32000, "invalid Log descriptor: "+descriptor)
	}

	opHash := fmt.Sprintf("%v", topics[1])
	data, _ := s.Db().Get(DbKeyUserOp(opHash))

	result := bytes.Join([][]byte{[]byte("["), data, []byte("]")}, []byte(""))

	resp := rpc.NewJsonRpcMessage(req.ID)
	resp.Result = result
	return resp
}
