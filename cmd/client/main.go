package main

import (
	"context"
	"log"
	"time"

	"github.com/mrigangha/nosqldb/generated"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {

	conn, err := grpc.Dial(
		"localhost:50051",
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)

	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	client := generated.NewNoSQLDBClient(conn)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	defer cancel()

	// SET
	_, err = client.Set(ctx, &generated.SetRequest{
		Key:   "name",
		Value: []byte("mrigangha"),
	})

	if err != nil {
		log.Fatal(err)
	}

	log.Println("SET success")

	// GET
	resp, err := client.Get(ctx, &generated.GetRequest{
		Key: "name",
	})

	if err != nil {
		log.Fatal(err)
	}

	log.Println(
		"GET:",
		string(resp.Value),
	)

	// DELETE
	_, err = client.Delete(ctx, &generated.DeleteRequest{
		Key: "name",
	})

	if err != nil {
		log.Fatal(err)
	}

	log.Println("DELETE success")
}
