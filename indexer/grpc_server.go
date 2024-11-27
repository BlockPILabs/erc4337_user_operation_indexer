package indexer

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"

	"github.com/BlockPILabs/erc4337_user_operation_indexer/database"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/log"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/rpc"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/x/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type GrpcServer struct {
	proto.UnimplementedRelayServer
	cfg                  *Config
	db                   database.KVStore
	handlers             map[string]handlerFunc
	maxConcurrentStreams int
	logger               log.Logger
	compress             bool
	chain                string
}

func (s *GrpcServer) Chain() string {
	return s.chain
}

func (s *GrpcServer) Db() database.KVStore {
	return s.db
}

func (s *GrpcServer) EntryPoints() []string {
	return s.cfg.EntryPoints
}

func (s *GrpcServer) Compressed() bool {
	return s.compress
}

func NewGrpcServer(cfg *Config, db database.KVStore) *GrpcServer {
	return &GrpcServer{
		cfg:                  cfg,
		db:                   db,
		handlers:             map[string]handlerFunc{},
		maxConcurrentStreams: 4096,
		logger:               log.Module("grpc-server"),
		compress:             cfg.Compress,
	}
}

func (s *GrpcServer) registerHandlers() {
	s.handlers["eth_getLogsByUserOperation"] = eth_getLogsByUserOperation
	s.handlers["eth_getLogs"] = eth_getLogs
}

func (s *GrpcServer) loadTLSCredentials() (credentials.TransportCredentials, error) {
	serverCert, err := credentials.NewServerTLSFromFile(s.cfg.TlsPubKey, s.cfg.TlsPrivateKey)
	if err != nil {
		return nil, err
	}
	//cfg := &tls.Config{
	//	Certificates: []tls.Certificate{serverCert},
	//	ClientAuth:   tls.NoClientCert,
	//}
	return serverCert, nil
}

func (s *GrpcServer) Run() error {
	s.registerHandlers()

	var opts []grpc.ServerOption
	var interceptor = func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		return handler(ctx, req)
	}

	opts = append(opts, grpc.UnaryInterceptor(interceptor))
	if s.cfg.UseTls {
		cred, err := s.loadTLSCredentials()
		if err != nil {
			panic(err)
		}
		opts = append(opts, grpc.Creds(cred))
	}

	server := grpc.NewServer(
		grpc.MaxConcurrentStreams(uint32(s.maxConcurrentStreams)),
	)

	proto.RegisterRelayServer(server, s)

	listen, err := net.Listen("tcp", s.cfg.GrpcListen)
	if err != nil {
		log.Error("failed to listen", "server", s.cfg.GrpcListen, "err", err)
		panic(err)
	}

	s.logger.Info("grpc server listen: " + s.cfg.GrpcListen)
	return server.Serve(listen)
}

func (s *GrpcServer) Relay(ctx context.Context, request *proto.Request) (*proto.Response, error) {
	req, resp, err := s.parseRequestBody(request)
	if err != nil {
		return resp, err
	}

	chain := ""
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		chainMd := md.Get(HeaderChain)
		if len(chainMd) > 0 {
			chain = strings.TrimSpace(chainMd[0])
		}
	}

	if len(chain) == 0 {
		return nil, errors.New(string(invalidChain))
	}

	msg := s.handlers[req.Method](s, chain, req)
	data, _ := json.Marshal(msg)

	return &proto.Response{Body: data}, nil
}

func (s *GrpcServer) parseRequestBody(request *proto.Request) (*rpc.JsonRpcMessage, *proto.Response, error) {
	req := rpc.ParseJsonRpcMessage(request.Body)
	if req == nil {
		resp, _ := json.Marshal(rpc.NewJsonRpcMessageWithError(rpc.ID0, -32000, string(invalidRequest)))
		return nil, &proto.Response{Body: resp}, errors.New(string(invalidRequest))
	}

	_, ok := s.handlers[req.Method]
	if !ok {
		resp, _ := json.Marshal(rpc.NewJsonRpcMessageWithError(req.ID, -32000, string(invalidRequest)))
		return req, &proto.Response{Body: resp}, errors.New(string(invalidRequest))
	}
	return req, nil, nil
}
