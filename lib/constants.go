package lib

const infoNumber = 1
const infoColor = 2

const MovePlay = 1
const MoveDiscard = 2
const MoveHint = 3

const StateNotStarted = 1
const StateStarted = 2
const StateBombedOut = 3
const StatePerfect = 4
const StateDeckEmpty = 5

var colors = [...]string{"red", "green", "blue", "yellow", "white"}
var numbers = [...]int{0, 3, 2, 2, 2, 1} // this represents the COUNTS of each number (0 added for simplicity)
const maxHints = 8
const startingHints = 8
const startingBombs = 3

var cardsInHand = [...]int{0, 0, 5, 5, 4, 4} // this represents the COUNTS for each # of players
var MaxPlayers = 5

const MaxConcurrentGames = 100
const DefaultClientDirectory = "/var/www/"
const DefaultPort = 8080
const DefaultCertificate = "server.crt"
const DefaultKey = "server.key"

const MaxPlayerNameLength = 10
const MaxGameNameLength = 20
