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

func isArray(raw []byte) bool {
	for _, c := range raw {
		// skip insignificant whitespace (http://www.ietf.org/rfc/rfc4627.txt)
		if c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d {
			continue
		}
		return c == '['
	}
	return false
}

func eth_getLogs(s Rpc, req *rpc.JsonRpcMessage) *rpc.JsonRpcMessage {
	var params []struct {
		Address   json.RawMessage
		Topics    []json.RawMessage
		FromBlock string
		ToBlock   string
	}

	err := json.Unmarshal(req.Params, &params)
	if err != nil {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32000, "invalid json")
	}
	if len(params) != 1 {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32602, "too many arguments, want at most 1")
	}

	address := ""
	param := params[0]
	if isArray(param.Address) {
		var addresses []string
		json.Unmarshal(param.Address, &addresses)
		if len(addresses) > 1 {
			return rpc.NewJsonRpcMessageWithError(req.ID, -32602, "too many addresses, want at most 1")
		}
		if len(addresses) == 1 {
			address = addresses[0]
		}
	} else {
		json.Unmarshal(param.Address, &address)
	}

	if len(address) == 0 {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32602, "address wanted")
	}

	address = strings.ToLower(address)
	entrypoint := s.EntryPoint()
	if address != entrypoint {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32000, "address mismatch entrypoint "+entrypoint)
	}

	if len(param.Topics) == 0 {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32000, "topics wanted")
	}

	var topics []string
	if isArray(param.Topics[0]) {
		if len(param.Topics) > 1 {
			return rpc.NewJsonRpcMessageWithError(req.ID, -32000, "too many topics, want at most 1")
		}
		json.Unmarshal(param.Topics[0], &topics)
	} else {
		for _, topic := range param.Topics {
			var topicStr string
			json.Unmarshal(topic, &topicStr)
			topics = append(topics, topicStr)
		}
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

	if len(data) > 0 {
		info := struct {
			Topics      []string
			BlockNumber string
		}{}
		json.Unmarshal(data, &info)

		for i := 0; i < len(topics) && i < len(info.Topics); i++ {
			if topics[i] != info.Topics[i] {
				data = []byte("")
				break
			}
		}
	}

	result := bytes.Join([][]byte{[]byte("["), data, []byte("]")}, []byte(""))

	resp := rpc.NewJsonRpcMessage(req.ID)
	resp.Result = result
	return resp
}
