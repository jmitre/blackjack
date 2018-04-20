package main

import (
	"fmt"
	"log"
	"net"
	"encoding/gob"
)

type P struct {
	M, N int64
}

func main() {
	fmt.Println("start client")
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatal("Connection error", err)
	}

	encoder := gob.NewEncoder(conn)
	p := &P{1, 2}
	encoder.Encode(p)

	dec := gob.NewDecoder(conn)
	p2 := &P{}
	dec.Decode(p2)
	fmt.Printf("Received : %+v\n", p2)

	conn.Close()
	fmt.Println("done")
}