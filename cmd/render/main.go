package main

import (
	"context"
	"log"
	"time"

	"github.com/mrigangha/nosqldb/generated"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {

	// TLS connection for Render
	conn, err := grpc.Dial(
		"godb-lzra.onrender.com:443",
		grpc.WithTransportCredentials(
			credentials.NewTLS(nil),
		),
	)

	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	client := generated.NewNoSQLDBClient(conn)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	defer cancel()

	// =========================
	// SET
	// =========================

	_, err = client.Set(ctx, &generated.SetRequest{
		Key:   "name",
		Value: []byte("mrigangha"),
	})

	if err != nil {
		log.Fatal("SET ERROR:", err)
	}

	log.Println("SET SUCCESS")

	// =========================
	// GET
	// =========================

	getResp, err := client.Get(ctx, &generated.GetRequest{
		Key: "name",
	})

	if err != nil {
		log.Fatal("GET ERROR:", err)
	}

	if getResp.Found {

		log.Println(
			"GET SUCCESS:",
			string(getResp.Value),
		)

	} else {

		log.Println("KEY NOT FOUND")
	}

	// =========================
	// DELETE
	// =========================

	_, err = client.Delete(ctx, &generated.DeleteRequest{
		Key: "name",
	})

	if err != nil {
		log.Fatal("DELETE ERROR:", err)
	}

	log.Println("DELETE SUCCESS")

	// =========================
	// VERIFY DELETE
	// =========================

	verifyResp, err := client.Get(ctx, &generated.GetRequest{
		Key: "name",
	})

	if err != nil {
		log.Fatal(err)
	}

	if !verifyResp.Found {
		log.Println("DELETE VERIFIED")
	}
}
