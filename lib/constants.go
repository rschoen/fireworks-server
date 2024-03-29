package lib

const VERSION = "2.0.2"

const HintNumber = 1
const HintColor = 2

const MovePlay = 1
const MoveDiscard = 2
const MoveHint = 3

const StateNotStarted = 1
const StateStarted = 2
const StateBombedOut = 3
const StatePerfect = 4
const StateDeckEmpty = 5
const StateNoPlays = 6

func GameStateIsFinished(state int) bool {
	if state != StateNotStarted && state != StateStarted {
		return true
	} else {
		return false
	}
}

const ModeNormal = 1
const ModeRainbow = 2
const ModeWildcard = 3
const ModeHard = 4
const ModeRainbowLimited = 5
const Modes = 5

const ColorRainbow = "rainbow"

const ResultOther = 0
const ResultPlay = 1
const ResultBomb = 2

var normalColors = [...]string{"red", "green", "blue", "yellow", "white"}
var rainbowColors = [...]string{"red", "green", "blue", "yellow", "white", ColorRainbow}
var numbers = [...]int{0, 3, 2, 2, 2, 1} // this represents the COUNTS of each number (0 added for simplicity)

var cardsInHand = [...]int{0, 0, 5, 5, 4, 4} // this represents the COUNTS for each # of players
const MaxPlayers = 5
const MaxScoreAllModes = 30

const MaxConcurrentGames = 100
const MaxStoredGames = 1000
const DefaultClientDirectory = "/var/www/html/"
const DefaultPort = 8080
const DefaultCertificate = "server.crt"
const DefaultKey = "server.key"
const DefaultDatabaseFile = "database.db"
const AuthExpirationSeconds = 7 * 24 * 60 * 60

const MaxHints = 8
const StartingHints = 8
const StartingBombs = 3

const MaxPlayerNameLength = 10
const MaxGameNameLength = 20
