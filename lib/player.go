package lib

type Player struct {
	ID    string
	Cards []Card
}

func (p *Player) Initialize(maxCards int) {
	p.Cards = make([]Card, 0, maxCards)
}

func (p *Player) ReceiveHint(i int, infoType int) ([]int, string) {
	card, err := p.GetCard(i)
	if err != "" {
		return "Error retrieving card from player's hand: " + err
	}
	number := card.Number
	color := card.Color
	changedCards := []int;
	for index, _ := range p.Cards {
		if infoType == infoNumber && p.Cards[index].Number == number {
			p.Cards[index].KnownNumber = number
			changedCards = append(changedCards, p.Cards[index].Index)
		} else if infoType == infoColor && p.Cards[index].Color == color {
			p.Cards[index].KnownColor = color
			changedCards = append(changedCards, p.Cards[index].Index)
		}
	}
	return ""
}

func (p *Player) GetCard(i int) (Card, string) {
	if i >= len(p.Cards) {
		return Card{}, "Referenced a non-existent card in a player's hand."
	}
	return p.Cards[i], ""
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
