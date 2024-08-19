package indexer

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/BlockPILabs/erc4337_user_operation_indexer/database"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/log"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/rpc"
	"github.com/spf13/cast"
)

const HeaderChain = "x-bpi-chain"

var (
	invalidRequest = []byte("invalid request")
	invalidChain   = []byte("invalid chain")
)

type handlerFunc func(s Rpc, chain string, req *rpc.JsonRpcMessage) *rpc.JsonRpcMessage

type Server struct {
	listen      string
	db          database.KVStore
	logger      log.Logger
	entryPoints []string
	handlers    map[string]handlerFunc
	compress    bool
	readonly    bool
	chains      []string
}

func (s *Server) Db() database.KVStore {
	return s.db
}

func (s *Server) EntryPoints() []string {
	return s.entryPoints
}

func (s *Server) Compressed() bool {
	return s.compress
}

func NewServer(cfg *Config, db database.KVStore) *Server {
	s := &Server{
		listen:      cfg.Listen,
		db:          db,
		logger:      log.Module("server"),
		handlers:    map[string]handlerFunc{},
		entryPoints: cfg.EntryPoints,
		compress:    cfg.Compress,
		readonly:    cfg.Readonly,
	}
	for _, chain := range cfg.Chains {
		s.chains = append(s.chains, chain.Chain)
	}
	return s
}

func (s *Server) Run() error {
	s.registerHandlers()
	http.HandleFunc("/", s.handler)
	http.HandleFunc("/status", s.status)
	s.logger.Info("api server listen: " + s.listen)
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

func (s *Server) validRequest(w http.ResponseWriter, r *http.Request) (*rpc.JsonRpcMessage, string, bool) {
	if r.Method != "POST" {
		resp, _ := json.Marshal(rpc.NewJsonRpcMessageWithError(rpc.ID0, -32000, string(invalidRequest)))
		s.writeJson(w, resp)
		return nil, "", false
	}

	chain := r.Header.Get(HeaderChain)
	if len(chain) == 0 {
		resp, _ := json.Marshal(rpc.NewJsonRpcMessageWithError(rpc.ID0, -32000, string(invalidChain)))
		s.writeJson(w, resp)
		return nil, chain, false
	}

	reqBody, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	req := rpc.ParseJsonRpcMessage(reqBody)
	if req == nil {
		resp, _ := json.Marshal(rpc.NewJsonRpcMessageWithError(rpc.ID0, -32000, string(invalidRequest)))
		s.writeJson(w, resp)
		return nil, chain, false
	}

	_, ok := s.handlers[req.Method]
	if !ok {
		resp, _ := json.Marshal(rpc.NewJsonRpcMessageWithError(req.ID, -32000, string(invalidRequest)))
		s.writeJson(w, resp)
		return nil, chain, false
	}

	return req, chain, true
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	req, chain, ok := s.validRequest(w, r)
	if !ok {
		return
	}

	msg := s.handlers[req.Method](s, chain, req)
	resp, _ := json.Marshal(msg)

	s.writeJson(w, resp)
}

func (s *Server) registerHandlers() {
	s.handlers["eth_getLogsByUserOperation"] = eth_getLogsByUserOperation
	s.handlers["eth_getLogs"] = eth_getLogs
}

type Status struct {
	Chain       string `json:"chain"`
	BlockNumber int64  `json:"block_number"`
	LatestBlock int64  `json:"latest_block"`
	CatchingUp  bool   `json:"catching_up"`
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	var stats []Status

	for _, chain := range s.chains {
		var blockNumber, latestBlock int64
		if s.readonly {
			v, _ := s.db.Get(DbKeyStartBlock(chain))
			blockNumber = cast.ToInt64(string(v))
			latestBlock = blockNumber
		} else {
			v, ok := gBlockNumberMap.Load(chain)
			if ok {
				blockNumber = v.(int64)
			}
			v, ok = gLatestBlockMap.Load(chain)
			if ok {
				latestBlock = v.(int64)
			}
		}

		stats = append(stats, Status{
			Chain:       chain,
			BlockNumber: blockNumber,
			LatestBlock: latestBlock,
			CatchingUp:  !(blockNumber >= (latestBlock - 5)),
		})
	}

	data, _ := json.Marshal(stats)
	w.Write(data)
}
