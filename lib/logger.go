package lib

type Logger struct {
	Players      map[string]PlayerLog
	LastMoveTime int64
	Initialized  bool
	Stats        SlicedStatLog
}

type LoggerMessage struct {
	Players map[string]PlayerLog
	Stats   SlicedStatLog
}

type LogEntry struct {
	Timestamp int64
	Game      Game
	Move      Message
}

type PlayerLog struct {
	ID    string
	Name  string
	Stats SlicedStatLog
}

type SlicedStatLog struct {
	Overall         StatLog
	Modes           []StatLog
	NumPlayers      []StatLog
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

func (l *Logger) CreateStatsLog() Logger {
	lCopy := Logger{}
	lCopy = *l
	return lCopy
}

func CreateEmptySlicedStatLog() SlicedStatLog {
	ssl := SlicedStatLog{}

	//ssl.Modes = make([]StatLog, Modes+1, Modes+1)
	//ssl.NumPlayers = make([]StatLog, MaxPlayers+1, MaxPlayers+1)
	ssl.ModesAndPlayers = make([][]StatLog, Modes+1, Modes+1)

	for i := 0; i <= Modes; i++ {
		ssl.ModesAndPlayers[i] = make([]StatLog, MaxPlayers+1, MaxPlayers+1)
	}

	return ssl
}
