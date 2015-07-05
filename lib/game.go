package lib

import (
	"fmt"
	"math/rand"
	"os"
)

type Game struct {
	GameID      string
	players     []Player
	Initialized bool
	Started     bool

	hints         int
	bombs         int
	deck          []Card
	discard       []Card
	piles         []int
	currentPlayer int
	startingTime  int
	finished      bool
	won           bool
	turnsLeft     int
}

func (g *Game) Initialize() bool {
	if g.finished {
		fmt.Printf("Attempting to initialize game after game has ended.")
		return false
	}

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
	g.turnsLeft = -1

	// start with no players
	g.players = make([]Player, 0, len(cardsInHand)-1)
	g.Initialized = true
	fmt.Println("Game initialized.\n")
	return true
}

func (g *Game) AddPlayer(id string) bool {
	if !g.Initialized {
		fmt.Printf("Attempting to add player before game has been fully Initialized.")
		return false
	}
	if g.Started {
		fmt.Printf("Attempting to add player after game has Started.")
		return false
	}
	if g.finished {
		fmt.Printf("Attempting to add players after game has ended.")
		return false
	}
	if len(g.players) >= len(cardsInHand)-1 {
		fmt.Printf("Attempted to add a player to a full game.")
		return false
	}

	g.players = append(g.players, Player{id: id})
	return true
}

func (g *Game) Start() {
	if !g.Initialized {
		fmt.Printf("Attempting to start before game has been fully Initialized.")
		os.Exit(1)
	}
	if g.Started {
		fmt.Printf("Attempting to start a game already in progress.")
		os.Exit(1)
	}
	if g.finished {
		fmt.Printf("Attempting to start a game after it has ended.")
		os.Exit(1)
	}
	numPlayers := len(g.players)
	if numPlayers >= len(cardsInHand) || cardsInHand[numPlayers] == 0 {
		fmt.Printf("Attempted to start game with invalid number of players.")
		os.Exit(1)
	}

	// create hands
	for _, player := range g.players {
		player.Initialize(cardsInHand[numPlayers])
		for i := 0; i < cardsInHand[numPlayers]; i++ {
			player.AddCard(g.DrawCard())
		}
	}

	// let's do it
	g.currentPlayer = rand.Intn(numPlayers)
	g.Started = true
}

func (g *Game) ProcessMove(m Message) bool {
	if !g.Started {
		fmt.Printf("Attempting to process move for a game that hasn't Started yet.")
		os.Exit(1)
	}
	if g.finished {
		fmt.Printf("Attempting to process a move for a finished game.")
		return false
	}
	p := g.GetPlayerByID(m.Player)
	if p == nil {
		fmt.Printf("Attempting to process a move for a nonexistent player.")
		return false
	}

	// TODO: more checking that it's a valid move

	card := p.GetCard(m.CardIndex)

	if m.MoveType == MovePlay {
		p.RemoveCard(m.CardIndex)

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
	} else if m.MoveType == MoveDiscard {
		p.RemoveCard(m.CardIndex)
		g.discard = append(g.discard, card)
		g.hints++
		if g.hints > maxHints {
			g.hints = maxHints
		}
	} else if m.MoveType == MoveHint {
		if g.hints <= 0 {
			return false
		}
		hintReceiver := g.GetPlayerByID(m.HintPlayer)
		if hintReceiver == nil {
			fmt.Printf("Attempting to give hint to a nonexistent player.")
			return false
		}
		hintReceiver.ReceiveHint(m.HintCard, m.HintInfoType)
		g.hints--
	} else {
		fmt.Printf("Attempting to process unknown move type.")
		return false
	}

	if m.MoveType == MovePlay || m.MoveType == MoveDiscard {
		if len(g.deck) > 0 {
			p.AddCard(g.DrawCard())
		} else if g.turnsLeft == -1 {
			// deck is empty, start the countdown
			g.turnsLeft = len(g.players)
		}
	}

	if g.turnsLeft == 0 {
		g.Lose()
	}

	// TODO: log move (if it's valid)
	return true
}

func (g *Game) CreateState(playerid string) Game {
	p := g.GetPlayerByID(playerid)

	gCopy := Game{}
	gCopy = *g

	// clear the deck (could be used to determine your hand)
	gCopy.deck = make([]Card, 0, 0)

	// clear your hand, except for revealed info
	for _, player := range gCopy.players {
		if p.id == player.id {
			for _, card := range player.cards {
				card.Color = ""
				card.Number = 0
			}
		}
	}

	return gCopy
}

func (g *Game) Win() {
	g.finished = true
	g.won = true
}

func (g *Game) Lose() {
	g.finished = true
	g.won = false
}
