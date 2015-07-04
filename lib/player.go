package lib

import (
	"fmt"
	"os"
)

type Player struct {
	id    string
	cards []Card
}

func (p *Player) Initialize(maxCards int) {
	p.cards = make([]Card, 0, maxCards)
}

func (p *Player) ReceiveHint(i int, infoType int) {
	var card = p.GetCard(i)
	var number = card.Number
	var color = card.Color
	for _, potentialMatch := range p.cards {
		if infoType == infoNumber && potentialMatch.Number == number {
			potentialMatch.RevealedNumber = number
		} else if infoType == infoColor && potentialMatch.Color == color {
			potentialMatch.RevealedColor = color
		}
	}
}

func (p *Player) GetCard(i int) Card {
	if i >= len(p.cards) {
		fmt.Printf("Referenced a non-existent card in a player's hand.")
		os.Exit(1)
	}
	return p.cards[i]
}

func (p *Player) AddCard(c Card) {
	if len(p.cards) == cap(p.cards) {
		fmt.Printf("Attempted to add a card beyond hand capacity.")
	}
	p.cards = append(p.cards, c)
}

func (p *Player) RemoveCard(i int) Card {
	if i >= len(p.cards) {
		fmt.Printf("Attempted to remove a non-existent card.")
	}
	var removedCard = p.cards[i]
	p.cards = append(p.cards[:i], p.cards[i+1:]...)
	return removedCard
}

func (g *Game) GetPlayerByID(id string) Player {
	var p Player
	for _, player := range g.players {
		if player.id == id {
			return player
		}
	}
	return p
}
