package lib

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/NaySoftware/go-fcm"
)

type Game struct {
	ID         string
	Name       string
	Players    []Player
	Public     bool
	IgnoreTime bool
	SighButton bool

	State          int
	StartTime      int64
	LastUpdateTime int64
	Mode           int
	CurrentScore   int
	Table          *Table
	Score          int

	Stats StatLog
}

func (g *Game) Initialize(public bool, ignoreTime bool, sighButton bool, gameMode int) string {
	g.State = StateNotStarted
	g.Table = new(Table)

	// validate input
	if gameMode != ModeNormal && gameMode != ModeRainbow && gameMode != ModeWildcard && gameMode != ModeHard && gameMode != ModeRainbowLimited {
		gameMode = ModeNormal
	}

	// figure out how many cards are in the Deck
	if gameMode == ModeNormal {
		g.Table.Colors = normalColors[:]
	} else {
		g.Table.Colors = rainbowColors[:]
	}
	g.Mode = gameMode
	maxCards := g.Table.MaxCards()

	// populate the Deck, Discard, and Piles
	g.Table.Deck = make([]Card, maxCards, maxCards)
	g.Table.PopulateDeck()
	g.Table.Discard = make([]Card, 0, maxCards)
	g.Table.Piles = make([]int, len(g.Table.Colors), len(g.Table.Colors))

	g.Table.BombsLeft = StartingBombs
	g.Table.HintsLeft = StartingHints

	g.Table.Mode = gameMode
	g.Table.NumPlayers = 0

	// set starting values
	g.Public = public
	g.IgnoreTime = ignoreTime
	g.SighButton = sighButton
	g.Mode = gameMode
	g.Table.TurnsLeft = -1
	g.Table.Turn = 0
	g.LastUpdateTime = -1

	// start with no Players
	g.Players = make([]Player, 0, len(cardsInHand)-1)
	g.Table.HighestPossibleScore = g.GetHighestPossibleScore()
	return ""
}

func (g *Game) AddPlayer(id string, name string) string {
	if g.State != StateNotStarted {
		return "Attempting to add players after game has started."
	}
	if len(g.Players) >= MaxPlayers {
		return "Attempted to add a player to a full game."
	}

	g.Players = append(g.Players, Player{GoogleID: id, Name: name})
	g.Table.NumPlayers++
	g.Table.Turn++
	return ""
}

func (g *Game) Start() string {
	if g.State != StateNotStarted {
		return "Attempting to start a game that has already been started."
	}

	numPlayers := len(g.Players)
	if numPlayers >= len(cardsInHand) || cardsInHand[numPlayers] == 0 {
		return "Attempted to start game with invalid number of players."
	}
	g.Table.NumPlayers = numPlayers

	if len(g.Table.Deck) <= 0 {
		log.Fatal("Drawing card on empty deck!")
	}

	// create hands
	for index, _ := range g.Players {
		g.Players[index].Initialize(cardsInHand[numPlayers])
		for i := 0; i < cardsInHand[numPlayers]; i++ {
			err := g.Players[index].AddCard(g.Table.DrawCard())
			if err != "" {
				return "Error initializing player's hand: " + err
			}
		}
	}

	// let's do it
	g.Table.CurrentPlayerIndex = rand.Intn(numPlayers)
	g.State = StateStarted
	g.StartTime = time.Now().Unix()
	g.LastUpdateTime = g.StartTime
	g.Table.Turn++
	g.SendCurrentPlayerNotification()

	return ""
}

//Wrapper in case we ever need a global time stamp to coordinate amongst distributed servers
func getCurrentTime() int64 {
	return time.Now().Unix()
}

func (g *Game) ProcessAnnouncement(mp *Message) string {
	m := *mp
	if g.State != StateStarted {
		return "Attempting to process an announcement for a non-ongoing game."
	}
	p := g.GetPlayerByGoogleID(m.Player)
	if p == nil {
		return "Attempting to process an announcement for a nonexistent player."
	}

	// TODO: protect against code injection

	// make announcement:
	p.LastMove = ": " + m.Announcement
	g.LastUpdateTime = getCurrentTime()

	// success:
	return ""
}

func (g *Game) ProcessMove(mp *Message) string {
	m := *mp

	if m.MoveType == MoveHint && g.Table.HintsLeft <= 0 {
		return "There are no hints left. Discard to earn more hints."
	}
	if g.State != StateStarted {
		return "Attempting to process a move for a non-ongoing game."
	}
	if m.Player != g.Players[g.Table.CurrentPlayerIndex].GoogleID {
		return "Attempting to process a move for out-of-turn player."
	}
	p := g.GetPlayerByGoogleID(m.Player)
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
		if g.Table.PlayCard(card) {
			// play was successful!
			mp.Result = ResultPlay
			if card.Number == 5 {
				g.Table.HintsLeft++
				if g.Table.HintsLeft > MaxHints {
					g.Table.HintsLeft = MaxHints
				}
			}
			if g.Table.ArePilesComplete() {
				g.State = StatePerfect
			}
			p.LastMove = "played " + card.Color + " " + strconv.Itoa(card.Number)
		} else {
			// play was unsuccessful :(
			mp.Result = ResultBomb
			g.Table.BombsLeft--
			if g.Table.BombsLeft == 0 {
				g.State = StateBombedOut
			}
			g.Table.Discard = append(g.Table.Discard, card)
			p.LastMove = "bombed " + card.Color + " " + strconv.Itoa(card.Number)
		}
	} else if m.MoveType == MoveDiscard {
		card, err := p.RemoveCard(m.CardIndex)
		if err != "" {
			return "Error removing card from player's hand to discard: " + err
		}
		cardsModified = append(cardsModified, card.ID)
		g.Table.Discard = append(g.Table.Discard, card)
		g.Table.HintsLeft++
		if g.Table.HintsLeft > MaxHints {
			g.Table.HintsLeft = MaxHints
		}
		p.LastMove = "discarded " + card.Color + " " + strconv.Itoa(card.Number)
	} else if m.MoveType == MoveHint {
		if g.Table.HintsLeft <= 0 {
			return "Attempting to hint with no hints remaining."
		}
		hintReceiver := g.GetPlayerByGoogleID(m.HintPlayer)
		if hintReceiver == nil {
			return "Attempting to give hint to a nonexistent player."
		}
		cardsHinted, err := hintReceiver.ReceiveHint(m.CardIndex, m.HintInfoType, m.HintColor, g.Mode)
		if err != "" {
			return "Error giving hint: " + err
		}
		cardsModified = append(cardsModified, cardsHinted...)
		g.Table.HintsLeft--

		p.LastMove = "➡ " + hintReceiver.Name + " "
		hintedCard := hintReceiver.GetCardByID(cardsModified[0])
		if m.HintInfoType == HintNumber {
			p.LastMove += strconv.Itoa(hintedCard.Number) + "s"
		} else if (g.Mode == ModeWildcard || g.Mode == ModeHard) && hintedCard.Color == "rainbow" {
			p.LastMove += mp.HintColor + "s"
		} else {
			p.LastMove += hintedCard.Color + "s"
		}

	} else {
		return "Attempting to process unknown move type."
	}

	if m.MoveType == MovePlay || m.MoveType == MoveDiscard {
		if len(g.Table.Deck) > 0 {
			drawnCard := g.Table.DrawCard()
			err := p.AddCard(drawnCard)
			if err != "" {
				return "Error drawing card: " + err
			}
			cardsModified = append(cardsModified, drawnCard.ID)
		}
	}

	g.Table.Turn++
	if g.Table.TurnsLeft > 0 {
		g.Table.TurnsLeft--
	}
	if g.State == StateStarted && g.Table.TurnsLeft == 0 {
		g.State = StateDeckEmpty
	}

	if len(g.Table.Deck) == 0 && g.Table.TurnsLeft == -1 {
		// Deck is empty, start the countdown
		g.Table.TurnsLeft = len(g.Players)
	}

	g.Table.CurrentPlayerIndex = (g.Table.CurrentPlayerIndex + 1) % len(g.Players)
	g.SendCurrentPlayerNotification()
	g.Table.CardsLastModified = cardsModified

	if g.State == StateStarted && !g.AnyPlayableCards() {
		g.State = StateNoPlays
	}

	g.CurrentScore = g.Table.Score()
	g.Table.HighestPossibleScore = g.GetHighestPossibleScore()

	return ""
}

func (g *Game) CreateState(playerid string) Game {
	p := g.GetPlayerByGoogleID(playerid)

	gCopy := Game{}
	gCopy = *g

	// clear the Deck (could be used to determine your hand)

	tCopy := Table{}
	tCopy = *g.Table
	gCopy.Table = &tCopy

	gCopy.Table.CardsLeft = len(gCopy.Table.Deck)
	gCopy.Table.Deck = make([]Card, 0, 0)

	// clear your hand, except for revealed info
	newPlayers := make([]Player, len(g.Players), len(g.Players))
	for playerIndex, player := range gCopy.Players {
		if p.GoogleID == player.GoogleID {
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

func (g *Game) GetPlayerByGoogleID(id string) *Player {
	var p *Player
	if g.Players == nil {
		return p
	}
	for index, _ := range g.Players {
		if g.Players[index].GoogleID == id {
			return &g.Players[index]
		}
	}
	return p
}

func (g *Game) SendCurrentPlayerNotification() {
	token := g.GetPlayerByGoogleID(g.Players[g.Table.CurrentPlayerIndex].GoogleID).PushToken
	if token == "" {
		return
	}

	data := map[string]string{
		"msg": "Other players are waiting! Take your turn.",
		"sum": "Fireworks - it's your turn!",
	}

	c := fcm.NewFcmClient(PushServerKey)
	c.NewFcmRegIdsMsg([]string{token}, data)

	n := fcm.NotificationPayload{}
	n.Title = "Fireworks - it's your turn!"
	n.Body = "Other players are waiting! Take your turn."
	n.ClickAction = "https://ryanschoen.com/fireworks/#!/games/" + g.ID
	n.Icon = "https://ryanschoen.com/fireworks/images/icons/fireworks128.png"
	c.SetNotificationPayload(&n)

	_, err := c.Send()

	if err != nil {
		fmt.Println(err)
	}
}

func (g *Game) GetPlayerListAsString() string {
	playerString := ""
	for _, player := range g.Players {
		playerString += "'" + player.GoogleID + "'',"
	}
	return playerString[:-len(playerString)-1]
}

func (g *Game) AnyPlayableCards() bool {
	if g.Table.Turn < 15 {
		// not possible for there to be no playable cards yet
		return true
	}

	for i, _ := range g.Players {
		if g.Table.TurnsLeft != -1 && i >= g.Table.TurnsLeft {
			break
		}
		p := g.Players[(i+g.Table.CurrentPlayerIndex)%len(g.Players)]
		for _, c := range p.Cards {
			if g.Table.CardPlayableOnPile(c) > -1 {
				return true
			}
		}
	}

	for _, c := range g.Table.Deck {
		if g.Table.CardPlayableOnPile(c) > -1 {
			return true
		}
	}

	return false
}

func (g *Game) GetHighestPossibleScore() int {
	score := g.Table.Score()
	cards := make([]Card, 0, g.Table.MaxCards())
	cards = append(cards, g.Table.Deck...)
	for _, player := range g.Players {
		cards = append(cards, player.Cards...)
	}

	piles := make([]int, len(g.Table.Piles), len(g.Table.Piles))
	for i, value := range g.Table.Piles {
		piles[i] = value
	}

	turnsLeft := len(g.Table.Deck) + len(g.Players)
	if g.Table.TurnsLeft > -1 && turnsLeft > g.Table.TurnsLeft {
		turnsLeft = g.Table.TurnsLeft
	}

MainLoop:
	for turnsLeft > 0 {
		for _, c := range cards {
			pile := g.Table.CardPlayableOnCustomPile(c, piles)
			if pile > -1 {
				piles[pile]++
				score++
				turnsLeft--
				if turnsLeft == 0 {
					break MainLoop
				}
				continue MainLoop
			}
		}
		break
	}

	return score
}
