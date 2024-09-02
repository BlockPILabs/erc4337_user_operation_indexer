package indexer

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/BlockPILabs/erc4337_user_operation_indexer/database"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/rpc"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/web3"
	"github.com/golang/snappy"
	"golang.org/x/exp/slices"
)

type Rpc interface {
	Db() database.KVStore
	EntryPoints() []string
	Compressed() bool
}

func eth_getLogsByUserOperation(s Rpc, chain string, req *rpc.JsonRpcMessage) *rpc.JsonRpcMessage {
	var params []string
	err := json.Unmarshal(req.Params, &params)
	if err != nil || len(params) == 0 {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32000, string(invalidRequest))
	}

	var logs = make([][]byte, len(params))
	for i, hash := range params {
		data, _ := s.Db().Get(DbKeyUserOp(chain, hash))
		if data != nil && s.Compressed() {
			decoded, err := snappy.Decode(nil, data)
			if err == nil {
				data = decoded
			}
		}

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

func eth_getLogs(s Rpc, chain string, req *rpc.JsonRpcMessage) *rpc.JsonRpcMessage {
	param, errMsg := web3.ParseEthGetLogsRequestParams(req)
	if errMsg != nil {
		return errMsg
	}

	if !slices.Contains(s.EntryPoints(), strings.ToLower(param.Address)) {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32000, "address mismatch entrypoint "+param.Address)
	}

	descriptor := strings.ToLower(param.Topics[0])
	if descriptor != LogDescriptor {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32000, "invalid Log descriptor: "+descriptor)
	}

	opHash := strings.ToLower(param.Topics[1])
	data, _ := s.Db().Get(DbKeyUserOp(chain, opHash))

	if data != nil && s.Compressed() {
		decoded, err := snappy.Decode(nil, data)
		if err == nil {
			data = decoded
		}
	}

	if len(data) > 0 {
		info := struct {
			Address     string
			Topics      []string
			BlockNumber string
		}{}
		json.Unmarshal(data, &info)
		if strings.ToLower(param.Address) != strings.ToLower(info.Address) {
			data = []byte("")
		} else {
			for i := 0; i < len(param.Topics) && i < len(info.Topics); i++ {
				if strings.ToLower(param.Topics[i]) != strings.ToLower(info.Topics[i]) {
					data = []byte("")
					break
				}
			}
		}
	}

	result := bytes.Join([][]byte{[]byte("["), data, []byte("]")}, []byte(""))

	resp := rpc.NewJsonRpcMessage(req.ID)
	resp.Result = result
	return resp
}
