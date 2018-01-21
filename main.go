package main

import (
	"fmt"
	"net"
	"os"

	"github.com/cnosuke/jalindi/pb"
	"github.com/lestrrat/go-fluent-client"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var logger *zap.SugaredLogger

func main() {
	zapConfig := zap.NewDevelopmentConfig()
	zapConfig.OutputPaths = []string{"stdout"}
	zapLogger, err := zapConfig.Build()
	if err != nil {
		fmt.Printf("Building logger error: %v", err)
		os.Exit(1)
	}

	defer zapLogger.Sync()
	logger = zapLogger.Sugar()

	undo := zap.ReplaceGlobals(zapLogger)
	defer undo()

	binding := "localhost:8888"

	listener, err := net.Listen("tcp", binding)

	grpcServerOptions := []grpc.ServerOption{}

	svr := grpc.NewServer(grpcServerOptions...)

	fl, _ := fluent.NewBuffered()

	handler := NewHandler(fl, "events", logger)

	jalindi.RegisterJalindiServiceServer(
		svr,
		handler,
	)

	reflection.Register(svr)

	if err := svr.Serve(listener); err != nil {
		logger.Fatal(err)
	}

}
