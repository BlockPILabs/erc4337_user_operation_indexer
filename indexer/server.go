package indexer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/log"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/rpc"
	"io"
	"net/http"
	"strings"
)

var (
	invalidRequest = []byte("invalid request")
)

type Server struct {
	listen     string
	db         database.KVStore
	logger     log.Logger
	entryPoint string
	handlers   map[string]func(req *rpc.JsonRpcMessage) *rpc.JsonRpcMessage
}

func NewServer(cfg *Config, db database.KVStore) *Server {
	return &Server{
		listen:     cfg.RpcListen,
		db:         db,
		logger:     log.Module("server"),
		handlers:   map[string]func(req *rpc.JsonRpcMessage) *rpc.JsonRpcMessage{},
		entryPoint: cfg.EntryPoint,
	}
}

func (s *Server) Run() error {
	s.registerHandlers()
	http.HandleFunc("/", s.handler)
	s.logger.Info("aip server listen: " + s.listen)
	err := http.ListenAndServe(s.listen, nil)
	if err != nil {
		s.logger.Error("aip server listen failed: " + s.listen)
	}
	return err
}

func (s *Server) writeJson(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (s *Server) validRequest(w http.ResponseWriter, r *http.Request) (*rpc.JsonRpcMessage, bool) {
	if r.Method != "POST" {
		resp, _ := json.Marshal(rpc.NewJsonRpcMessageWithError(rpc.ID0, -32000, string(invalidRequest)))
		s.writeJson(w, resp)
		return nil, false
	}

	defer r.Body.Close()
	reqBody, _ := io.ReadAll(r.Body)
	req := rpc.ParseJsonRpcMessage(reqBody)
	if req == nil {
		resp, _ := json.Marshal(rpc.NewJsonRpcMessageWithError(rpc.ID0, -32000, string(invalidRequest)))
		s.writeJson(w, resp)
		return nil, false
	}

	_, ok := s.handlers[req.Method]
	if !ok {
		resp, _ := json.Marshal(rpc.NewJsonRpcMessageWithError(req.ID, -32000, string(invalidRequest)))
		s.writeJson(w, resp)
		return nil, false
	}

	return req, true
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	req, ok := s.validRequest(w, r)
	if !ok {
		return
	}
	s.logger.Info(req.Method)

	msg := s.handlers[req.Method](req)
	resp, _ := json.Marshal(msg)

	s.writeJson(w, resp)
}

func (s *Server) registerHandlers() {
	s.handlers["eth_getLogsByUserOperation"] = s.eth_getLogsByUserOperation
	s.handlers["eth_getLogs"] = s.eth_getLogs
}

func (s *Server) eth_getLogsByUserOperation(req *rpc.JsonRpcMessage) *rpc.JsonRpcMessage {
	var params []string
	err := json.Unmarshal(req.Params, &params)
	if err != nil || len(params) == 0 {
		return rpc.NewJsonRpcMessageWithError(req.ID, -32000, string(invalidRequest))
	}

	var logs = make([][]byte, len(params))
	for i, hash := range params {
		data, _ := s.db.Get(DbKeyUserOp(hash))
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

func (s *Server) eth_getLogs(req *rpc.JsonRpcMessage) *rpc.JsonRpcMessage {
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
	case []string:
		arr := v.([]string)
		if len(arr) != 1 {
			return rpc.NewJsonRpcMessageWithError(req.ID, -32000, "most 1 address")
		}
		address = arr[0]
	}
	address = strings.ToLower(address)
	if address != s.entryPoint {
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
	data, _ := s.db.Get(DbKeyUserOp(opHash))

	result := bytes.Join([][]byte{[]byte("["), data, []byte("]")}, []byte(""))

	resp := rpc.NewJsonRpcMessage(req.ID)
	resp.Result = result
	return resp
}
