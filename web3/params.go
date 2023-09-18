package web3

import (
	"encoding/json"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/rpc"
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
		Address   *json.RawMessage
		Topics    *json.RawMessage
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
	param := params[0]

	addrArr, _ := parseToStringOrArray(*param.Address)
	if len(addrArr) == 0 {
		return nil, rpc.NewJsonRpcMessageWithError(req.ID, -32602, "address wanted")
	}
	if len(addrArr) != 1 {
		return nil, rpc.NewJsonRpcMessageWithError(req.ID, -32602, "invalid addresses, want at most 1")
	}

	topicsParam := *param.Topics

	if !IsJsonArray(topicsParam) {
		return nil, rpc.NewJsonRpcMessageWithError(req.ID, -32000, "topics wanted")
	}

	var topics []json.RawMessage
	json.Unmarshal(topicsParam, &topics)
	if len(topics) < 2 {
		return nil, rpc.NewJsonRpcMessageWithError(req.ID, -32000, "too less topics, want at least 2")
	}

	var topicsArr []string
	topics1, _ := parseToStringOrArray(topics[0])
	if len(topics1) == 0 {
		return nil, rpc.NewJsonRpcMessageWithError(req.ID, -32000, "too less topics")
	}
	if len(topics1) != 1 {
		return nil, rpc.NewJsonRpcMessageWithError(req.ID, -32000, "too many topics")
	}
	topicsArr = append(topicsArr, topics1[0])

	topics2, _ := parseToStringOrArray(topics[1])
	if len(topics2) == 0 {
		return nil, rpc.NewJsonRpcMessageWithError(req.ID, -32000, "too less topics")
	}
	if len(topics2) != 1 {
		return nil, rpc.NewJsonRpcMessageWithError(req.ID, -32000, "too many topics")
	}
	topicsArr = append(topicsArr, topics2[0])

	return &EthGetLogsRequestParams{
		Address:   addrArr[0],
		Topics:    topicsArr,
		FromBlock: param.FromBlock,
		ToBlock:   param.ToBlock,
	}, nil
}

func parseToStringOrArray(data []byte) ([]string, bool) {
	var result []string
	isArray := IsJsonArray(data)
	if isArray {
		json.Unmarshal(data, &result)
	} else {
		var str string
		json.Unmarshal(data, &str)
		result = append(result, str)
	}
	return result, isArray
}
