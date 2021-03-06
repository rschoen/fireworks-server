package lib

import (
	"fmt"
	"github.com/NaySoftware/go-fcm"
	"math/rand"
	"strconv"
	"time"
)

type Game struct {
	ID          string
	Name        string
	Players     []Player
	Initialized bool
	Public      bool
	IgnoreTime  bool
	SighButton  bool

	Hints                int
	MaxHints             int
	Bombs                int
	Deck                 []Card
	Discard              []Card
	Piles                []int
	PileCards            []Card
	CurrentPlayerIndex   int
	CurrentPlayer        string
	State                int
	UpdateTime           int64
	Turn                 int
	TurnsLeft            int
	CardsLeft            int
	CardsLastModified    []int
	Type                 int
	StartTime            int64
	Mode                 int
	Colors               []string
	CurrentScore         int
	HighestPossibleScore int
}

func (g *Game) Initialize(public bool, ignoreTime bool, sighButton bool, gameMode int, startingHints int, maxHints int, startingBombs int) string {
	g.State = StateNotStarted

	// validate input

	if maxHints < 1 {
		maxHints = DefaultMaxHints
	}
	if startingHints < 1 || startingHints > maxHints {
		startingHints = maxHints
	}
	if startingBombs < 1 {
		startingBombs = DefaultStartingBombs
	}

	if gameMode != ModeNormal && gameMode != ModeRainbow && gameMode != ModeWildcard && gameMode != ModeHard && gameMode != ModeRainbowLimited {
		gameMode = ModeNormal
	}

	// figure out how many cards are in the Deck
	if gameMode == ModeNormal {
		g.Colors = normalColors[:]
	} else {
		g.Colors = rainbowColors[:]
	}
	g.Mode = gameMode
	maxCards := g.MaxCards()

	// populate the Deck, Discard, and Piles
	g.Deck = make([]Card, maxCards, maxCards)
	g.PopulateDeck()
	g.Discard = make([]Card, 0, maxCards)
	g.Piles = make([]int, len(g.Colors), len(g.Colors))

	// set starting values
	g.Public = public
	g.IgnoreTime = ignoreTime
	g.SighButton = sighButton
	g.Mode = gameMode
	g.Hints = startingHints
	g.MaxHints = maxHints
	g.Bombs = startingBombs
	g.TurnsLeft = -1
	g.Turn = 0
	g.UpdateTime = -1

	// start with no Players
	g.Players = make([]Player, 0, len(cardsInHand)-1)
	g.Initialized = true
	g.HighestPossibleScore = g.GetHighestPossibleScore()
	return ""
}

func (g *Game) AddPlayer(id string, name string) string {
	if !g.Initialized {
		return "Attempting to add player before game has been fully initialized."
	}
	if g.State != StateNotStarted {
		return "Attempting to add players after game has started."
	}
	if len(g.Players) >= MaxPlayers {
		return "Attempted to add a player to a full game."
	}

	g.Players = append(g.Players, Player{GoogleID: id, Name: name})
	g.Turn++
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
	g.CurrentPlayer = g.Players[g.CurrentPlayerIndex].GoogleID
	g.State = StateStarted
	g.StartTime = time.Now().Unix()
	g.Turn++
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
	g.UpdateTime = getCurrentTime()

	// success:
	return ""
}

func (g *Game) ProcessMove(mp *Message) string {
	m := *mp
	if g.State != StateStarted {
		return "Attempting to process a move for a non-ongoing game."
	}
	if m.Player != g.CurrentPlayer {
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
		if g.PlayCard(card) {
			// play was successful!
			mp.Result = ResultPlay
			if card.Number == 5 {
				g.Hints++
				if g.Hints > g.MaxHints {
					g.Hints = g.MaxHints
				}
			}
			if g.PilesComplete() {
				g.State = StatePerfect
			}
			p.LastMove = "played " + card.Color + " " + strconv.Itoa(card.Number)
		} else {
			// play was unsuccessful :(
			mp.Result = ResultBomb
			g.Bombs--
			if g.Bombs == 0 {
				g.State = StateBombedOut
			}
			g.Discard = append(g.Discard, card)
			p.LastMove = "bombed " + card.Color + " " + strconv.Itoa(card.Number)
		}
	} else if m.MoveType == MoveDiscard {
		card, err := p.RemoveCard(m.CardIndex)
		if err != "" {
			return "Error removing card from player's hand to discard: " + err
		}
		cardsModified = append(cardsModified, card.ID)
		g.Discard = append(g.Discard, card)
		g.Hints++
		if g.Hints > g.MaxHints {
			g.Hints = g.MaxHints
		}
		p.LastMove = "discarded " + card.Color + " " + strconv.Itoa(card.Number)
	} else if m.MoveType == MoveHint {
		if g.Hints <= 0 {
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
		g.Hints--

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
		if len(g.Deck) > 0 {
			drawnCard := g.DrawCard()
			err := p.AddCard(drawnCard)
			if err != "" {
				return "Error drawing card: " + err
			}
			cardsModified = append(cardsModified, drawnCard.ID)
		}
	}

	g.Turn++
	if g.TurnsLeft > 0 {
		g.TurnsLeft--
	}
	if g.State == StateStarted && g.TurnsLeft == 0 {
		g.State = StateDeckEmpty
	}

	if len(g.Deck) == 0 && g.TurnsLeft == -1 {
		// Deck is empty, start the countdown
		g.TurnsLeft = len(g.Players)
	}

	g.CurrentPlayerIndex = (g.CurrentPlayerIndex + 1) % len(g.Players)
	g.CurrentPlayer = g.Players[g.CurrentPlayerIndex].GoogleID
	g.SendCurrentPlayerNotification()
	g.CardsLastModified = cardsModified

	if g.State == StateStarted && !g.AnyPlayableCards() {
		g.State = StateNoPlays
	}

	g.CurrentScore = g.Score()
	g.HighestPossibleScore = g.GetHighestPossibleScore()

	return ""
}

func (g *Game) CreateState(playerid string) Game {
	p := g.GetPlayerByGoogleID(playerid)

	gCopy := Game{}
	gCopy = *g

	// clear the Deck (could be used to determine your hand)
	gCopy.CardsLeft = len(gCopy.Deck)
	gCopy.Deck = make([]Card, 0, 0)

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
	token := g.GetPlayerByGoogleID(g.CurrentPlayer).PushToken
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
