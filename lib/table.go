package lib

// game.go was getting big, so I separated some stuff into here

import (
	"math/rand"
)

func (g *Game) PopulateDeck() {
	i := 0
	for number, count := range numbers {
		for _, color := range colors {
			for j := 0; j < count; j++ {
				g.Deck[i].ID = i
				g.Deck[i].Color = color
				g.Deck[i].Number = number
				i++
			}
		}
	}
}

func (g *Game) DrawCard() Card {
	index := rand.Intn(len(g.Deck))
	card := g.Deck[index]
	g.Deck = append(g.Deck[:index], g.Deck[index+1:]...)
	return card
}

func (g *Game) PlayCard(c Card) bool {
	for index, count := range g.Piles {
		if colors[index] == c.Color {
			if count+1 == c.Number {
				// good play!
				g.Piles[index]++
				g.PileCards = append(g.PileCards, c)
				return true
			} else {
				return false
			}
		}
	}
	return false
}

func (g *Game) PilesComplete() bool {
	for _, count := range g.Piles {
		if count != len(numbers)-1 {
			return false
		}
	}
	return true
}
