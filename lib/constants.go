package lib

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

const ModeNormal = 1
const ModeRainbow = 2
const ModeWildcard = 3
const ModeHard = 4

const ResultOther = 0
const ResultPlay = 1
const ResultBomb = 2

var normalColors = [...]string{"red", "green", "blue", "yellow", "white"}
var rainbowColors = [...]string{"red", "green", "blue", "yellow", "white", "rainbow"}
var numbers = [...]int{0, 3, 2, 2, 2, 1} // this represents the COUNTS of each number (0 added for simplicity)

var cardsInHand = [...]int{0, 0, 5, 5, 4, 4} // this represents the COUNTS for each # of players
const MaxPlayers = 5

const MaxConcurrentGames = 100
const MaxStoredGames = 1000
const DefaultClientDirectory = "/var/www/"
const DefaultPort = 8080
const DefaultCertificate = "server.crt"
const DefaultKey = "server.key"
const DefaultLogDirectory = "log/"

const DefaultMaxHints = 8
const DefaultStartingHints = 8
const DefaultStartingBombs = 3

const MaxPlayerNameLength = 10
const MaxGameNameLength = 20
