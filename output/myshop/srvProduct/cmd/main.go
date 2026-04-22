package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"myshop/pkg/config"
	"myshop/pkg/database"
	"myshop/srvProduct/internal/handler"
	pb "myshop/common/kitexGen/product"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var confPath string

func init() {
	flag.StringVar(&confPath, "config", "configs/config.yaml", "config file")
}

func main() {
	flag.Parse()
	cfg, _ := config.Load(confPath)
	db, _ := database.NewDB(&cfg.Database)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	lis, _ := net.Listen("tcp", addr)
	s := grpc.NewServer()
	pb.RegisterProductServiceServer(s, handler.NewProductHandler(db))
	reflection.Register(s)
	log.Printf("Starting on %s", addr)
	s.Serve(lis)
}
