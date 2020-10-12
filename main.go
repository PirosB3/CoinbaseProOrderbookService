package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"pirosb3/real_feed/controller"
	"pirosb3/real_feed/rpc"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
	market := os.Getenv("MARKET")
	port := "8000"
	ctx, _ := context.WithCancel(context.Background())

	// Start feed controller
	fc := controller.NewFeedController(ctx, market)
	fc.Start()

	// Start prometheus server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)
	}()

	// Create wrapper service
	orderbookController := controller.NewOrderbookGrpcController(fc, market)

	// Start gRPC server
	grpcServer := grpc.NewServer()
	rpc.RegisterOrderbookServiceServer(grpcServer, *orderbookController)
	// ... // determine whether to use TLS
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalln(err.Error())
	}
	log.WithField("market", market).WithField("port", port).Infoln("Starting gRPC server")
	grpcServer.Serve(lis)
}
