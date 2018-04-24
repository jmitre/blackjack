package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"math/rand"
)

type client struct {
	name string
	id int
	chips int
}

type card struct {
	value string
	suit string
}
type Deck []card
const numDecks = 8

func buildDeck() (deck Deck) {
	values := []string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}
	suits := []string{"H", "D", "C", "S"}

	for j := 0; j < numDecks; j++ {
		for i := 0; i < len(values); i++ {
			for n := 0; n < len(suits); n++ {
				card := card{
					value: values[i],
					suit:  suits[n],
				}
				deck = append(deck, card)
			}
		}
	}
	deck = shuffle(deck)
	fmt.Printf("deck: %s", deck)
	return
}

func shuffle(d Deck) Deck {
	for i := 1; i < len(d); i++ {
		r := rand.Intn(i + 1)
		if i != r {
			d[r], d[i] = d[i], d[r]
		}
	}
	return d
}

func initializeGame() {
	buildDeck()
}

func main() {
	clientCount := 0
	allClients := make(map[net.Conn] client)
	newConnections := make(chan net.Conn)
	deadConnections := make(chan net.Conn)
	messagesToClients := make(chan string)

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

	// Start game if clientCount > 0
	// Initial setup:
	//		1. initialize chips for each player
	//		2. setup deck
	// 			a. 8 decks, 52 cards each, shuffled
	//
	// Game Loop (update state to all clients after each state change)
	//		1. all clients place bets
	//		2. burn 1 card
	// 		3. deal 1 to each (down for the dealer, up for all players)
	//		4. deal 1 to each (up for all)
	//			a. check if dealer has blackjack
	//			b. if dealer has Ace, offer insurance
	//		5. iterate through clients and ask for their move
	//			a. hit (until they bust)
	//			b. stay
	//			c. split
	//			d. double-down
	//		6. dealer reveals 2nd card
	//			a. dealer hits if total is < 17
	//		7. deal out winnings/take losses
	//		8. start another round

	// For first MVP: all players have unlimited chips, 1 value chips exists, clients cannot split or double-down,
	// insurance is not available

	//gameStart := false
	for {
		//if !gameStart && clientCount > 0 {
		//	gameStart = true
		//	initializeGame()
		//	messagesToClients <- fmt.Sprintln("Game start!")
		//} else {
		//	gameStart = false
		//}

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
				c.chips = 100

				allClients[conn] = *c
				clientCount += 1
				go func (){ messagesToClients <- fmt.Sprintf("%s has connected", c.name) }()

			// broadcast a message on the messagesToClients channel
			case message := <-messagesToClients:
				for conn := range allClients {
					go func(conn net.Conn, message string) {
						_, err := conn.Write([]byte(message))

						if err != nil {
							deadConnections <- conn
						}
					}(conn, message)
				}
				log.Printf("message: %s", message)
				log.Printf("Broadcast to %d clients", len(allClients))

			// remove clients that have disconnected from the allClients channel
			case conn := <-deadConnections:
				log.Printf("Client %s disconnected", allClients[conn].name)
				delete(allClients, conn)
		}
	}
}
