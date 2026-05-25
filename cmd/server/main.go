package main

import (
	"context"
	"log"
	"net"
	"os"

	"github.com/mrigangha/nosqldb/generated"
	"github.com/mrigangha/nosqldb/internal"

	"google.golang.org/grpc"
)

type Server struct {
	generated.UnimplementedNoSQLDBServer

	db *internal.Database
}

func (s *Server) Set(
	ctx context.Context,
	req *generated.SetRequest,
) (*generated.SetResponse, error) {

	err := s.db.Set(req.Key, req.Value)

	if err != nil {
		return nil, err
	}

	return &generated.SetResponse{
		Ok: true,
	}, nil
}

func (s *Server) Get(
	ctx context.Context,
	req *generated.GetRequest,
) (*generated.GetResponse, error) {

	val := s.db.Get(req.Key)

	if val == nil {
		return &generated.GetResponse{
			Found: false,
		}, nil
	}

	return &generated.GetResponse{
		Value: val,
		Found: true,
	}, nil
}

func (s *Server) Delete(
	ctx context.Context,
	req *generated.DeleteRequest,
) (*generated.DeleteResponse, error) {

	err := s.db.Del(req.Key)

	if err != nil {
		return nil, err
	}

	return &generated.DeleteResponse{
		Ok: true,
	}, nil
}

func main() {

	db := internal.NewDatabase()
	defer db.Close()

	port := os.Getenv("PORT")

	if port == "" {
		port = "10000"
	}

	lis, err := net.Listen(
		"tcp",
		"0.0.0.0:"+port,
	)

	if err != nil {
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer()

	generated.RegisterNoSQLDBServer(
		grpcServer,
		&Server{
			db: db,
		},
	)

	log.Println("gRPC DB running on :" + port)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
