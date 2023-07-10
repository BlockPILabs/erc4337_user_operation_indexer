package web3

import (
	"encoding/json"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/rpc"
	"strings"
)

func IsJsonArray(raw []byte) bool {
	for _, c := range raw {
		// skip insignificant whitespace (http://www.ietf.org/rfc/rfc4627.txt)
		if c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d {
			continue
		}
		return c == '['
	}
	return false
}

type EthGetLogsRequestParams struct {
	Address   string
	Topics    []string
	FromBlock string
	ToBlock   string
}

func ParseEthGetLogsRequestParams(req *rpc.JsonRpcMessage) (*EthGetLogsRequestParams, *rpc.JsonRpcMessage) {
	var params []struct {
		Address   json.RawMessage
		Topics    []json.RawMessage
		FromBlock string
		ToBlock   string
	}

	err := json.Unmarshal(req.Params, &params)
	if err != nil {
		return nil, rpc.NewJsonRpcMessageWithError(req.ID, -32000, "invalid json")
	}
	if len(params) != 1 {
		return nil, rpc.NewJsonRpcMessageWithError(req.ID, -32602, "too many arguments, want at most 1")
	}

	address := ""
	param := params[0]
	if IsJsonArray(param.Address) {
		var addresses []string
		json.Unmarshal(param.Address, &addresses)
		if len(addresses) > 1 {
			return nil, rpc.NewJsonRpcMessageWithError(req.ID, -32602, "too many addresses, want at most 1")
		}
		if len(addresses) == 1 {
			address = addresses[0]
		}
	} else {
		json.Unmarshal(param.Address, &address)
	}

	if len(address) == 0 {
		return nil, rpc.NewJsonRpcMessageWithError(req.ID, -32602, "address wanted")
	}

	address = strings.ToLower(address)

	if len(param.Topics) == 0 {
		return nil, rpc.NewJsonRpcMessageWithError(req.ID, -32000, "topics wanted")
	}

	var topics []string
	if IsJsonArray(param.Topics[0]) {
		if len(param.Topics) > 1 {
			return nil, rpc.NewJsonRpcMessageWithError(req.ID, -32000, "too many topics, want at most 1")
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
		return nil, rpc.NewJsonRpcMessageWithError(req.ID, -32000, "require at least 2 topic descriptors")
	}

	return &EthGetLogsRequestParams{
		Address:   address,
		Topics:    topics,
		FromBlock: param.FromBlock,
		ToBlock:   param.ToBlock,
	}, nil
}
