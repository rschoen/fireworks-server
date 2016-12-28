package lib

type Player struct {
	GoogleID string
	Name     string
	Cards    []Card
	LastMove string
}

func (p *Player) Initialize(maxCards int) {
	p.Cards = make([]Card, 0, maxCards)
}

func (p *Player) ReceiveHint(i int, infoType int, hintColor string, mode int) ([]int, string) {
	var changedCards []int
	card, err := p.GetCard(i)
	if err != "" {
		return changedCards, "Error retrieving card from player's hand: " + err
	}
	number := card.Number
	color := card.Color
	if color == "rainbow" {
		color = hintColor;
	}
	for index, _ := range p.Cards {
		if infoType == HintNumber && p.Cards[index].Number == number {
			p.Cards[index].KnownNumber = number
			changedCards = append(changedCards, p.Cards[index].ID)
		} else if infoType == HintColor && p.Cards[index].Color == color {
			p.Cards[index].KnownColor = color
			addCard = true
		} else if infoType == infoColor && (mode == ModeWildcard || mode == ModeHard) && p.Cards[index].Color == "rainbow" {
			if p.Cards[index].KnownColor == "" {
				p.Cards[index].KnownColor = color
			} else if p.Cards[index].KnownColor != color {
				p.Cards[index].KnownColor = "rainbow"
			}
			addCard = true
		}
		if addCard {
			changedCards = append(changedCards, p.Cards[index].ID)
		}
	}
	return changedCards, ""
}

func (p *Player) GetCard(i int) (Card, string) {
	if i >= len(p.Cards) {
		return Card{}, "Referenced a non-existent card in a player's hand."
	}
	return p.Cards[i], ""
}

func (p *Player) GetCardByID(id int) Card {
	for index, _ := range p.Cards {
		if p.Cards[index].ID == id {
			return p.Cards[index]
		}
	}
	return Card{}
}

func (p *Player) AddCard(c Card) string {
	if len(p.Cards) == cap(p.Cards) {
		return "Attempted to add a card beyond hand capacity."
	}
	p.Cards = append(p.Cards, c)

	return ""
}

func (p *Player) RemoveCard(i int) (Card, string) {
	if i >= len(p.Cards) {
		return Card{}, "Attempted to remove a non-existent card"
	}
	var removedCard = p.Cards[i]
	p.Cards = append(p.Cards[:i], p.Cards[i+1:]...)
	return removedCard, ""
}
