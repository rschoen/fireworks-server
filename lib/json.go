package lib

import (
	"encoding/json"
)

type Message struct {
	Game          string
	Player        string
	CardIndex     int
	MoveType      int
	HintPlayer    string
	HintInfoType  int
	HintNumber    int
	HintColor     string
	Token         string
	PushToken     string
	Result        int
	GameMode      int
	Public        bool
	StartingHints int
	StartingBombs int
	MaxHints      int
	LastTurn      int
	UpdateTime    int64
	IgnoreTime    bool
	SighButton    bool
	Announcement  string
}

type MinimalGame struct {
	ID      string
	Name    string
	Players string
	Mode    int
}

type GamesList struct {
	OpenGames    []MinimalGame
	PlayersGames []MinimalGame
}

type StatsMessage struct {
	Players      map[string]PlayerStats
	LastMoveTime int64
	Stats        SlicedStatLog
}

type PlayerStats struct {
	ID    string
	Name  string
	Stats SlicedStatLog
}

type SlicedStatLog struct {
	ModesAndPlayers [][]StatLog
}

type StatLog struct {
	Turns         int64
	TimedTurns    int64
	Plays         int64
	Bombs         int64
	Discards      int64
	Hints         int64
	BombsLosses   int64
	TurnsLosses   int64
	NoPlaysLosses int64
	TurnTime      int64
	GameTime      int64
	StartedGames  int64
	FinishedGames int64
	Scores        []int
}

func CreateEmptySlicedStatLog() SlicedStatLog {
	ssl := SlicedStatLog{}

	//ssl.Modes = make([]StatLog, Modes+1, Modes+1)
	//ssl.NumPlayers = make([]StatLog, MaxPlayers+1, MaxPlayers+1)
	ssl.ModesAndPlayers = make([][]StatLog, Modes+1, Modes+1)

	for i := 0; i <= Modes; i++ {
		ssl.ModesAndPlayers[i] = make([]StatLog, MaxPlayers+1, MaxPlayers+1)
		for j := 0; j <= MaxPlayers; j++ {
			ssl.ModesAndPlayers[i][j].Scores = make([]int, MaxScoreAllModes+1, MaxScoreAllModes+1)
		}
	}

	return ssl
}

func EncodeList(gl GamesList) (string, string) {
	b, err := json.Marshal(gl)
	if err != nil {
		return "", "Error encoding list to JSON string: " + err.Error()
	}

	return string(b), ""
}

func EncodeGame(g Game) (string, string) {
	b, err := json.Marshal(g)
	if err != nil {
		return "", "Error encoding game to JSON string: " + err.Error()
	}

	return string(b), ""
}

func DecodeMove(s string) (Message, string) {
	b := []byte(s)
	var m Message
	err := json.Unmarshal(b, &m)
	if err != nil {
		return Message{}, "Error decoding move from JSON string.\nDecoding message: " + s + "\nError: " + err.Error()
	}

	return m, ""
}

func EncodeStatsMessage(sm StatsMessage) (string, string) {
	b, err := json.Marshal(sm)
	if err != nil {
		return "", "Error encoding stats log to JSON string: " + err.Error()
	}

	return string(b), ""
}

func DecodeTable(s string) (Table, string) {
	if s == "" {
		return Table{}, ""
	}
	b := []byte(s)
	var table Table
	err := json.Unmarshal(b, &table)
	if err != nil {
		return Table{}, "Error decoding table from JSON string.\nDecoding string: " + s + "\nError: " + err.Error()
	}

	return table, ""
}

func EncodeTable(table *Table) (string, string) {
	b, err := json.Marshal(table)
	if err != nil {
		return "", "Error encoding table to JSON string: " + err.Error()
	}

	return string(b), ""
}

func DecodePlayerHand(s string) ([]Card, string) {
	if s == "" {
		return make([]Card, 0), ""
	}
	b := []byte(s)
	var hand []Card
	err := json.Unmarshal(b, &hand)
	if err != nil {
		return make([]Card, 0), "Error decoding table from JSON string.\nDecoding string: " + s + "\nError: " + err.Error()
	}

	return hand, ""
}

func EncodePlayerHand(player Player) (string, string) {
	b, err := json.Marshal(player.Cards)
	if err != nil {
		return "", "Error encoding player hand to JSON string: " + err.Error()
	}

	return string(b), ""
}
