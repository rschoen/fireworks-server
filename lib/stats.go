package lib

import (
	"log"
	"strconv"
	"strings"
)

func (db *Database) CreateStatsMessage() StatsMessage {
	sm := StatsMessage{}
	sm.Players = make(map[string]PlayerStats)
	sm.Stats = CreateEmptyStatsArray()

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
		var turnTime, gameTime int64
		var mode, players, finishedGames, turns, timedTurns, plays, bombs, discards, hints, bombsLosses, turnsLosses, noPlaysLosses int
		for rows.Next() {
			err = rows.Scan(&id, &name, &mode, &players, &finishedGames, &turns, &timedTurns, &turnTime, &gameTime, &plays, &bombs, &discards, &hints, &bombsLosses, &turnsLosses, &noPlaysLosses, &scoreString)
			if err != nil {
				log.Fatal(err)
			}

			if id != lastId {
				lastId = id
				sm.Players[id] = PlayerStats{ID: id, Name: name, Stats: CreateEmptyStatsArray()}
			}
			sm.Players[id].Stats[mode][players].FinishedGames += int64(finishedGames)
			sm.Players[id].Stats[mode][players].Turns += int64(turns)
			sm.Players[id].Stats[mode][players].TimedTurns += int64(timedTurns)
			sm.Players[id].Stats[mode][players].TurnTime += int64(turnTime)
			sm.Players[id].Stats[mode][players].GameTime += int64(gameTime)
			sm.Players[id].Stats[mode][players].Plays += int64(plays)
			sm.Players[id].Stats[mode][players].Bombs += int64(bombs)
			sm.Players[id].Stats[mode][players].Discards += int64(discards)
			sm.Players[id].Stats[mode][players].Hints += int64(hints)
			sm.Players[id].Stats[mode][players].BombsLosses += int64(bombsLosses)
			sm.Players[id].Stats[mode][players].TurnsLosses += int64(turnsLosses)
			sm.Players[id].Stats[mode][players].NoPlaysLosses += int64(noPlaysLosses)

			scoreList := scoreListFromString(scoreString)
			sm.Players[id].Stats[mode][players].Scores = scoreList
		}
	}
	{
		rows, err := db.dbRef.Query(`
									SELECT player_id,
									players.name,
									game_players.turns,
									game_players.timed_turns,
									game_players.turn_time,
									game_players.plays,
									game_players.bombs,
									game_players.discards,
									game_players.hints,
									score,
									mode,
									players,
									state
									FROM game_players
									INNER JOIN games on game_id=games.id
									INNER JOIN players on player_id=players.id
									order by player_id `)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		{
			var id, name string
			var turnTime int64
			var turns, timedTurns, plays, bombs, discards, hints, score, mode, players, state int
			for rows.Next() {
				err = rows.Scan(&id, &name, &turns, &timedTurns, &turnTime, &plays, &bombs, &discards, &hints, &score, &mode, &players, &state)
				if err != nil {
					log.Fatal(err)
				}

				if _, ok := sm.Players[id]; !ok {
					sm.Players[id] = PlayerStats{ID: id, Name: name, Stats: CreateEmptyStatsArray()}
				}

				if state != StateNotStarted && state != StateStarted {
					sm.Players[id].Stats[mode][players].FinishedGames += 1
				}
				sm.Players[id].Stats[mode][players].Turns += int64(turns)
				sm.Players[id].Stats[mode][players].TimedTurns += int64(timedTurns)
				sm.Players[id].Stats[mode][players].TurnTime += int64(turnTime)
				sm.Players[id].Stats[mode][players].Plays += int64(plays)
				sm.Players[id].Stats[mode][players].Bombs += int64(bombs)
				sm.Players[id].Stats[mode][players].Discards += int64(discards)
				sm.Players[id].Stats[mode][players].Hints += int64(hints)

				if state == StateBombedOut {
					sm.Players[id].Stats[mode][players].BombsLosses += 1
				} else if state == StateDeckEmpty {
					sm.Players[id].Stats[mode][players].TurnsLosses += 1
				} else if state == StateNoPlays {
					sm.Players[id].Stats[mode][players].NoPlaysLosses += 1
				}
				sm.Players[id].Stats[mode][players].Scores[score] += 1
			}
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
		var turnTime, gameTime int64
		var last_move_time, mode, players, turns, timedTurns, plays, bombs, discards, hints, state, score int
		for rows.Next() {
			err = rows.Scan(&id, &name, &last_move_time, &mode, &players, &turns, &timedTurns, &turnTime, &gameTime, &plays, &bombs, &discards, &hints, &state, &score)
			if err != nil {
				log.Fatal(err)
			}
			sm.Stats[mode][players].Turns += int64(turns)
			sm.Stats[mode][players].TimedTurns += int64(timedTurns)
			sm.Stats[mode][players].TurnTime += int64(turnTime)
			sm.Stats[mode][players].GameTime += int64(gameTime)
			sm.Stats[mode][players].Plays += int64(plays)
			sm.Stats[mode][players].Bombs += int64(bombs)
			sm.Stats[mode][players].Discards += int64(discards)
			sm.Stats[mode][players].Hints += int64(hints)

			if state != StateNotStarted {
				sm.Stats[mode][players].FinishedGames += 1
			}
			if state == StateNoPlays {
				sm.Stats[mode][players].NoPlaysLosses += 1
			}
			if state == StateBombedOut {
				sm.Stats[mode][players].BombsLosses += 1
			}
			if state == StateDeckEmpty {
				sm.Stats[mode][players].TurnsLosses += 1
			}

			//scoreList := scoreListFromString(scores)

			if len(sm.Stats[mode][players].Scores) == 0 {
				sm.Stats[mode][players].Scores = make([]int, PerfectScoreForMode(mode)+1)
			}
			sm.Stats[mode][players].Scores[score] += 1

			// TODO: modify game_players so that it holds the stats for what the player did in that game
			// TODO: then we need to go through and sum up what each of the players did in each of the {mode,game} combos
			// and add it into their player stats above
			// BUT as for right now, the stats is complete as emitted

		}
	}

	return sm
}

func scoreListFromString(scoreString string) []int {
	scores := strings.Split(scoreString[1:len(scoreString)-1], " ")
	scoreList := make([]int, len(scores))
	var err error
	for i, score := range scores {
		if score != "" {
			scoreList[i], err = strconv.Atoi(score)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	return scoreList
}
