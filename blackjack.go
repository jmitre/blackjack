package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

type client struct {
	name string
	id int
	cash int
}

func main() {
	clientCount := 0

	allClients := make(map[net.Conn] client)

	newConnections := make(chan net.Conn)
	deadConnections := make(chan net.Conn)
	messages := make(chan string)

	// startup server listening on port 6000
	server, err := net.Listen("tcp", ":6000")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// accept new connections
	go func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			newConnections <- conn
		}
	}()

	for {
		select {
		// new connection from a client
		case conn := <-newConnections:
			log.Printf("Accepted new client, #%d", clientCount)

			c := new(client)
			c.id = clientCount

			go func(conn net.Conn, message string) {
				_, err := conn.Write([]byte(message))

				if err != nil {
					deadConnections <- conn
				}
			}(conn, "What is your name? ")
			reader := bufio.NewReader(conn)
			buf := make([]byte, 256)
			buf,_, _ = reader.ReadLine()
			c.name = string(buf)
			c.cash = 100

			allClients[conn] = *c
			clientCount += 1
			messages <- fmt.Sprintf("Client %s connected", c.name)

			// go on with the game...?
			// print game state
			// spike a simple card game




		// broadcast a message on the messages channel
		case message := <-messages:
			for conn := range allClients {

				go func(conn net.Conn, message string) {
					_, err := conn.Write([]byte(message))

					if err != nil {
						deadConnections <- conn
					}
				}(conn, message)
			}
			log.Printf("New message: %s", message)
			log.Printf("Broadcast to %d clients", len(allClients))

		// remove clients that have disconnected from the allClients channel
		case conn := <-deadConnections:
			log.Printf("Client %s disconnected", allClients[conn].name)
			delete(allClients, conn)
		}
	}
}
