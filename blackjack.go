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
}

func main() {

	clientCount := 0

	allClients := make(map[net.Conn] client)

	newConnections := make(chan net.Conn)

	deadConnections := make(chan net.Conn)

	messages := make(chan string)

	server, err := net.Listen("tcp", ":6000")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

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

			fmt.Sprintf("name: %s", c.name)

			allClients[conn] = *c
			clientCount += 1

			go func(conn net.Conn, name string) {
				reader := bufio.NewReader(conn)
				for {
					incoming, err := reader.ReadString('\n')
					if err != nil {
						break
					}
					messages <- fmt.Sprintf("Client %s > %s", name, incoming)
				}

				deadConnections <- conn

			}(conn, allClients[conn].name)

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

		case conn := <-deadConnections:
			log.Printf("Client %s disconnected", allClients[conn].name)
			delete(allClients, conn)
		}
	}
}
