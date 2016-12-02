package lib

import (
	"math/rand"
)

type Game struct {
	GameID      string
	Players     []Player
	Initialized bool

	Hints              int
	Bombs              int
	Deck               []Card
	Discard            []Card
	Piles              []int
	CurrentPlayerIndex int
	CurrentPlayer      string
	StartingTime       int
	State              int
	TurnsLeft          int
	CardsLeft          int
	CardsLastModified  []int
}

func (g *Game) Initialize() string {
	g.State = StateNotStarted

	// figure out how many cards are in the Deck
	maxCards := 0
	for _, count := range numbers {
		maxCards += count * len(colors)
	}

	// populate the Deck, Discard, and Piles
	g.Deck = make([]Card, maxCards, maxCards)
	g.PopulateDeck()
	g.Discard = make([]Card, 0, maxCards)
	g.Piles = make([]int, len(colors), len(colors))

	// set starting values
	g.Hints = startingHints
	g.Bombs = startingBombs
	g.TurnsLeft = -1

	// start with no Players
	g.Players = make([]Player, 0, len(cardsInHand)-1)
	g.Initialized = true
	return ""
}

func (g *Game) AddPlayer(id string) string {
	if !g.Initialized {
		return "Attempting to add player before game has been fully initialized."
	}
	if g.State != StateNotStarted {
		return "Attempting to add players after game has started."
	}
	if len(g.Players) >= len(cardsInHand)-1 {
		return "Attempted to add a player to a full game."
	}

	g.Players = append(g.Players, Player{ID: id})
	return ""
}

func (g *Game) Start() string {
	if !g.Initialized {
		return "Attempting to start before game has been fully Initialized."
	}
	if g.State != StateNotStarted {
		return "Attempting to start a game that has already been started."
	}

	numPlayers := len(g.Players)
	if numPlayers >= len(cardsInHand) || cardsInHand[numPlayers] == 0 {
		return "Attempted to start game with invalid number of players."
	}

	// create hands
	for index, _ := range g.Players {
		g.Players[index].Initialize(cardsInHand[numPlayers])
		for i := 0; i < cardsInHand[numPlayers]; i++ {
			err := g.Players[index].AddCard(g.DrawCard())
			if err != "" {
				return "Error initializing player's hand: " + err
			}
		}
	}

	// let's do it
	g.CurrentPlayerIndex = rand.Intn(numPlayers)
	g.CurrentPlayer = g.Players[g.CurrentPlayerIndex].ID
	g.State = StateStarted

	return ""
}

func (g *Game) ProcessMove(m Message) string {
	if g.State != StateStarted {
		return "Attempting to process a move for a non-ongoing game."
	}
	if m.Player != g.CurrentPlayer {
		return "Attempting to process a move for out-of-turn player."
	}
	p := g.GetPlayerByID(m.Player)
	if p == nil {
		return "Attempting to process a move for a nonexistent player."
	}

	// TODO: more checking that it's a valid move

	var cardsModified []int

	if m.MoveType == MovePlay {
		card, err := p.RemoveCard(m.CardIndex)
		if err != "" {
			return "Error removing card from player's hand to play: " + err
		}
		cardsModified = append(cardsModified, card.ID)
		if g.PlayCard(card) {
			// play was successful!
			if card.Number == 5 {
				g.Hints++
				if g.Hints > maxHints {
					g.Hints = maxHints
				}
			}
			if g.PilesComplete() {
				g.State = StatePerfect
			}
		} else {
			// play was unsuccessful :(
			g.Bombs--
			if g.Bombs == 0 {
				g.State = StateBombedOut
			}
			g.Discard = append(g.Discard, card)
		}
	} else if m.MoveType == MoveDiscard {
		card, err := p.RemoveCard(m.CardIndex)
		if err != "" {
			return "Error removing card from player's hand to discard: " + err
		}
		cardsModified = append(cardsModified, card.ID)
		g.Discard = append(g.Discard, card)
		g.Hints++
		if g.Hints > maxHints {
			g.Hints = maxHints
		}
	} else if m.MoveType == MoveHint {
		if g.Hints <= 0 {
			return "Attempting to hint with no hints remaining."
		}
		hintReceiver := g.GetPlayerByID(m.HintPlayer)
		if hintReceiver == nil {
			return "Attempting to give hint to a nonexistent player."
		}
		cardsHinted, err := hintReceiver.ReceiveHint(m.CardIndex, m.HintInfoType)
		if err != "" {
			return "Error giving hint: " + err
		}
		cardsModified = append(cardsModified, cardsHinted...)
		g.Hints--
	} else {
		return "Attempting to process unknown move type."
	}

	if m.MoveType == MovePlay || m.MoveType == MoveDiscard {
		if len(g.Deck) > 0 {
			drawnCard := g.DrawCard()
			err := p.AddCard(drawnCard)
			if err != "" {
				return "Error drawing card: " + err
			}
			cardsModified = append(cardsModified, drawnCard.ID)
		} else if g.TurnsLeft == -1 {
			// Deck is empty, start the countdown
			g.TurnsLeft = len(g.Players)
		}
	}

	g.CurrentPlayerIndex = (g.CurrentPlayerIndex + 1) % len(g.Players)
	g.CurrentPlayer = g.Players[g.CurrentPlayerIndex].ID
	g.CardsLastModified = cardsModified

	if g.TurnsLeft == 0 {
		g.State = StateDeckEmpty
	}
	if g.TurnsLeft > 0 {
		g.TurnsLeft--
	}

	// TODO: log move (if it's valid)
	return ""
}

func (g *Game) CreateState(playerid string) Game {
	p := g.GetPlayerByID(playerid)

	gCopy := Game{}
	gCopy = *g

	// clear the Deck (could be used to determine your hand)
	gCopy.CardsLeft = len(gCopy.Deck)
	gCopy.Deck = make([]Card, 0, 0)

	// clear your hand, except for revealed info
	newPlayers := make([]Player, len(g.Players), len(g.Players))
	for playerIndex, player := range gCopy.Players {
		if p.ID == player.ID {
			newHand := make([]Card, len(player.Cards), len(player.Cards))
			for cardIndex, card := range player.Cards {
				card.Color = ""
				card.Number = 0
				newHand[cardIndex] = card
			}
			player.Cards = newHand
		}
		newPlayers[playerIndex] = player
	}
	gCopy.Players = newPlayers
	return gCopy
}
