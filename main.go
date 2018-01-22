package main

import (
	"fmt"
	"net"
	"os"

	"github.com/cnosuke/jalindi/pb"
	"github.com/lestrrat/go-fluent-client"
	"github.com/urfave/cli"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

var (
	// Version and Revision are replaced when building.
	// To set specific version, edit Makefile.
	Version  = "0.0.1"
	Revision = "xxx"

	Name  = "jalindi"
	Usage = "gRPC to fluentd event logger proxy"
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

	app := cli.NewApp()
	app.Version = fmt.Sprintf("%s (%s)", Version, Revision)
	app.Name = Name
	app.Usage = Usage

	var (
		binding      string
		insecureFlag bool
		certPath     string
		keyPath      string
		fluentdAddr  string
		fluentdTag   string
	)

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "binding, b",
			Usage:       "Server binding address",
			Value:       "127.0.0.1:8888",
			Destination: &binding,
		},
		cli.BoolFlag{
			Name:        "insecure",
			Usage:       "Run without TLS",
			Destination: &insecureFlag,
		},
		cli.StringFlag{
			Name:        "cert",
			Usage:       "Path to TLS cert file",
			Value:       "",
			Destination: &certPath,
		},
		cli.StringFlag{
			Name:        "key",
			Usage:       "Path to TLS key file",
			Value:       "",
			Destination: &keyPath,
		},
		cli.StringFlag{
			Name:        "fluentd",
			Usage:       "Address to fluentd (like 127.0.0.1:24224)",
			Value:       "127.0.0.1:24224",
			Destination: &fluentdAddr,
		},
		cli.StringFlag{
			Name:        "fluentdTag",
			Usage:       "Fluentd tag",
			Value:       "events",
			Destination: &fluentdTag,
		},
	}

	app.Action = func(c *cli.Context) error {

		listener, err := net.Listen("tcp", binding)

		grpcServerOptions := []grpc.ServerOption{}

		if !insecureFlag {
			if len(certPath) == 0 || len(keyPath) == 0 {
				return cli.NewExitError("--cert and --key should be set. If you want to serve without TLS, use --insecure flag.", 1)
			}

			cred, err := credentials.NewServerTLSFromFile(certPath, keyPath)
			if err != nil {
				logger.Errorf("Building up server TLS credentials failed: %v", err)
				return cli.NewExitError("", 1)
			}

			grpcServerOptions = append(grpcServerOptions, grpc.Creds(cred))
		}

		svr := grpc.NewServer(grpcServerOptions...)

		fl, err := fluent.NewBuffered(
			fluent.WithConnectOnStart(true),
			fluent.WithAddress(fluentdAddr),
			fluent.WithWriteThreshold(10*1024),
		)
		if err != nil {
			logger.Errorf("Building up fluentd client failed: %v", err)
			return cli.NewExitError("", 1)
		}

		handler := NewHandler(fl, fluentdTag, logger)

		jalindi.RegisterJalindiServiceServer(
			svr,
			handler,
		)

		reflection.Register(svr)

		logger.Infow("Starting server",
			"binding", binding,
			"insecure", insecureFlag,
			"certPath", certPath,
			"keyPath", keyPath,
			"fluentd", fluentdAddr,
			"tag", fluentdTag,
		)

		if err := svr.Serve(listener); err != nil {
			logger.Error(err)
			return cli.NewExitError("", 1)
		}

		return nil
	}

	app.Run(os.Args)
}
