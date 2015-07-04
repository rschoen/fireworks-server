package lib

import (
	"fmt"
	rand "math/rand"
	"os"
)

type Game struct {
	gameID      string
	players     []Player
	initialized bool
	started     bool

	hints         int
	bombs         int
	deck          []Card
	discard       []Card
	piles         []int
	currentPlayer int
	startingTime  int
}

func (g *Game) Initialize() {
	// figure out how many cards are in the deck
	maxCards := 0
	for _, count := range numbers {
		maxCards += count * len(colors)
	}

	// populate the deck, discard, and piles
	g.deck = make([]Card, maxCards, maxCards)
	g.PopulateDeck()
	g.discard = make([]Card, 0, maxCards)
	g.piles = make([]int, len(colors), len(colors))

	// set starting values
	g.hints = startingHints
	g.bombs = startingBombs

	// start with no players
	g.players = make([]Player, 0, len(cardsInHand)-1)
	g.initialized = true

}

func (g *Game) AddPlayer(id string) bool {
	if !g.initialized {
		fmt.Printf("Attempting to add player before game has been fully initialized.")
		return false
	}
	if g.started {
		fmt.Printf("Attempting to add player after game has started.")
		return false
	}
	if len(g.players) >= len(cardsInHand)-1 {
		fmt.Printf("Attempted to add a player to a full game.")
		return false
	}

	g.players = append(g.players, Player{ID: id})
	return true
}

func (g *Game) Start() {
	if !g.initialized {
		fmt.Printf("Attempting to start before game has been fully initialized.")
		os.Exit(1)
	}
	if g.started {
		fmt.Printf("Attempting to start a game already in progress.")
		os.Exit(1)
	}
	numPlayers := len(g.players)
	if numPlayers >= len(cardsInHand) || cardsInHand[numPlayers] == 0 {
		fmt.Printf("Attempted to start game with invalid number of players.")
		os.Exit(1)
	}

	// create hands
	for _, player := range g.players {
		for i := 0; i < cardsInHand[numPlayers]; i++ {
			player.AddCard(g.DrawCard())
		}
	}

	// let's do it
	g.currentPlayer = rand.Intn(numPlayers)
	g.started = true
}

func (g *Game) ProcessMove(move string) bool {
	// TODO: parse JSON
	// TODO: more checking that it's a valid move

	var player, moveType, cardIndex, hintPlayer, hintCard, hintInfoType int // will be replaced
	card := g.players[player].GetCard(cardIndex)

	if moveType == movePlay {
		g.players[player].RemoveCard(player)

		if g.PlayCard(card) {
			// play was successful!
			if g.PilesComplete() {
				g.Win()
			}
		} else {
			// play was unsuccessful :(
			g.bombs--
			if g.bombs == 0 {
				g.Lose()
			}
			g.discard = append(g.discard, card)
		}
	} else if moveType == moveDiscard {
		g.players[player].RemoveCard(cardIndex)
		g.discard = append(g.discard, card)
		g.hints++
		if g.hints > maxHints {
			g.hints = maxHints
		}
	} else {
		if g.hints <= 0 {
			return false
		}
		g.players[hintPlayer].ReceiveHint(hintCard, hintInfoType)
		g.hints--
	}

	if moveType == movePlay || moveType == moveDiscard {
		if len(g.deck) > 0 {
			g.players[player].AddCard(g.DrawCard())
		}
	}

	// TODO: log move (if it's valid)
	// return whether it worked or not
	return true
}

func (g *Game) CreateState(p Player) string {
	// TODO: make JSON to send to a certain player about the state of the world
	// aka don't send them their hand
	return ""
}

func (g *Game) Win() {
	// TODO
}

func (g *Game) Lose() {
	// TODO
}
