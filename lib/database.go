package lib

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	dbRef          *sql.DB
	LastUpdateTime int64
	tx             *sql.Tx
}

func (db *Database) Connect(dbFile string) {

	dbRef, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	db.dbRef = dbRef
	db.tx = nil
}

func (db *Database) openTransaction() {
	if db.tx != nil {
		log.Fatal("Attempting to open a transaction when one is already open. Quitting.")
	}
	var err error
	db.tx, err = db.dbRef.BeginTx(context.Background(), nil)
	if err != nil {
		log.Print("Error opening transaction:")
		log.Fatal(err)
	}
	log.Print("OPENED TRANSACTION")
}

func (db *Database) execWithinTransaction(query string, args ...interface{}) {
	if db.tx == nil {
		log.Fatal("Attempting to execute a query within a transaction without an open transaction. Quitting")
	}
	res, err := db.tx.Exec(query, args...)
	if err != nil {
		db.tx.Rollback()
		log.Printf("Error executing query: %s", query)
		log.Fatal(err)
	} else {
		rows, rowsErr := res.RowsAffected()
		if rowsErr != nil {
			db.tx.Rollback()
			log.Printf("Error calculating rows affected by query: %s", query)
			log.Fatal(rowsErr)
		} else {
			log.Printf("Ran transaction query: %s", query)
			log.Printf("AFFECTED %d ROWS", rows)
		}
	}
}

func (db *Database) closeTransaction() {
	if db.tx == nil {
		log.Fatal("Attempting to close transaction without an open transaction. Quitting")
	}
	err := db.tx.Commit()
	if err != nil {
		log.Print("Error closing transaction:")
		log.Fatal(err)
	}
	db.tx = nil
	log.Print("TRANSACTION CLOSED")
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
		var turnTime, gameTime int64
		var mode, players, finishedGames, turns, timedTurns, plays, bombs, discards, hints, bombsLosses, turnsLosses, noPlaysLosses int
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
					sm.Players[id] = PlayerStats{ID: id, Name: name, Stats: CreateEmptySlicedStatLog()}
				}

				if state != StateNotStarted && state != StateStarted {
					sm.Players[id].Stats.ModesAndPlayers[mode][players].FinishedGames += 1
				}
				sm.Players[id].Stats.ModesAndPlayers[mode][players].Turns += int64(turns)
				sm.Players[id].Stats.ModesAndPlayers[mode][players].TimedTurns += int64(timedTurns)
				sm.Players[id].Stats.ModesAndPlayers[mode][players].TurnTime += int64(turnTime)
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
		var turnTime, gameTime int64
		var last_move_time, mode, players, turns, timedTurns, plays, bombs, discards, hints, state, score int
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

func (db *Database) GetGamesPlayerIsIn(player string) []string {
	rows, err := db.dbRef.Query(`select game_id from game_players left join games on games.id=game_id where player_id=? and (state=? or state=?) order by time_started desc`, player, StateStarted, StateNotStarted)

	if err != nil {
		log.Println("Error fetching list of games for player.")
		log.Fatal(err)
	}
	defer rows.Close()
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
func (db *Database) GetJoinableGames() []string {
	rows, err := db.dbRef.Query(`select id from games where state=? AND public=1 AND players<? order by time_started desc`, StateNotStarted, MaxPlayers)

	if err != nil {
		log.Println("Error fetching list of joinable games.")
		log.Fatal(err)
	}
	defer rows.Close()
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

func (db *Database) GetActiveGames() map[string]*Game {
	rows, err := db.dbRef.Query(`select id from games where state == ? or state == ?`, StateNotStarted, StateStarted)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var games = make(map[string]*Game)
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

func (db *Database) LookupGameById(id string) *Game {
	row := db.dbRef.QueryRow(`select name,
		state, time_started, last_move_time, turns, timed_turns,
		turn_time, game_time, plays, bombs, discards, hints,
		score, mode, players, public, ignore_time, sigh_button, table_state
		 												from games where id=?`, id)
	var name, tableState string
	var public, ignoreTime, sighButton bool
	var state, lastMoveTime, turns, timedTurns,
		plays, bombs, discards, hints, score, mode, players int
	var timeStarted, turnTime, gameTime int64

	switch err := row.Scan(&name,
		&state, &timeStarted, &lastMoveTime, &turns, &timedTurns,
		&turnTime, &gameTime, &plays, &bombs, &discards, &hints,
		&score, &mode, &players, &public, &ignoreTime, &sighButton, &tableState); err {
	case sql.ErrNoRows:
		fmt.Println("Game not found: " + id)
	case nil:
		game := new(Game)
		game.ID = id
		game.Name = name
		game.State = state
		game.StartTime = int64(timeStarted)
		game.LastUpdateTime = int64(lastMoveTime)
		game.Mode = mode
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

		table, err := DecodeTable(tableState)
		if err != "" {
			log.Fatal(err)
		}
		game.Table = &table
		game.Players = db.GetGamePlayers(id)

		return game

	default:
		panic(err)
	}
	return new(Game)
}

func (db *Database) SaveGameToDatabase(game *Game) {
	json, error := EncodeTable(game.Table)
	if error != "" {
		log.Fatal(error)
	}

	db.openTransaction()

	db.execWithinTransaction(`update games set state=?, last_move_time=?,
		score=?, players=?, table_state=?, time_started=? where id=?`,
		game.State, game.LastUpdateTime, game.Score, len(game.Players), json, game.StartTime, game.ID)

	for _, player := range game.Players {
		cardJson, cardError := EncodePlayerHand(player)
		if cardError != "" {
			log.Fatal(error)
		}

		db.execWithinTransaction(`update game_players set last_move=?, hand_state=? where game_id=? AND player_id=?`, player.LastMove, cardJson, game.ID, player.GoogleID)
	}

	db.closeTransaction()
}
func (db *Database) CreateGame(game Game) {
	json, error := EncodeTable(game.Table)
	if error != "" {
		log.Fatal(error)
	}

	db.execQuery(`insert into games (id, name, time_started,
		last_move_time, mode, players, state, table_state, public, ignore_time, sigh_button) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		game.ID, game.Name, game.StartTime, game.LastUpdateTime, game.Mode,
		len(game.Players), game.State, json, game.Public, game.IgnoreTime, game.SighButton)

}
func (db *Database) AddPlayer(playerId string, gameId string) {
	var nextIndex = db.GetNumPlayersInGame(gameId)

	db.openTransaction()
	db.execWithinTransaction(`insert into game_players (game_id, player_id, player_index)
			values (?, ?, ?)`, gameId, playerId, nextIndex)
	db.execWithinTransaction(`update games set players=players+1 where id=?`, gameId)
	db.closeTransaction()
}

func (db *Database) CreatePlayerIfNotExists(id string, name string) {
	row := db.dbRef.QueryRow(`select name from players where id=?`, id)

	var foundName string
	switch err := row.Scan(&foundName); err {
	case sql.ErrNoRows:
		db.execQuery(`insert into players (id,name) values (?,?)`, id, name)
		return
	case nil:
		if name != foundName {
			db.execQuery(`update players set name=? where id=?`, name, id)
		}
		return
	default:
		log.Println("Error checking if player exists")
		panic(err)
	}
}

func (db *Database) GetNumPlayersInGame(gameId string) int {
	row := db.dbRef.QueryRow(`select count(player_index) as players from game_players where game_id=?`, gameId)

	var players int
	switch err := row.Scan(&players); err {
	case sql.ErrNoRows:
		return 0
	case nil:
		return players
	default:
		log.Println("Error retrieving player indices.")
		panic(err)
		return -1
	}
}

func (db *Database) GetGamePlayers(id string) []Player {
	rows, err := db.dbRef.Query(`select player_id,name,last_move,hand_state from game_players left join players on players.id=player_id where game_id=? order by player_index`, id)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	// TODO: fix this
	var players = make([]Player, 0, MaxPlayers)
	var i = 0
	for rows.Next() {
		var playerId, name, lastMove, handState string
		err = rows.Scan(&playerId, &name, &lastMove, &handState)
		if err != nil {
			log.Println("Error retrieving list of players in game.")
			log.Fatal(err)
		}

		cards, jsonErr := DecodePlayerHand(handState)
		if jsonErr != "" {
			log.Println("Error decoding player hand state stored in database.")
			log.Fatal(err)
		}

		players = append(players, Player{GoogleID: playerId, Name: name, LastMove: lastMove, Cards: cards})
		log.Printf("Artificially adding player %s (%s) to game %s", name, playerId, id)
		i++
	}
	return players
}

func (db *Database) LogMove(g Game, m Message, t int64) string {

	var mainPlayerSql = "turns=turns+1, "
	var gameSql = "turns=turns+1, "

	if !g.IgnoreTime {
		mainPlayerSql += "timed_turns=timed_turns+1, turn_time=turn_time+" + fmt.Sprint(t-g.LastUpdateTime) + ", "
		gameSql += "timed_turns=timed_turns+1, turn_time=turn_time+" + fmt.Sprint(t-g.LastUpdateTime) + ", "
	}

	if m.MoveType == MovePlay && m.Result == ResultPlay {
		mainPlayerSql += "plays=plays+1, "
		gameSql += "plays=plays+1, "
	} else if m.MoveType == MovePlay && m.Result == ResultBomb {
		mainPlayerSql += "bombs=bombs+1, "
		gameSql += "bombs=bombs+1, "
	} else if m.MoveType == MoveDiscard {
		mainPlayerSql += "discards=discards+1, "
		gameSql += "discards=discards+1, "
	} else if m.MoveType == MoveHint {
		mainPlayerSql += "hints=hints+1, "
		gameSql += "hints=hints+1, "
	}

	db.openTransaction()
	db.execWithinTransaction("update game_players set "+mainPlayerSql[:len(mainPlayerSql)-2]+" where player_id=? AND game_id=?", m.Player, g.ID)
	db.execWithinTransaction("update games set "+gameSql[:len(gameSql)-2]+" where id=?", g.ID)
	db.closeTransaction()

	g.LastUpdateTime = t
	if t > db.LastUpdateTime {
		db.LastUpdateTime = t
	}

	return ""
}

func (db *Database) execQuery(query string, args ...interface{}) {
	res, err := db.dbRef.Exec(query, args...)
	if err != nil {
		log.Printf("Error executing query: %s", query)
		log.Fatal(err)
	} else {
		rows, rowsErr := res.RowsAffected()
		if rowsErr != nil {
			log.Printf("Error calculating rows affected by query: %s", query)
			log.Fatal(rowsErr)
		} else {
			log.Printf("Ran query: %s", query)
			log.Printf("AFFECTED %d ROWS", rows)
		}
	}
}

func (db *Database) cleanup() {
	// NOT CURRENTLY IN USE
	db.openTransaction()
	db.execWithinTransaction(`delete from games where state=?`, StateNotStarted)
	db.execWithinTransaction(`delete from game_players where game_id in (select game_id from game_players left join games on game_id=id where id is null)`)
	db.closeTransaction()
}
