package lib

type Game struct {
	gameID		string
	maxNumber	int 
	maxHints	int
	startingHints	int
	currentHints	int
	startingBombs	int
	currentBombs	int
	players		[]Player	
	colors		[]string
	deck		[]Card
	table		[]int
	discard		[]Card
	currentPlayer	int
	startingTime	int
}

func (g *Game) Initialize() {
	// create hands
	// randomize currentPlayer
	// set current stuff to starting stuff
}

func (g *Game) ProcessMove(move string) bool {
	// parse JSON
	// make sure it's a valid move
	// update internal values
	// log move and current state - CreateState(0)
	// return whether it worked or not
	return true
}

func (g *Game) CreateState(p Player) string {
	// pull out information into JSON
	// make sure to obscure current player's info
	// return JSON
	return ""
}

func (g *Game) PlayCard(p Player, card int) {
	// move cards around
	// bomb maybe?!
}

func (g *Game) DiscardCard(p Player, card int) {
	// move cards around
	// get a hint back!
}

func (g *Game) GiveHint(p Player, card int, infoType int) {
	// reveal the info
	// take away a hint
}
