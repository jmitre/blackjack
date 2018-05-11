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
var allPlayers = make(map[net.Conn]*player)
var newConnections = make(chan net.Conn)
var deadConnections = make(chan net.Conn)
var messages = make(chan string)

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
	broadcastMessage(fmt.Sprintf("%s disconnected", allPlayers[conn].name))
	delete(allPlayers, conn)
}

func broadcastMessage(msg string) {
	for conn := range allPlayers {
		sendMsg(conn, msg)
	}
	log.Printf("msg broadcast to %d player(s): %s", len(allPlayers), msg)
}

func newConnection(conn net.Conn) {
	log.Printf("accepted new player, #%d", playerCount)

	p := player{}
	p.id = playerCount
	sendMsg(conn, "What is your name?")
	p.name = string(read(conn))
	p.chips = 200
	p.cards = nil

	allPlayers[conn] = &p
	playerCount += 1

	broadcastMessage(fmt.Sprintf("%s has connected", p.name))
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
		messages <- msg
	}(conn, msg)
}

func runGame() {
	go func() {
		deck := buildDeck()
		dealer := player{"Dealer", 0, 1000000, nil}
		bets := make(map[int]int)
		results := make(map[string] int)
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
				//		8. clear and start another round

				// For first MVP: players cannot split or double-down, insurance is not available
				// Future features: counting cards score, GUI,

				//		1. all players place bets
				bets = getBets(bets)

				//		2. burn 1 card
				deck = deck[:len(deck)-1]
				broadcastMessage("dealer has burned 1 card")

				// 		3. deal 1 to each (down for the dealer, up for all players)
				deck, dealer = deal(deck, dealer, true)

				//		4. deal 1 to each (up for all)
				deck, dealer = deal(deck, dealer, false)

				//		5. iterate through players and ask for their move
				//			a. hit (until they bust)
				//			b. stay
				for conn := range allPlayers {
					stay := false
					for stay == false {
						for correctInput := false; !correctInput; {
							sendMsg(conn, "Would you like to (h)it or (s)tay?")
							move := string(read(conn))
							correctInput = true
							if move != "h" && move != "s" {
								sendMsg(conn, "incorrect input")
								correctInput = false
							} else {
								player := allPlayers[conn]
								if move == "h" {
									card := deck[len(deck)-1]
									player.cards = append(player.cards, card)
									deck = deck[:len(deck)-1]
									broadcastMessage(fmt.Sprintf("%s has %v", player.name, player.cards))
									sum := getSumOfHand(player)
									log.Printf("%s has handSum: %v", player.name, sum)
									if sum > 21 {
										broadcastMessage(fmt.Sprintf("%s bust", player.name))
										results[player.name] = 0
										stay = true
									} else {
										results[player.name] = sum
									}
								} else {
									sum := getSumOfHand(player)
									results[player.name] = sum
									stay = true
								}
							}
						}
					}
				}

				//		6. dealer reveals 2nd card
				//			a. dealer hits if total is < 17
				broadcastMessage(fmt.Sprintf("dealer has %v", dealer.cards))
				sum := getSumOfHand(&dealer)
				for sum < 17 {
					card := deck[len(deck)-1]
					dealer.cards = append(dealer.cards, card)
					deck = deck[:len(deck)-1]
					broadcastMessage(fmt.Sprintf("dealer has %v", dealer.cards))
					sum = getSumOfHand(&dealer)
				}
				results[dealer.name] = sum
				log.Printf("dealer has handSum: %v", sum)
				log.Printf("results map: %v\n", results)

				//		7. deal out winnings/take losses
				if results[dealer.name] > 21 {
					broadcastMessage("dealer bust")
				}

				//		8. clear and start another round
				dealer.cards = nil
				for conn := range allPlayers {
					allPlayers[conn].cards = nil
				}

				for i := range bets {
					delete(bets, i)
				}

				for i := range results {
					delete(results, i)
				}
			}
		}
	}()
}

func getSumOfHand(p *player) int {
	sum := 0
	numAces := 0
	for i := range p.cards {
		if p.cards[i].value == "J" || p.cards[i].value == "Q" || p.cards[i].value == "K" {
			sum += 10
		} else if p.cards[i].value == "A" {
			numAces++
		} else {
			val, err := strconv.Atoi(p.cards[i].value)
			if err != nil {
				log.Println(err)
			} else {
				sum += val
			}
		}
	}
	if sum + 11 + (numAces - 1) > 21 {
		sum += numAces
	} else {
		sum += 11 + (numAces - 1)
	}
	return sum
}

func getBets(bets map[int]int) map[int]int {
	for conn := range allPlayers {
		for correctInput := false; !correctInput; {
			sendMsg(conn, fmt.Sprintf("You have %d chips.", allPlayers[conn].chips))
			sendMsg(conn, "How much would you like to bet? ")
			betString := string(read(conn))
			bet, err := strconv.Atoi(betString)
			correctInput = true
			if err != nil {
				log.Println(err)
				sendMsg(conn, "incorrect input")
				bet = 0
				correctInput = false
			}
			bets[allPlayers[conn].id] = bet
			log.Printf("bets: %v", bets)
		}
	}
	return bets
}

func deal(deck Deck, dealer player, printDealer bool) (Deck, player) {
	card := deck[len(deck)-1]
	dealer.cards = append(dealer.cards, card)
	deck = deck[:len(deck)-1]
	log.Printf("dealer has: %v", dealer.cards)
	for conn := range allPlayers {
		player := allPlayers[conn]
		card := deck[len(deck)-1]
		player.cards = append(player.cards, card)
		deck = deck[:len(deck)-1]
		log.Printf("%s has: %v", player.name, player.cards)
	}
	if printDealer {
		broadcastMessage(fmt.Sprintf("dealer has %v", dealer.cards[0]))
		for conn := range allPlayers {

			broadcastMessage(fmt.Sprintf("%s has %v", allPlayers[conn].name, allPlayers[conn].cards))
		}
	} else {
		for conn := range allPlayers {
			broadcastMessage(fmt.Sprintf("%s has %v", allPlayers[conn].name, allPlayers[conn].cards))
		}
	}
	log.Printf("deck: %v", deck)
	return deck, dealer
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


