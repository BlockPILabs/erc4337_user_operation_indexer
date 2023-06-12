package rpc

import "encoding/json"

var ID0 = json.RawMessage{0, 0, 0, 0}

type JsonrpcMessage struct {
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

func ParseJsonrpcMessage(data []byte) *JsonrpcMessage {
	var ret *JsonrpcMessage
	json.Unmarshal(data, &ret)
	return ret
}

func NewJsonrpcMessage(id []byte) *JsonrpcMessage {
	return &JsonrpcMessage{ID: id, Version: "2.0"}
}

func NewJsonrpcMessageWithError(id []byte, err string) *JsonrpcMessage {
	return &JsonrpcMessage{
		ID:      id,
		Version: "2.0",
		Error:   &JsonrpcError{Code: -32000, Message: err},
	}
}
