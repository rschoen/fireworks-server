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

func EncodeLogEntry(le LogEntry) (string, string) {
	b, err := json.Marshal(le)
	if err != nil {
		return "", "Error encoding log entry to JSON string: " + err.Error()
	}

	return string(b), ""
}

func DecodeLogEntry(s string) (LogEntry, string) {
	b := []byte(s)
	var le LogEntry
	err := json.Unmarshal(b, &le)
	if err != nil {
		return LogEntry{}, "Error decoding log entry from JSON string.\nDecoding string: " + s + "\nError: " + err.Error()
	}

	return le, ""
}

func EncodeStatsLog(l Logger) (string, string) {
	lm := LoggerMessage{Players: l.Players, Stats: l.Stats}
	b, err := json.Marshal(lm)
	if err != nil {
		return "", "Error encoding stats log to JSON string: " + err.Error()
	}

	return string(b), ""
}

func EncodeWholeStatsLog(l *Logger) (string, string) {
	b, err := json.Marshal(l)
	if err != nil {
		return "", "Error encoding stats log to JSON string: " + err.Error()
	}

	return string(b), ""
}

func DecodeWholeStatsLog(s string) (Logger, string) {
	b := []byte(s)
	var l Logger
	err := json.Unmarshal(b, &l)
	if err != nil {
		return Logger{}, "Error decoding stat logger from JSON string.\nDecoding string: " + s + "\nError: " + err.Error()
	}

	return l, ""
}


func DecodeTable(s string) (Table, string) {
	b := []byte(s)
	var table Table
	err := json.Unmarshal(b, &table)
	if err != nil {
		return Table{}, "Error decoding table from JSON string.\nDecoding string: " + s + "\nError: " + err.Error()
	}

	return table, ""
}

func EncodeTable(table Table) (string, string) {
	b, err := json.Marshal(table)
	if err != nil {
		return "", "Error encoding table to JSON string: " + err.Error()
	}

	return string(b), ""
}
