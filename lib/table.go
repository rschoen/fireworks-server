package lib

// game.go was getting big, so I separated some stuff into here

import (
	"log"
	"math/rand"
)

type Table struct {
	HintsLeft         int
	BombsLeft         int
	Deck              []Card
	Discard           []Card
	Piles             []int
	PileCards         []Card
	CardsLastModified []int
	Colors            []string

	CurrentPlayerIndex   int
	Turn                 int
	TurnsLeft            int
	CardsLeft            int
	HighestPossibleScore int
	NumPlayers           int
	Mode                 int
}

func (t *Table) PopulateDeck() {
	i := 0
	for number, count := range numbers {
		for _, color := range t.Colors {
			if (t.Mode == ModeHard || t.Mode == ModeRainbowLimited) && color == ColorRainbow && count > 0 {
				count = 1
			}
			for j := 0; j < count; j++ {
				t.Deck[i].ID = i
				t.Deck[i].Color = color
				t.Deck[i].Number = number
				i++
			}
		}
	}
}

func (t *Table) DrawCard() Card {
	if len(t.Deck) <= 0 {
		log.Fatal("Attempting to draw card from empty deck!")
	}
	index := rand.Intn(len(t.Deck))
	card := t.Deck[index]
	t.Deck = append(t.Deck[:index], t.Deck[index+1:]...)
	return card
}

func (t *Table) PlayCard(c Card) bool {
	pile := t.CardPlayableOnPile(c)
	if pile > -1 {
		t.Piles[pile]++
		t.PileCards = append(t.PileCards, c)
		return true
	}
	return false
}

func (t *Table) ArePilesComplete() bool {
	for _, count := range t.Piles {
		if count != len(numbers)-1 {
			return false
		}
	}
	return true
}

func (t *Table) Score() int {
	score := 0
	for _, count := range t.Piles {
		score += count
	}
	return score
}

// returns pile index if playable, -1 if not
func (t *Table) CardPlayableOnCustomPile(c Card, p []int) int {
	for index, count := range p {
		if t.Colors[index] == c.Color {
			if count+1 == c.Number {
				return index
			} else {
				return -1
			}
		}
	}
	return -1
}

func (t *Table) CardPlayableOnPile(c Card) int {
	return t.CardPlayableOnCustomPile(c, t.Piles)
}

func (t *Table) MaxCards() int {
	maxCards := 0
	for _, count := range numbers {
		maxCards += count * len(t.Colors)
	}
	if t.Mode == ModeHard || t.Mode == ModeRainbowLimited {
		maxCards -= 5
	}
	return maxCards
}

func PerfectScoreForMode(mode int) int {
	highScore := 30
	if mode == ModeNormal {
		highScore = 25
	}
	return highScore
}
