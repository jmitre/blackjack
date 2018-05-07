package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"math/rand"
	"strconv"
	"time"
)

var playerCount = 1
var allPlayers = make(map[net.Conn]player)
var newConnections = make(chan net.Conn)
var deadConnections = make(chan net.Conn)

type player struct {
	name string
	id int
	chips int
	cards []card
}

type card struct {
	value string
	suit string
}

type Deck []card
const numDecks = 8


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
	log.Printf("player %s disconnected", allPlayers[conn].name)
	delete(allPlayers, conn)
}

func broadcastMessage(message string) {
	for conn := range allPlayers {
		go func(conn net.Conn, message string) {
			message += "\n"
			_, err := conn.Write([]byte(message))

			if err != nil {
				deadConnections <- conn
			}
		}(conn, message)
	}
	log.Printf("message broadcast to %d player(s): %s", len(allPlayers), message)
}

func newConnection(conn net.Conn) {
	log.Printf("accepted new player, #%d", playerCount)

	c := new(player)
	c.id = playerCount
	sendMsg(conn, "What is your name? ")
	c.name = string(read(conn))
	c.chips = 100
	c.cards = nil

	allPlayers[conn] = *c
	playerCount += 1

	broadcastMessage(fmt.Sprintf("%s has connected", c.name))
}

func read(conn net.Conn) []byte {
	reader := bufio.NewReader(conn)
	buf := make([]byte, 256)
	buf, _, _ = reader.ReadLine()
	return buf
}

func sendMsg(conn net.Conn, msg string) {
	go func(conn net.Conn, msg string) {
		msg += "\n"
		_, err := conn.Write([]byte(msg))

		if err != nil {
			deadConnections <- conn
		}
	}(conn, msg)
}

func runGame() {
	go func() {
		deck := buildDeck()
		dealer := player{"Dealer", 0, 1000000, nil}
		for {
			if playerCount > 1 {
				// Game Loop (update state to all players after each state change)
				//		1. all players place bets
				//		2. burn 1 card
				// 		3. deal 1 to each (down for the dealer, up for all players)
				//		4. deal 1 to each (up for all)
				//			a. check if dealer has blackjack
				//			b. if dealer has Ace, offer insurance
				//		5. iterate through players and ask for their move
				//			a. hit (until they bust)
				//			b. stay
				//			c. split
				//			d. double-down
				//		6. dealer reveals 2nd card
				//			a. dealer hits if total is < 17
				//		7. deal out winnings/take losses
				//		8. start another round

				// For first MVP: all players have unlimited chips, 1 value chips exists, players cannot split or double-down,
				// insurance is not available
				bets := make(map[int]int)

				//		1. all players place bets
				for conn := range allPlayers {
					for correctInput := false; !correctInput; {
						sendMsg(conn, "How much would you like to bet? ")
						betString := string(read(conn))
						player := allPlayers[conn]
						bet, err := strconv.Atoi(betString)
						correctInput = true
						if err != nil {
							log.Println(err)
							sendMsg(conn, "incorrect input")
							bet = 0
							correctInput = false
						}
						bets[player.id] = bet
						log.Printf("bets: %s", bets)
					}
				}

				//		2. burn 1 card
				deck = deck[:len(deck)-1]
				broadcastMessage("dealer has burned 1 card")

				// 		3. deal 1 to each (down for the dealer, up for all players)
				card := deck[len(deck)-1]
				dealer.cards = append(dealer.cards, card)
				log.Printf("dealer has: %s", dealer.cards)
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
	log.Printf("initial deck: %s", deck)
	return
}

func shuffle(d Deck) Deck {
	for i := 1; i < len(d); i++ {
		rand.Seed(time.Now().UTC().UnixNano())
		r := rand.Intn(i + 1)
		if i != r {
			d[r], d[i] = d[i], d[r]
		}
	}
	return d
}


