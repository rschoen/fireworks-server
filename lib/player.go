package lib

import (
	"fmt"
	"os"
)

type Player struct {
	ID    string
	Cards []Card
}

func (p *Player) Initialize(maxCards int) {
	p.Cards = make([]Card, 0, maxCards)
}

func (p *Player) ReceiveHint(i int, infoType int) {
	var card = p.GetCard(i)
	var number = card.Number
	var color = card.Color
	for index, _ := range p.Cards {
		if infoType == infoNumber && p.Cards[index].Number == number {
			p.Cards[index].KnownNumber = number
		} else if infoType == infoColor && p.Cards[index].Color == color {
			p.Cards[index].KnownColor = color
		}
	}
}

func (p *Player) GetCard(i int) Card {
	if i >= len(p.Cards) {
		fmt.Printf("Referenced a non-existent card in a player's hand.")
		os.Exit(1)
	}
	return p.Cards[i]
}

func (p *Player) AddCard(c Card) {
	if len(p.Cards) == cap(p.Cards) {
		fmt.Printf("Attempted to add a card beyond hand capacity.")
	}
	p.Cards = append(p.Cards, c)
}

func (p *Player) RemoveCard(i int) Card {
	if i >= len(p.Cards) {
		fmt.Printf("Attempted to remove a non-existent card.")
	}
	var removedCard = p.Cards[i]
	p.Cards = append(p.Cards[:i], p.Cards[i+1:]...)
	return removedCard
}

func (g *Game) GetPlayerByID(id string) *Player {
	var p *Player
	if g.Players == nil {
		return p
	}
	for index, _ := range g.Players {
		if g.Players[index].ID == id {
			return &g.Players[index]
		}
	}
	return p
}
