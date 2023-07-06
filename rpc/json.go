package rpc

import "encoding/json"

var ID0 []byte = nil

type JsonRpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   *JsonrpcError   `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

type JsonrpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func ParseJsonRpcMessage(data []byte) *JsonRpcMessage {
	var ret *JsonRpcMessage
	json.Unmarshal(data, &ret)
	return ret
}

func NewJsonRpcMessage(id []byte) *JsonRpcMessage {
	return &JsonRpcMessage{ID: id, Version: "2.0"}
}

func NewJsonRpcMessageWithError(id []byte, code int, err string) *JsonRpcMessage {
	return &JsonRpcMessage{
		ID:      id,
		Version: "2.0",
		Error:   &JsonrpcError{Code: code, Message: "indexer: " + err},
	}
}
