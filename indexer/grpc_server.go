package indexer

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/log"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/rpc"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/x/proto"
	"google.golang.org/grpc"
	"net"
)

type GrpcServer struct {
	proto.UnimplementedRelayServer
	listen               string
	db                   database.KVStore
	entryPoint           string
	handlers             map[string]func(s Rpc, req *rpc.JsonRpcMessage) *rpc.JsonRpcMessage
	maxConcurrentStreams int
	logger               log.Logger
}

func (s *GrpcServer) Db() database.KVStore {
	return s.db
}

func (s *GrpcServer) EntryPoint() string {
	return s.entryPoint
}

func NewGrpcServer(cfg *Config, db database.KVStore) *GrpcServer {
	return &GrpcServer{
		listen:               cfg.GrpcListen,
		db:                   db,
		entryPoint:           cfg.EntryPoint,
		handlers:             map[string]func(s Rpc, req *rpc.JsonRpcMessage) *rpc.JsonRpcMessage{},
		maxConcurrentStreams: 4096,
		logger:               log.Module("grpc-server"),
	}
}

func (s *GrpcServer) registerHandlers() {
	s.handlers["eth_getLogsByUserOperation"] = eth_getLogsByUserOperation
	s.handlers["eth_getLogs"] = eth_getLogs
}

func (s *GrpcServer) Run() error {
	s.registerHandlers()

	var opts []grpc.ServerOption
	var interceptor = func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		return handler(ctx, req)
	}

	opts = append(opts, grpc.UnaryInterceptor(interceptor))
	server := grpc.NewServer(
		grpc.MaxConcurrentStreams(uint32(s.maxConcurrentStreams)),
	)

	proto.RegisterRelayServer(server, s)

	listen, err := net.Listen("tcp", s.listen)
	if err != nil {
		log.Error("failed to listen", "server", s.listen, "err", err)
		panic(err)
	}

	s.logger.Info("grpc server listen: " + s.listen)
	return server.Serve(listen)
}

func (s *GrpcServer) Relay(ctx context.Context, request *proto.Request) (*proto.Response, error) {
	req, resp, err := s.parseRequestBody(request)
	if err != nil {
		return resp, err
	}

	msg := s.handlers[req.Method](s, req)
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
