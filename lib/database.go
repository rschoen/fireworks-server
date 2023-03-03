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
								finished_games,
								turns,
								timed_turns,
								turn_time,
								game_time,
								plays,
								bombs,
								discards,
								hints,
								bombs_losses,
								turns_losses,
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
		var id, name, scoreString string
		var mode, players, finishedGames, turns, timedTurns, turnTime, gameTime, plays, bombs, discards, hints, bombsLosses, turnsLosses, noPlaysLosses int
		for rows.Next() {
			err = rows.Scan(&id, &name, &mode, &players, &finishedGames, &turns, &timedTurns, &turnTime, &gameTime, &plays, &bombs, &discards, &hints, &bombsLosses, &turnsLosses, &noPlaysLosses, &scoreString)
			if err != nil {
				log.Fatal(err)
			}

			if id != lastId {
				lastId = id
				l.Players = append(l.Players, PlayerLog{ID: id, Name: name, Stats: CreateEmptySlicedStatLog()})
			}
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].FinishedGames += int64(finishedGames)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Turns += int64(turns)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].TimedTurns += int64(timedTurns)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].TurnTime += int64(turnTime)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].GameTime += int64(gameTime)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Plays += int64(plays)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Bombs += int64(bombs)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Discards += int64(discards)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].Hints += int64(hints)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].BombsLosses += int64(bombsLosses)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].TurnsLosses += int64(turnsLosses)
			l.Players[len(l.Players)-1].Stats.ModesAndPlayers[mode][players].NoPlaysLosses += int64(noPlaysLosses)

			scoreList := scoreListFromString(scoreString)
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
								turn_time,
								game_time,
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
		var last_move_time, mode, players, turns, timedTurns, turnTime, gameTime, plays, bombs, discards, hints, state, score int
		for rows.Next() {
			err = rows.Scan(&id, &name, &last_move_time, &mode, &players, &turns, &timedTurns, &turnTime, &gameTime, &plays, &bombs, &discards, &hints, &state, &score)
			if err != nil {
				log.Fatal(err)
			}
			l.Stats.ModesAndPlayers[mode][players].Turns += int64(turns)
			l.Stats.ModesAndPlayers[mode][players].TimedTurns += int64(timedTurns)
			l.Stats.ModesAndPlayers[mode][players].TurnTime += int64(turnTime)
			l.Stats.ModesAndPlayers[mode][players].GameTime += int64(gameTime)
			l.Stats.ModesAndPlayers[mode][players].Plays += int64(plays)
			l.Stats.ModesAndPlayers[mode][players].Bombs += int64(bombs)
			l.Stats.ModesAndPlayers[mode][players].Discards += int64(discards)
			l.Stats.ModesAndPlayers[mode][players].Hints += int64(hints)

			if(state != StateNotStarted) {
				l.Stats.ModesAndPlayers[mode][players].FinishedGames += 1
			}
			if(state == StateNoPlays) {
				l.Stats.ModesAndPlayers[mode][players].NoPlaysLosses += 1
			}
			if(state == StateBombedOut) {
				l.Stats.ModesAndPlayers[mode][players].BombsLosses += 1
			}
			if(state == StateDeckEmpty) {
				l.Stats.ModesAndPlayers[mode][players].TurnsLosses += 1
			}


			//scoreList := scoreListFromString(scores)

			if(len(l.Stats.ModesAndPlayers[mode][players].Scores) == 0) {
				l.Stats.ModesAndPlayers[mode][players].Scores = make([]int,PerfectScoreForMode(mode)+1)
			}
			l.Stats.ModesAndPlayers[mode][players].Scores[score] += 1


			// TODO: modify game_players so that it holds the stats for what the player did in that game
			// TODO: then we need to go through and sum up what each of the players did in each of the {mode,game} combos
			// and add it into their player stats above
			// BUT as for right now, the stats is complete as emitted


		}
	}



	return l
}

func scoreListFromString(scoreString string) []int {
	scores := strings.Split(scoreString[1:len(scoreString)-1], " ")
	scoreList := make([]int, len(scores))
	var err error
	for i, score := range scores {
		if score != "" {
			scoreList[i], err = strconv.Atoi(score)
			if(err != nil) {
				log.Fatal(err)
			}
		}
	}
	return scoreList
}

/*func addScoreToScoreList(scoreList []int, score int) {
	for i := range(math.Max(len(s1,s2))) {
		s1[i] = s1[i] + s2[i]
	}
	return s1
}*/
