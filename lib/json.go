package lib

import (
	"encoding/json"
	"fmt"
	"os"
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
}

func EncodeGame(g Game) string {
	b, err := json.Marshal(g)
	if err != nil {
		fmt.Printf("Error encoding game to JSON string.")
		os.Exit(1)
	}

	return string(b)
}

func DecodeMove(s string) (Message, bool) {
	b := []byte(s)
	var m Message
	err := json.Unmarshal(b, &m)
	if err != nil {
		fmt.Printf("Error decoding move from JSON string.\nGot message: %s\nError:%s\n\n", err, s)
		return Message{}, false
	}

	return m, true
}
