package lib

// game.go was getting big, so I separated some stuff into here

import (
	"math/rand"
)

func (g *Game) PopulateDeck() {
	i := 0
	for number, count := range numbers {
		for _, color := range g.Colors {
			if g.Mode == ModeHard {
				count = 1;
			}
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
	pile := g.CardPlayableOnPile(c)
	if pile > -1 {
		g.Piles[pile]++
		g.PileCards = append(g.PileCards, c)
		return true
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

func (g *Game) Score() int {
	score := 0
	for _, count := range g.Piles {
		score += count
	}
	return score
}

// returns pile index if playable, -1 if not
func (g *Game) CardPlayableOnPile(c Card) int {
	for index, count := range g.Piles {
		if g.Colors[index] == c.Color {
			if count+1 == c.Number {
				return index 
			} else {
				return -1
			}
		}
	}
	return -1
}

func (g *Game) AnyPlayableCards() bool {
	if g.Turn < 15 {
		// not possible for there to be no playable cards yet
		return true
	}

	for _, p := range g.Players {
		for _, c := range p.Cards {
			if g.CardPlayableOnPile(c) > -1 {
				return true
			}
		}
	}

	for _, c := range g.Deck {
		if g.CardPlayableOnPile(c) > -1 {
			return true
		}
	}

	return false
}
