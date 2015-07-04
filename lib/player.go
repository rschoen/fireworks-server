package lib

type Player struct {
	playerID	string
	cards	[]Card
}

func (h *Player) ReceiveHint(c *Card, infoType int) {
	// find the card
	// depending on infoType, find all cards of that number/color
	// reveal that card's number/color
}

func (h *Player) GetCard(i int) *Card {
	// return *Card i
	return &Card{}
}

func (h *Player) AddCard(c *Card) {
	// add to the list
}

func (h *Player) RemoveCard(c *Card) {
	// remove card from list
}
