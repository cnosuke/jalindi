package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/cnosuke/jalindi/pb"
	"github.com/golang/glog"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/urfave/cli"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	Name     = "gateway"
	Version  string
	Revision string

	endpoint     string
	binding      string
	headerPrefix string
)

func run() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var mux *runtime.ServeMux

	if len(headerPrefix) != 0 {
		headerMatcherFunc := func(str string) (string, bool) {
			return str, strings.HasPrefix(strings.ToLower(str), headerPrefix)
		}

		headerMatcher := runtime.HeaderMatcherFunc(headerMatcherFunc)

		muxOpts := []runtime.ServeMuxOption{
			runtime.WithIncomingHeaderMatcher(headerMatcher),
		}

		mux = runtime.NewServeMux(muxOpts...)
	} else {
		mux = runtime.NewServeMux()
	}

	opts := []grpc.DialOption{grpc.WithInsecure()}

	err := jalindi.RegisterJalindiServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)

	if err != nil {
		return err
	}

	return http.ListenAndServe(binding, mux)
}

func main() {
	defer glog.Flush()

	app := cli.NewApp()
	app.Version = fmt.Sprintf("%s (%s)", Version, Revision)
	app.Name = Name

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "endpoint, e",
			Usage:       "gRPC server endpoint",
			Value:       "127.0.0.1:5000",
			Destination: &endpoint,
		},
		cli.StringFlag{
			Name:        "binding, b",
			Usage:       "Server binding address",
			Value:       "127.0.0.1:8080",
			Destination: &binding,
		},
		cli.StringFlag{
			Name:        "Incoming HTTP header matching prefix",
			Usage:       "header-prefix, h",
			Value:       "",
			Destination: &headerPrefix,
		},
	}

	app.Action = func(c *cli.Context) error {
		if err := run(); err != nil {
			glog.Fatal(err)
			return err
		}

		return nil
	}

	app.Run(os.Args)
}
