package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"connectrpc.com/connect"

	"github.com/mrigangha/nosqldb/generated"
	"github.com/mrigangha/nosqldb/generated/generatedconnect"
	"github.com/mrigangha/nosqldb/internal"
)

type Server struct {
	db *internal.Database
}

func (s *Server) Set(
	ctx context.Context,
	req *connect.Request[generated.SetRequest],
) (*connect.Response[generated.SetResponse], error) {

	err := s.db.Set(
		req.Msg.Key,
		req.Msg.Value,
	)

	if err != nil {
		return nil, err
	}

	return connect.NewResponse(
		&generated.SetResponse{
			Ok: true,
		},
	), nil
}

func (s *Server) Get(
	ctx context.Context,
	req *connect.Request[generated.GetRequest],
) (*connect.Response[generated.GetResponse], error) {

	val := s.db.Get(req.Msg.Key)

	if val == nil {
		return connect.NewResponse(
			&generated.GetResponse{
				Found: false,
			},
		), nil
	}

	return connect.NewResponse(
		&generated.GetResponse{
			Value: val,
			Found: true,
		},
	), nil
}

func (s *Server) Delete(
	ctx context.Context,
	req *connect.Request[generated.DeleteRequest],
) (*connect.Response[generated.DeleteResponse], error) {

	err := s.db.Del(req.Msg.Key)

	if err != nil {
		return nil, err
	}

	return connect.NewResponse(
		&generated.DeleteResponse{
			Ok: true,
		},
	), nil
}

func main() {

	db := internal.NewDatabase()
	defer db.Close()

	server := &Server{
		db: db,
	}

	mux := http.NewServeMux()

	path, handler := generatedconnect.NewNoSQLDBHandler(server)

	mux.Handle(path, handler)

	port := os.Getenv("PORT")

	if port == "" {
		port = "10000"
	}

	log.Println("ConnectRPC DB running on :" + port)

	err := http.ListenAndServe(
		"0.0.0.0:"+port,
		mux,
	)

	if err != nil {
		log.Fatal(err)
	}
}
