package lib

import (
	"os"
	"reflect"
	"strings"
	"time"
)

type Logger struct {
	Games []GameLog
	Players []PlayerLog
	Stats StatLog
	Directory string
}

type LogEntry struct {
	Game Game
	Move Message
}

type GameLog struct {
	ID	  string
	Name  string
	File  *os.File
	Stats StatLog
	LastMoveTime int64
}

type PlayerLog struct {
	ID    string
	Name  string
	Stats StatLog
}

type StatLog struct {
	Moves	int64
	Plays	int64
	Bombs	int64
	Discards int64
	Hints	int64
	NumberHints int64
	ColorHints int64
	BombsLosses int64
	TurnsLosses int64
	NoPlaysLosses int64
	TurnTime int64
	GameTime int64
	StartedGames int64
	FinishedGames int64
	Scores map[int]int
}

func (l *Logger) Initialize() ([]*Game, string) {
	l.Games = make([]GameLog, 0, MaxStoredGames)
	l.Players = make([]PlayerLog, 0, MaxStoredGames*MaxPlayers)

	err := os.Mkdir(l.Directory, os.ModeDir | os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return make([]*Game, 0, 0),"Error creating log directory: "+err.Error()
	}

	dir, openError := os.Open(l.Directory)
	if openError != nil {
		return make([]*Game, 0, 0),"Error opening log directory: "+openError.Error()
	}

	names, readError := dir.Readdirnames(0)
	if readError != nil {
		return make([]*Game, 0, 0),"Error opening log directory: "+readError.Error()
	}

	games := make([]*Game, 0, MaxConcurrentGames)
	for _, name := range names {
		if strings.Index(name, ".json") > -1 {
			json := "" // TODO read umtil newline

			le, decodeError := DecodeLogEntry(json)
			if decodeError != "" {
				return make([]*Game, 0, 0),"Error decoding log: "+readError.Error()
			}
			l.LogMove(le.Game, le.Move, false)

			if le.Game.State == StateNotStarted || le.Game.State == StateStarted {
				games = append(games, &le.Game)
			}
		}
	}

	return games, ""
}

func (l *Logger) LogMove(g Game, m Message, writeToFile bool) string {
	// find or create game log
	gl, err := l.GetOrCreateGameLog(g);
	if err != "" {
		return "Error retrieving game log: " + err;
	}

	// log move to game log
	if writeToFile {
		le := LogEntry{Game: g, Move: m}
		logError := gl.LogMove(le)
		if logError != "" {
			return "Error logging move: " + logError;
		}
	}

	// figure out all the stats
	now := time.Now().Unix()
	pl := l.GetOrCreatePlayerLog(m.Player, g)
	statList := [...]*StatLog{&l.Stats, &gl.Stats, &pl.Stats}
	allPlayersStatsList := l.GetAllPlayersStatList(g, gl)

	IncrementProperty("Moves", statList[:]...)
	IncreaseProperty("TurnTime", now-gl.LastMoveTime, statList[:]...)

	if m.MoveType == MovePlay && m.Result == ResultPlay {
		IncrementProperty("Plays", statList[:]...)
	} else if m.MoveType == MovePlay && m.Result == ResultBomb {
		IncrementProperty("Bombs", statList[:]...)
	} else if m.MoveType == MoveDiscard {
		IncrementProperty("Discards", statList[:]...)
	} else if m.MoveType == MoveHint {
		IncrementProperty("Hints", statList[:]...)
		if m.HintInfoType == HintNumber {
			IncrementProperty("NumberHints", statList[:]...)
		} else {
			IncrementProperty("ColorHints", statList[:]...)
		}
	}

	if gl.Stats.StartedGames == 0 {
		IncrementProperty("StartedGames", allPlayersStatsList[:]...)
	}

	if g.State == StateBombedOut {
		IncrementProperty("BombLosses", allPlayersStatsList[:]...)
	} else if g.State == StateDeckEmpty {
		IncrementProperty("TurnLosses", allPlayersStatsList[:]...)
	} else if g.State == StateNoPlays {
		IncrementProperty("NoPlaysLosses", allPlayersStatsList[:]...)
	}

	if g.State == StateBombedOut || g.State == StateDeckEmpty || g.State == StateNoPlays || g.State == StatePerfect {
		IncrementProperty("FinishedGames", allPlayersStatsList[:]...)
		IncreaseProperty("GameTime", now-g.StartTime, allPlayersStatsList[:]...)
		IncrementScore(g.Score(), allPlayersStatsList[:]...)

		gl.File.Close()
	}


	return ""
}

func (l *Logger) GetOrCreateGameLog(g Game) (*GameLog, string) {
	for index, _ := range l.Games {
		if l.Games[index].ID == g.ID {
			return &l.Games[index], ""
		}
	}

	gl := GameLog{ID: g.ID, Name: g.Name, Stats: StatLog{}}
	gl.LastMoveTime = g.StartTime
	file, err := os.OpenFile(l.Directory + g.ID + ".json", os.O_APPEND, os.ModeAppend)
	if err != nil {
		return new(GameLog), "Error opening log file: " + err.Error();
	}
	gl.File = file

	l.Games = append(l.Games, gl)
	return &gl, ""
}

func (l *Logger) GetOrCreatePlayerLog(p string, g Game) *PlayerLog {
	for index, _ := range l.Players {
		if l.Players[index].ID == p {
			return &l.Players[index]
		}
	}

	pl := PlayerLog{ID: p, Stats: StatLog{}}
	pl.Name = g.GetPlayerByGoogleID(p).Name
	l.Players = append(l.Players, pl)
	return &pl
}


func (l *Logger) GetAllPlayersStatList(g Game, gl *GameLog) []*StatLog {
	sl := make([]*StatLog, len(g.Players)+2, len(g.Players)+2)
	for _, player := range g.Players {
		pl := l.GetOrCreatePlayerLog(player.Name, g)
		sl = append(sl, &pl.Stats)
	}
	sl = append(sl, &l.Stats)
	sl = append(sl, &gl.Stats)
	return sl
}

func (l *Logger) CreateStatsLog() Logger {
	lCopy := Logger{}
	lCopy = *l

	// clear the logging directory
	lCopy.Directory = ""

	// clear game file handlers
	for index, _ := range lCopy.Games {
		lCopy.Games[index].File = nil
	}
	return lCopy
}



func (gl *GameLog) LogMove(le LogEntry) string {
	json, encodeError := EncodeLogEntry(le)
	if encodeError != "" {
		return "Error encoding log entry to JSON: " + encodeError;
	}
	_, err := gl.File.WriteString(json + "\n")
	if err != nil {
		return "Error writing log entry: " + err.Error();
	}
	return ""
}

func IncrementProperty(p string, stats ...*StatLog) {
	IncreaseProperty(p, 1, stats...)
}

func IncreaseProperty(p string, n int64, stats ...*StatLog) {
	for i := range stats {
		r := reflect.ValueOf(stats[i])
    	f := reflect.Indirect(r).FieldByName(p)
    	f.SetInt(f.Int() + n)
	}
}

func IncrementScore(n int, stats ...*StatLog) {
	for i := range stats {
		if stats[i].Scores == nil {
			stats[i].Scores = make(map[int]int)
		}
		if _, ok := stats[i].Scores[n]; ok {
    		stats[i].Scores[n]++;
		} else {
			stats[i].Scores[n] = 1;
		}
	}
}