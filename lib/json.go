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
