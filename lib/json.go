package lib

import (
	"encoding/json"
)

type Message struct {
	Game         string
	Player       string
	CardIndex    int
	MoveType     int
	HintPlayer   string
	HintInfoType int
	HintNumber   int
	HintColor    string
	Token        string
	GameMode	 int
	Public		 bool
	StartingHints int
	StartingBombs int
	MaxHints	int
}

type MinimalGame struct {
	ID      string
	Name    string
	Players string
}

type GamesList struct {
	OpenGames    []MinimalGame
	PlayersGames []MinimalGame
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
