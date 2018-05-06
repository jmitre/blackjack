package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"math/rand"
	"strconv"
)

var clientCount = 0
var allClients = make(map[net.Conn] client)
var newConnections = make(chan net.Conn)
var deadConnections = make(chan net.Conn)

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

// Start game if clientCount > 0
// Initial setup:
//		1. initialize chips for each player
//		2. setup deck
// 			a. 8 decks, 52 cards each, shuffled
//


func main() {
	server := startupServer(6000)
	acceptNewConnections(server)
	runGame()
	manageConnections()
}

func manageConnections() {
	for {
		select {
		case conn := <-newConnections:
			newConnection(conn)
		case conn := <-deadConnections:
			deleteDeadConnections(conn)
		}
	}
}

func deleteDeadConnections(conn net.Conn) {
	log.Printf("Client %s disconnected", allClients[conn].name)
	delete(allClients, conn)
}

func broadcastMessage(message string) {
	for conn := range allClients {
		go func(conn net.Conn, message string) {
			_, err := conn.Write([]byte(message))

			if err != nil {
				deadConnections <- conn
			}
		}(conn, message)
	}
	log.Printf("message broadcast to %d client(s): %s", len(allClients), message)
}

func newConnection(conn net.Conn) {
	log.Printf("Accepted new client, #%d", clientCount)

	c := new(client)
	c.id = clientCount
	sendMsg(conn, "What is your name? ")
	c.name = string(read(conn))
	c.chips = 100

	allClients[conn] = *c
	clientCount += 1

	broadcastMessage(fmt.Sprintf("%s has connected\n", c.name))
}

func read(conn net.Conn) []byte {
	reader := bufio.NewReader(conn)
	buf := make([]byte, 256)
	buf, _, _ = reader.ReadLine()
	return buf
}

func sendMsg(conn net.Conn, msg string) {
	go func(conn net.Conn, msg string) {
		_, err := conn.Write([]byte(msg))

		if err != nil {
			deadConnections <- conn
		}
	}(conn, msg)
}

func runGame() {
	go func() {
		initializeGame()
		for {
			if clientCount > 0 {
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
				betMap := make(map[int]int)
				for conn := range allClients {
					sendMsg(conn, "How much would you like to bet? ")
					betString := string(read(conn))
					client := allClients[conn]
					bet, err := strconv.Atoi(betString)
					if err != nil {
						log.Println(err)
						bet = 0
					}
					betMap[client.id] = bet
					log.Printf("betMap: %s", betMap)
				}
			}
		}
	}()
}

func acceptNewConnections(server net.Listener) {
	go func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				log.Println(err)
				os.Exit(1)
			}
			newConnections <- conn
		}
	}()
}

func startupServer(port int) net.Listener {
	address := fmt.Sprintf(":%d", port)
	server, err := net.Listen("tcp", address)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	return server
}

func initializeGame() {
	buildDeck()
}

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
	log.Printf("deck: %s", deck)
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


