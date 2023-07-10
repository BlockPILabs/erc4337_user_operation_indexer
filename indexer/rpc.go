package indexer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/rpc"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/web3"
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
	param, errMsg := web3.ParseEthGetLogsRequestParams(req)
	if errMsg != nil {
		return errMsg
	}

	entrypoint := s.EntryPoint()
	if strings.ToLower(param.Address) != entrypoint {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32000, "address mismatch entrypoint "+entrypoint)
	}

	descriptor := strings.ToLower(fmt.Sprintf("%v", param.Topics[0]))
	if descriptor != LogDescriptor {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32000, "invalid Log descriptor: "+descriptor)
	}

	opHash := fmt.Sprintf("%v", param.Topics[1])
	data, _ := s.Db().Get(DbKeyUserOp(opHash))

	if len(data) > 0 {
		info := struct {
			Topics      []string
			BlockNumber string
		}{}
		json.Unmarshal(data, &info)

		for i := 0; i < len(param.Topics) && i < len(info.Topics); i++ {
			if param.Topics[i] != info.Topics[i] {
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
