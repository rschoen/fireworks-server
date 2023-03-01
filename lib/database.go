package lib

import (
	"database/sql"
	"log"

	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	dbRef *sql.DB
}

func (db *Database) Connect() {

	dbRef, err := sql.Open("sqlite3", DatabaseFile)
	if err != nil {
		log.Fatal(err)
	}
	db.dbRef = dbRef
}

func (db *Database) CreateStatsLog() Logger {
	l := Logger{}
	l.Games = make([]GameLog, 0, MaxStoredGames)
	l.Players = make([]PlayerLog, 0, MaxStoredGames*MaxPlayers)
	l.Stats = CreateEmptySlicedStatLog()

	rows, err := db.dbRef.Query(`
								SELECT players.id as id,
								name,
								mode,
								players,
								turns,
								timed_turns,
								plays,
								bombs,
								discards,
								hints,
								bombs_losses,
								turn_losses,
								no_plays_losses,
								score_list
								FROM players
								INNER JOIN legacy_player_stats ON players.id=legacy_player_stats.id
								ORDER BY players.id`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	{
		lastId := ""
		var id, name, scoreList string
		var mode, players, turns, timedTurns, plays, bombs, discards, hints, bombsLosses, turnLosses, noPlaysLosses int
		for rows.Next() {
			err = rows.Scan(&id, &name, &mode, &players, &turns, &timedTurns, &plays, &bombs, &discards, &hints, &bombsLosses, &turnLosses, &noPlaysLosses, &scoreList)
			if err != nil {
				log.Fatal(err)
			}

			if id != lastId {
				lastId = id
				l.Players = append(l.Players, PlayerLog{ID: id, Name: name, Stats: CreateEmptySlicedStatLog()})
			}
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Turns += int64(turns)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].TimedTurns += int64(timedTurns)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Plays += int64(plays)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Bombs += int64(bombs)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Discards += int64(discards)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Hints += int64(hints)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].BombsLosses += int64(hints)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Hints += int64(hints)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Hints += int64(hints)

			scores := strings.Split(" ", scoreList[1:len(scoreList)-1])
			scoreList := make([]int, len(scores))
			for i, _ := range scores {
				scoreList[i], err = strconv.Atoi(scores[i])
			}
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Scores = scoreList
		}
	}
	{
		rows, err := db.dbRef.Query(`
								SELECT 
								id,
								name,
								last_move_time,
								mode,
								players,
								turns,
								timed_turns,
								plays,
								bombs,
								discards,
								hints,
								state,
								score
								FROM games`)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		var id, name string
		var last_move_time, mode, players, turns, timedTurns, plays, bombs, discards, hints, state, score int
		for rows.Next() {
			err = rows.Scan(&id, &name, &last_move_time, &mode, &players, &turns, &timedTurns, &plays, &bombs, &discards, &hints, &state, &score)
			if err != nil {
				log.Fatal(err)
			}

			if id != lastId {
				lastId = id
				l.Players = append(l.Players, PlayerLog{ID: id, Name: name, Stats: CreateEmptySlicedStatLog()})
			}
			g := Game{}
			// TODO: finish out making it so that this only needs to update a single thing
			g.Stats.Overall
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Turns += int64(turns)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].TimedTurns += int64(timedTurns)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Plays += int64(plays)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Bombs += int64(bombs)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Discards += int64(discards)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Hints += int64(hints)

			scores := strings.Split(" ", scoreList[1:len(scoreList)-1])
			scoreList := make([]int, len(scores))
			for i, _ := range scores {
				scoreList[i], err = strconv.Atoi(scores[i])
			}
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Scores = scoreList
		}
	}

	return l
}
