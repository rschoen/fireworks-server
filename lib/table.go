package lib

// game.go was getting big, so I separated some stuff into here

import (
	"math/rand"
)

func (g *Game) PopulateDeck() {
	i := 0
	for number, count := range numbers {
		for _, color := range g.Colors {
			if g.Mode == ModeHard && color == "rainbow" && count > 0 {
				count = 1
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
func (g *Game) CardPlayableOnCustomPile(c Card, p []int) int {
	for index, count := range p {
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

func (g *Game) CardPlayableOnPile(c Card) int {
	return g.CardPlayableOnCustomPile(c, g.Piles)
}

func (g *Game) AnyPlayableCards() bool {
	if g.Turn < 15 {
		// not possible for there to be no playable cards yet
		return true
	}

	for i, _ := range g.Players {
		if g.TurnsLeft != -1 && i >= g.TurnsLeft {
			break
		}
		p := g.Players[(i+g.CurrentPlayerIndex)%len(g.Players)]
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

func (g *Game) MaxCards() int {
	maxCards := 0
	for _, count := range numbers {
		maxCards += count * len(g.Colors)
	}
	if g.Mode == ModeHard {
		maxCards -= 5
	}
	return maxCards
}

func (g *Game) GetHighestPossibleScore() int {
	score := g.Score()
	cards := make([]Card, 0, g.MaxCards())
	cards = append(cards, g.Deck...)
	for _, player := range g.Players {
		cards = append(cards, player.Cards...)
	}

	piles := make([]int, len(g.Piles), len(g.Piles))
	for i, value := range g.Piles {
		piles[i] = value
	}

	turnsLeft := len(g.Deck) + len(g.Players)
	if g.TurnsLeft > -1 && turnsLeft > g.TurnsLeft {
		turnsLeft = g.TurnsLeft
	}

MainLoop:
	for turnsLeft > 0 {
		for _, c := range cards {
			pile := g.CardPlayableOnCustomPile(c, piles)
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
