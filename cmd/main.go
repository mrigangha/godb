package main

import (
	"fmt"

	"github.com/mrigangha/nosqldb/internal"
)

func main() {
	fmt.Println("Hello World!!!")
	db := internal.NewDatabase()
	defer db.Close()

	fmt.Println(db.Get("hello"))
}
