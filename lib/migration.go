package lib

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func MigrateToSqlite(l *Logger) bool {

	ignoreGames := [12]string{"helloooooo-1558986018", "ht-1514332240",
		"ryan tests the nabs-1677375807", "newpinabi-1545197046",
		"ryan!-1552274353", "jakenabi-1573330579", "game-1577335419",
		"Ryans test game-1668884223", "game1-1584585659","shaved pup-1588205875",
		"burgers-1591760095", "wipeout-1644466001"}

	db := deleteAndCreateDatabase(DatabaseFile)
	defer db.Close()

	createTables(db)

	for _, player := range l.Players {
		insertPlayer(db, player.ID, player.Name)

		for mode, playerList := range player.Stats.ModesAndPlayers {
			for numPlayers, data := range playerList {
				if data.Turns > 0 {
					insertLegacyPlayerStats(db, player.ID, mode, numPlayers, data)
				}
			}
		}
	}

gameLoop:
	for _, game := range l.Games {

		for _, id := range ignoreGames {
			if id == game.ID {
				continue gameLoop
			}
		}

		foundGame := false
	scanningForStatsBlob:
		for mode, playerList := range game.Stats.ModesAndPlayers {
			for players, data := range playerList {
				if len(data.Scores) > 0 {

					gameScore := getScoreFromScoreList(data.Scores)

					state := determineFinalState(data, mode, gameScore)

					insertGame(db, game.ID, game.Name, game.LastMoveTime, data, state, gameScore, mode, players)

					foundGame = true
					break scanningForStatsBlob
				}
			}
		}
		if !foundGame {
			fmt.Printf("Couldn't find stats blob for game %s.\n", game.ID)
			//return false
		}
	}

	return true
}

func deleteAndCreateDatabase(file string) *sql.DB {
	os.Remove(file)

	db, err := sql.Open("sqlite3", file)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func createTables(db *sql.DB) {
	sqlStmt := `
	create table players (id text not null primary key,
						name text
						);
	create table games (id text not null primary key,
						name text,
						time_started int,
						last_move_time int,
						turns int default 0 not null,
						timed_turns int default 0 not null,
						turn_time int default 0 not null,
						game_time int default 0 not null,
						plays int default 0 not null,
						bombs int default 0 not null,
						discards int default 0 not null,
						hints int default 0 not null,
						state int,
						score int default 0 not null,
						mode int,
						players int,
						initialized bool default false not null,
						public bool default true not null,
						ignore_time bool default false not null,
						sigh_button bool default false not null,
						current_player_index int,
						table_state blob);

	create table game_players (game_id text references games(id),
							player_id text references players(id),
				 			player_index int,
							primary key (game_id, player_id));

	create table legacy_player_stats (id text references players(id),
									mode int not null,
									players int not null,
									finished_games int not null,
									turns int default 0 not null,
									timed_turns int default 0 not null,
									turn_time int default 0 not null,
									game_time int default 0 not null,
									plays int default 0 not null,
									bombs int default 0 not null,
									discards int default 0 not null,
									hints int default 0 not null,
									bombs_losses int default 0 not null,
									turns_losses int default 0 not null,
									no_plays_losses int default 0 not null,
									score_list text not null,
									primary key (id, mode, players));

	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("Failed to create tables. %q: %s\n", err, sqlStmt)
	}
}

const playerInsertSql = "insert into players (id, name) values ($1,$2)"

func insertPlayer(db *sql.DB, id string, name string) {

	_, err := db.Exec(playerInsertSql, id, name)
	if err != nil {
		log.Fatalf("Failed to insert player. %q: %s\n", err, playerInsertSql)
	}
}

const legacyPlayerStatSql = "insert into legacy_player_stats (id, mode, players, finished_games, turns, timed_turns, turn_time, game_time, plays, bombs, discards, hints, bombs_losses, turns_losses, no_plays_losses, score_list) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)"

func insertLegacyPlayerStats(db *sql.DB, id string, mode int, players int, data StatLog) {

	_, err := db.Exec(legacyPlayerStatSql,
		id,
		mode,
		players,
		data.FinishedGames,
		data.Turns,
		data.TimedTurns,
		data.TurnTime,
		data.GameTime,
		data.Plays,
		data.Bombs,
		data.Discards,
		data.Hints,
		data.BombsLosses,
		data.TurnsLosses,
		data.NoPlaysLosses,
		fmt.Sprint(data.Scores))
	if err != nil {
		log.Fatalf("Failed to insert legacy player stat. %q: %s\n", err, legacyPlayerStatSql)
	}
}

func getScoreFromScoreList(list []int) int {
	gameScore := -1
	for score, thisGame := range list {
		if thisGame == 1 {
			gameScore = score
			break
		}
	}

	if gameScore < 0 {
		log.Fatalf("Couldn't find score for game.")
	}

	return gameScore
}

func determineFinalState(data StatLog, mode int, score int) int {
	var highScore = PerfectScoreForMode(mode)

	if data.BombsLosses == 1 {
		return StateBombedOut
	} else if data.TurnsLosses == 1 {
		return StateDeckEmpty
	} else if data.NoPlaysLosses == 1 {
		return StateNoPlays
	} else if score == highScore {
		return StatePerfect
	}

	log.Fatal("Could not detect state of game. Is it still in progress???")
	return -1
}

const gameInsertSql = "insert into games (id, name, time_started, last_move_time, turns, timed_turns, turn_time, game_time, plays, bombs, discards, hints, state, score, mode, players) values ($1,$2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)"

func insertGame(db *sql.DB, id string, name string, lastMoveTime int64, data StatLog, state int, gameScore int, mode int, players int) {

	startTime := startTimeFromGameID(id)

	_, err := db.Exec(gameInsertSql,
		id,
		name,
		startTime,
		lastMoveTime,
		data.Turns,
		data.TimedTurns,
		data.TurnTime,
		data.GameTime,
		data.Plays,
		data.Bombs,
		data.Discards,
		data.Hints,
		state,
		gameScore,
		mode,
		players)

	if err != nil {
		log.Fatalf("Failed to insert game. %q: %s\n", err, gameInsertSql)
	}
}

func startTimeFromGameID(id string) int {
	dashIndex := strings.LastIndex(id, "-")
	if dashIndex == -1 {
		log.Fatalf("Malformed ID string: %s", id)
	}
	startTime, error := strconv.Atoi(id[dashIndex+1:])
	if error != nil {
		log.Fatal(error)
	}
	return startTime
}
