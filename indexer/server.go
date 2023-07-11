package indexer

import (
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
	listen     string
	db         database.KVStore
	logger     log.Logger
	entryPoint string
	handlers   map[string]func(s Rpc, req *rpc.JsonRpcMessage) *rpc.JsonRpcMessage
	compress   bool
}

func (s *Server) Db() database.KVStore {
	return s.db
}

func (s *Server) EntryPoint() string {
	return s.entryPoint
}

func (s *Server) Compressed() bool {
	return s.compress
}

func NewServer(cfg *Config, db database.KVStore) *Server {
	return &Server{
		listen:     cfg.RpcListen,
		db:         db,
		logger:     log.Module("server"),
		handlers:   map[string]func(s Rpc, req *rpc.JsonRpcMessage) *rpc.JsonRpcMessage{},
		entryPoint: cfg.EntryPoint,
		compress:   cfg.Compress,
	}
}

func (s *Server) Run() error {
	s.registerHandlers()
	http.HandleFunc("/", s.handler)
	http.HandleFunc("/status", s.status)
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
	//s.logger.Info(req.Method)

	msg := s.handlers[req.Method](s, req)
	resp, _ := json.Marshal(msg)

	s.writeJson(w, resp)
}

func (s *Server) registerHandlers() {
	s.handlers["eth_getLogsByUserOperation"] = eth_getLogsByUserOperation
	s.handlers["eth_getLogs"] = eth_getLogs
}

type Status struct {
	BlockNumber int64 `json:"block_number"`
	LatestBlock int64 `json:"latest_block"`
	CatchingUp  bool  `json:"catching_up"`
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	status := Status{
		BlockNumber: gBlockNumber,
		LatestBlock: gLatestBlock,
		CatchingUp:  !(gBlockNumber >= (gLatestBlock - 5)),
	}
	data, _ := json.Marshal(status)
	w.Write(data)
}
