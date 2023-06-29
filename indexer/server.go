package indexer

import (
	"bytes"
	"encoding/json"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/log"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/rpc"
	"io"
	"net/http"
)

var (
	invalidRequest = []byte("invalid request")
)

type Server struct {
	listen string
	db     database.KVStore
	logger log.Logger

	handlers map[string]func(req *rpc.JsonrpcMessage) *rpc.JsonrpcMessage
}

func NewServer(cfg *Config, db database.KVStore) *Server {
	return &Server{
		listen:   cfg.RpcListen,
		db:       db,
		logger:   log.Module("server"),
		handlers: map[string]func(req *rpc.JsonrpcMessage) *rpc.JsonrpcMessage{},
	}
}

func (s *Server) Run() error {
	s.registerHandlers()
	http.HandleFunc("/", s.handler)
	return http.ListenAndServe(s.listen, nil)
}

func (s *Server) writeJson(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (s *Server) validRequest(w http.ResponseWriter, r *http.Request) (*rpc.JsonrpcMessage, bool) {
	if r.Method != "POST" {
		resp, _ := json.Marshal(rpc.NewJsonrpcMessageWithError(rpc.ID0, string(invalidRequest)))
		s.writeJson(w, resp)
		return nil, false
	}

	defer r.Body.Close()
	reqBody, _ := io.ReadAll(r.Body)
	req := rpc.ParseJsonrpcMessage(reqBody)

	_, ok := s.handlers[req.Method]
	if !ok {
		resp, _ := json.Marshal(rpc.NewJsonrpcMessageWithError(req.ID, string(invalidRequest)))
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
}

func (s *Server) eth_getLogsByUserOperation(req *rpc.JsonrpcMessage) *rpc.JsonrpcMessage {
	var params []string
	err := json.Unmarshal(req.Params, &params)
	if err != nil || len(params) == 0 {
		return rpc.NewJsonrpcMessageWithError(req.ID, string(invalidRequest))
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

	resp := rpc.NewJsonrpcMessage(req.ID)
	resp.Result = result
	return resp
}
