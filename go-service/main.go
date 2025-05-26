package main

import (
	"log"
	"net"

	"go-service/db"
	pb "go-service/proto"
	"go-service/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	db.Init()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCryptoServiceServer(grpcServer, &server.CryptoServer{})
	// Enable reflection
	reflection.Register(grpcServer)

	log.Println("Server listening on port 50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
