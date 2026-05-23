package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mrigangha/nosqldb/internal"
)

var db internal.Database

var mu sync.RWMutex

type Request struct {
	Cmd   string `json:"cmd"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

func handle(conn net.Conn) {

	defer conn.Close()

	fmt.Println("Client connected")

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {

		line := scanner.Bytes()

		var req Request

		err := json.Unmarshal(line, &req)
		if err != nil {

			conn.Write([]byte("INVALID JSON\n"))
			continue
		}

		switch req.Cmd {

		case "SET":

			mu.Lock()

			db.Set(req.Key, []byte(req.Value))

			mu.Unlock()

			conn.Write([]byte("OK\n"))

		case "GET":

			mu.RLock()

			byteS := db.Get(req.Key)

			mu.RUnlock()

			conn.Write(byteS)
			conn.Write([]byte("\n"))

		case "DELETE":

			mu.Lock()

			db.Del(req.Key)

			mu.Unlock()

			conn.Write([]byte("DELETED\n"))

		default:

			conn.Write([]byte("UNKNOWN COMMAND\n"))
		}
	}

	fmt.Println("Client disconnected")
}

func flushLoop() {

	for {

		time.Sleep(5 * time.Second)

		mu.Lock()

		if db.ShouldFlush() {

			fmt.Println("Flushing MemTable...")

			db.Flush()
		}

		mu.Unlock()
	}
}

func main() {

	db = internal.NewDatabase()
	defer db.Close()

	go flushLoop()

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	fmt.Println("TCP DB Server Running on :8080")

	for {

		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		go handle(conn)
	}
}
