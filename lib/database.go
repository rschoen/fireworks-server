package lib

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"strconv"
	"strings"
)

type Database struct {
	dbRef          *sql.DB
	LastUpdateTime int64
}

func (db *Database) Connect(dbFile string) {

	dbRef, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	db.dbRef = dbRef
}

func (db *Database) CreateStatsMessage() StatsMessage {
	sm := StatsMessage{}
	sm.Players = make(map[string]PlayerStats)
	sm.Stats = CreateEmptySlicedStatLog()

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
				sm.Players[id] = PlayerStats{ID: id, Name: name, Stats: CreateEmptySlicedStatLog()}
			}
			sm.Players[id].Stats.ModesAndPlayers[mode][players].FinishedGames += int64(finishedGames)
			sm.Players[id].Stats.ModesAndPlayers[mode][players].Turns += int64(turns)
			sm.Players[id].Stats.ModesAndPlayers[mode][players].TimedTurns += int64(timedTurns)
			sm.Players[id].Stats.ModesAndPlayers[mode][players].TurnTime += int64(turnTime)
			sm.Players[id].Stats.ModesAndPlayers[mode][players].GameTime += int64(gameTime)
			sm.Players[id].Stats.ModesAndPlayers[mode][players].Plays += int64(plays)
			sm.Players[id].Stats.ModesAndPlayers[mode][players].Bombs += int64(bombs)
			sm.Players[id].Stats.ModesAndPlayers[mode][players].Discards += int64(discards)
			sm.Players[id].Stats.ModesAndPlayers[mode][players].Hints += int64(hints)
			sm.Players[id].Stats.ModesAndPlayers[mode][players].BombsLosses += int64(bombsLosses)
			sm.Players[id].Stats.ModesAndPlayers[mode][players].TurnsLosses += int64(turnsLosses)
			sm.Players[id].Stats.ModesAndPlayers[mode][players].NoPlaysLosses += int64(noPlaysLosses)

			scoreList := scoreListFromString(scoreString)
			sm.Players[id].Stats.ModesAndPlayers[mode][players].Scores = scoreList
		}
	}
	{
		rows, err := db.dbRef.Query(`
									SELECT player_id,
									players.name,
									turns,
									timed_turns,
									turn_time,
									game_time,
									plays,
									bombs,
									discards,
									hints,
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
			var turns, timedTurns, turnTime, gameTime, plays, bombs, discards, hints, score, mode, players, state int
			for rows.Next() {
				err = rows.Scan(&id, &name, &turns, &timedTurns, &turnTime, &gameTime, &plays, &bombs, &discards, &hints, &score, &mode, &players, &state)
				if err != nil {
					log.Fatal(err)
				}

				if !playersListContains(sm.Players, id) {
					sm.Players[id] = PlayerStats{ID: id, Name: name, Stats: CreateEmptySlicedStatLog()}
				}

				if state != StateNotStarted && state != StateStarted {
					sm.Players[id].Stats.ModesAndPlayers[mode][players].FinishedGames += 1
				}
				sm.Players[id].Stats.ModesAndPlayers[mode][players].Turns += int64(turns)
				sm.Players[id].Stats.ModesAndPlayers[mode][players].TimedTurns += int64(timedTurns)
				sm.Players[id].Stats.ModesAndPlayers[mode][players].TurnTime += int64(turnTime)
				sm.Players[id].Stats.ModesAndPlayers[mode][players].GameTime += int64(gameTime)
				sm.Players[id].Stats.ModesAndPlayers[mode][players].Plays += int64(plays)
				sm.Players[id].Stats.ModesAndPlayers[mode][players].Bombs += int64(bombs)
				sm.Players[id].Stats.ModesAndPlayers[mode][players].Discards += int64(discards)
				sm.Players[id].Stats.ModesAndPlayers[mode][players].Hints += int64(hints)

				if state == StateBombedOut {
					sm.Players[id].Stats.ModesAndPlayers[mode][players].BombsLosses += 1
				} else if state == StateDeckEmpty {
					sm.Players[id].Stats.ModesAndPlayers[mode][players].TurnsLosses += 1
				} else if state == StateNoPlays {
					sm.Players[id].Stats.ModesAndPlayers[mode][players].NoPlaysLosses += 1
				}
				sm.Players[id].Stats.ModesAndPlayers[mode][players].Scores[score] += 1
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
		var last_move_time, mode, players, turns, timedTurns, turnTime, gameTime, plays, bombs, discards, hints, state, score int
		for rows.Next() {
			err = rows.Scan(&id, &name, &last_move_time, &mode, &players, &turns, &timedTurns, &turnTime, &gameTime, &plays, &bombs, &discards, &hints, &state, &score)
			if err != nil {
				log.Fatal(err)
			}
			sm.Stats.ModesAndPlayers[mode][players].Turns += int64(turns)
			sm.Stats.ModesAndPlayers[mode][players].TimedTurns += int64(timedTurns)
			sm.Stats.ModesAndPlayers[mode][players].TurnTime += int64(turnTime)
			sm.Stats.ModesAndPlayers[mode][players].GameTime += int64(gameTime)
			sm.Stats.ModesAndPlayers[mode][players].Plays += int64(plays)
			sm.Stats.ModesAndPlayers[mode][players].Bombs += int64(bombs)
			sm.Stats.ModesAndPlayers[mode][players].Discards += int64(discards)
			sm.Stats.ModesAndPlayers[mode][players].Hints += int64(hints)

			if state != StateNotStarted {
				sm.Stats.ModesAndPlayers[mode][players].FinishedGames += 1
			}
			if state == StateNoPlays {
				sm.Stats.ModesAndPlayers[mode][players].NoPlaysLosses += 1
			}
			if state == StateBombedOut {
				sm.Stats.ModesAndPlayers[mode][players].BombsLosses += 1
			}
			if state == StateDeckEmpty {
				sm.Stats.ModesAndPlayers[mode][players].TurnsLosses += 1
			}

			//scoreList := scoreListFromString(scores)

			if len(sm.Stats.ModesAndPlayers[mode][players].Scores) == 0 {
				sm.Stats.ModesAndPlayers[mode][players].Scores = make([]int, PerfectScoreForMode(mode)+1)
			}
			sm.Stats.ModesAndPlayers[mode][players].Scores[score] += 1

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

/*func addScoreToScoreList(scoreList []int, score int) {
	for i := range(math.Max(len(s1,s2))) {
		s1[i] = s1[i] + s2[i]
	}
	return s1
}*/

func (db *Database) GetGameListForPlayer(player string) []string {
	rows, err := db.dbRef.Query(`select game_id from player_games left join on game.id=game_id where (player_id=$1 and state=$2) OR state=$3)`, player, StateStarted, StateNotStarted)

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	// TODO: fix this
	var gameIds = make([]string, 0, MaxConcurrentGames)
	for rows.Next() {
		var gameId string
		err = rows.Scan(&gameId)
		if err != nil {
			log.Fatal(err)
		}
		gameIds = append(gameIds, gameId)
	}
	return gameIds
}

func (db *Database) GetActiveGames() map[string]Game {
	rows, err := db.dbRef.Query(`select id from games where state == $1 or state == $2`, StateNotStarted, StateStarted)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	// TODO: fix this
	var games = make(map[string]Game)
	var id string
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			log.Fatal(err)
		}
		games[id] = db.LookupGameById(id)
	}
	return games
}

func (db *Database) LookupGameById(id string) Game {
	row := db.dbRef.QueryRow(`select name,
		current_player_index, state, time_started, last_move_time, turns, timed_turns,
		turn_time, game_time, plays, bombs, discards, hints,
		score, mode, players, initialized, public, ignore_time, sigh_button, table_state
		 												from games where id=$1`, id)
	var name, tableState string
	var initialized, public, ignoreTime, sighButton bool
	var currentPlayerIndex, state, timeStarted, lastMoveTime, turns, timedTurns,
		turnTime, gameTime, plays, bombs, discards, hints, score, mode, players int

	switch err := row.Scan(&name,
		&currentPlayerIndex, &state, &timeStarted, &lastMoveTime, turns, &timedTurns,
		&turnTime, &gameTime, &plays, &bombs, &discards, &hints,
		&score, &mode, &players, &initialized, &public, &ignoreTime, &sighButton, &tableState); err {
	case sql.ErrNoRows:
		fmt.Println("Game not found: " + id)
	case nil:
		game := Game{}
		game.ID = id
		game.Name = name
		game.State = state
		game.StartTime = int64(timeStarted)
		game.LastUpdateTime = int64(lastMoveTime)
		game.Mode = mode
		game.Initialized = initialized
		game.Public = public
		game.IgnoreTime = ignoreTime
		game.SighButton = sighButton
		game.Score = score

		game.Stats = StatLog{}
		game.Stats.Turns = int64(turns)
		game.Stats.TimedTurns = int64(timedTurns)
		game.Stats.TurnTime = int64(turnTime)
		game.Stats.GameTime = int64(gameTime)
		game.Stats.Plays = int64(plays)
		game.Stats.Bombs = int64(bombs)
		game.Stats.Discards = int64(discards)
		game.Stats.Hints = int64(hints)

		json, err := DecodeTable(tableState)
		if err != "" {
			log.Fatal(err)
		}
		game.Table = json
		game.Players = db.GetGamePlayers(id)

		return game

	default:
		panic(err)
	}
	return Game{}
}

func (db *Database) SaveGameToDatabase(game Game) {
	json, error := EncodeTable(game.Table)
	if error != "" {
		log.Fatal(error)
	}

	_, err := db.dbRef.Exec(`update games set current_player_index=$1, state=$2,
		time_started=$3, last_move_time=$4, turns=$5, timed_turns=$6, turn_time=$7,
		game_time=$8, plays=$9, bombs=$10, discards=$11, hints=$12, score=$13,
		players=$14, initialized=$15, table_state=$16, players=$17 where id=$18`,
		game.CurrentPlayerIndex, game.State, game.StartTime, game.LastUpdateTime,
		game.Stats.Turns, game.Stats.TimedTurns, game.Stats.TurnTime,
		game.Stats.GameTime, game.Stats.Plays, game.Stats.Bombs,
		game.Stats.Discards, game.Stats.Hints, game.Score,
		game.Players, game.Initialized, json, len(game.Players), game.ID)

	if err != nil {
		panic(err)
	}
}
func (db *Database) CreateGame(game Game) {
	_, err := db.dbRef.Exec(`insert into games (id, name, time_started,
		last_move_time, mode, players) values ($1, $2, $3, $4, $5, $6)`,
		game.ID, game.Name, game.StartTime, game.LastUpdateTime, game.Mode, game.Players)

	if err != nil {
		panic(err)
	}
}
func (db *Database) AddPlayer(playerId string, gameId string) {
	var nextIndex = db.GetNumPlayersInGame(gameId)
	_, err := db.dbRef.Exec(`insert into games (game_id, player_id, index)
			values ($1, $2, $3)`, gameId, playerId, nextIndex)
	if err != nil {
		panic(err)
	}

}

func (db *Database) GetNumPlayersInGame(gameId string) int {
	row := db.dbRef.QueryRow(`select count(index) as players from player_games where game_id=`, gameId)

	var players int
	switch err := row.Scan(&players); err {
	case sql.ErrNoRows:
		return 0
	case nil:
		return players
	default:
		panic(err)
		return -1
	}
}

func (db *Database) GetGamePlayers(id string) []Player {
	rows, err := db.dbRef.Query(`select player_id,name from player_games left join players on players.id=player_id where game_id=$1 order by index`, id)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	// TODO: fix this
	var players = make([]Player, MaxPlayers)
	var i = 0
	for rows.Next() {
		var playerId, name string
		err = rows.Scan(&playerId, &name)
		if err != nil {
			log.Fatal(err)
		}
		players[i] = Player{GoogleID: id, Name: name}
		i++
	}
	return players
}

func (db *Database) LogMove(g Game, m Message, t int64) string {

	var mainPlayerSql = "turns=turns+1, "
	var gameSql = "turns=turns+1, "
	var allPlayersSql = ""

	if !g.IgnoreTime {
		mainPlayerSql += "timed_turns=timed_turns+1, turn_time=turn_time+" + string(t-g.LastUpdateTime) + ", "
		gameSql += "timed_turns=timed_turns+1, turn_time=turn_time+" + string(t-g.LastUpdateTime) + ", "
	}

	if m.MoveType == MovePlay && m.Result == ResultPlay {
		mainPlayerSql += "plays=plays+1, "
		gameSql += "plays=plays+1, "
	} else if m.MoveType == MovePlay && m.Result == ResultBomb {
		mainPlayerSql += "bombs=bombs+1, "
		gameSql += "bombs=bombs+1, "
	} else if m.MoveType == MoveDiscard {
		mainPlayerSql += "discards=dicsards+1, "
		gameSql += "discards=dicsards+1, "
	} else if m.MoveType == MoveHint {
		mainPlayerSql += "hints=hints+1, "
		gameSql += "hints=hints+1, "
	}

	if g.State == StateBombedOut {
		allPlayersSql += "bomb_losses=bomb_losses+1, "
	} else if g.State == StateDeckEmpty {
		mainPlayerSql += "turns_losses=turns_losses+1, "
	} else if g.State == StateNoPlays {
		allPlayersSql += "no_plays_losses=no_plays_losses+1, "
	}

	if g.State == StateBombedOut || g.State == StateDeckEmpty || g.State == StateNoPlays || g.State == StatePerfect {
		allPlayersSql += "finished_games=finished_games+1, "
		if !g.IgnoreTime {
			allPlayersSql += "game_time=game_time+" + string(t-g.StartTime) + ", "
		}
	}

	db.execQuery("update players set " + allPlayersSql[:len(allPlayersSql)-2] + " where id in (" + g.GetPlayerListAsString() + ")")
	db.execQuery("update players set "+mainPlayerSql[:len(mainPlayerSql)-2]+" where id=$1", m.Player)
	db.execQuery("update game set "+gameSql[:len(gameSql)-2]+" where id=$1", g.ID)

	g.LastUpdateTime = t
	if t > db.LastUpdateTime {
		db.LastUpdateTime = t
	}

	return ""
}

func (db *Database) execQuery(query string, args ...string) {
	_, err := db.dbRef.Exec(query, args)
	if err != nil {
		log.Fatal(err)
	}
}

func playersListContains(list map[string]PlayerStats, id string) bool {
	for key, _ := range list {
		if key == id {
			return true
		}
	}
	return false
}
