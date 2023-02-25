package lib

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"fmt"
)

const gameInsertSql = "insert into games (id, name, turns, timedturns, plays, bombs, discards, hints, numberhints, colorhints, state, score, mode, players) values ($1,$2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)"

const playerInsertSql = "insert into players (id, name) values ($1,$2)"

func MigrateToSqlite(l *Logger) bool {


	ignoreGames := [1]string{"helloooooo-1558986018"}
	
	db := deleteAndCreateDatabase(DatabaseFile)
	defer db.Close()
	
	
	createTables(db)
	
	
	for _, player := range l.Players {
		insertPlayer(db,player.ID,player.Name)
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
					
					state := determineFinalState(data,mode,gameScore)
						
					insertGame(db, game.ID, game.Name, data, state, gameScore, mode, players)
					
					
					
					foundGame = true
					break scanningForStatsBlob					
				}
			}
		}
		if !foundGame {
			fmt.Printf("Couldn't find stats blob for game %s.",game.Name)
			return false
		}
	}
/*
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare("insert into foo(id, name) values(?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	for i := 0; i < 100; i++ {
		_, err = stmt.Exec(i, fmt.Sprintf("こんにちは世界%03d", i))
		if err != nil {
			log.Fatal(err)
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	rows, err := db.Query("select id, name from foo")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		err = rows.Scan(&id, &name)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(id, name)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err = db.Prepare("select name from foo where id = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	var name string
	err = stmt.QueryRow("3").Scan(&name)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(name)

	_, err = db.Exec("delete from foo")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("insert into foo(id, name) values(1, 'foo'), (2, 'bar'), (3, 'baz')")
	if err != nil {
		log.Fatal(err)
	}

	rows, err = db.Query("select id, name from foo")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		err = rows.Scan(&id, &name)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(id, name)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}*/
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
						name text,
						turns int default 0 not null,
						timedTurns int default 0 not null,
						plays int default 0 not null,
						bombs int default 0 not null,
						discards int default 0 not null,
						hints int default 0 not null,
						numberHints int default 0 not null,
						colorHints int default 0 not null
						);
	create table games (id text not null primary key,
						name text
						lastMoveTime int,
						turns int default 0 not null,
						timedTurns int default 0 not null,
						plays int default 0 not null,
						bombs int default 0 not null,
						discards int default 0 not null,
						hints int default 0 not null,
						numberHints int default 0 not null,
						colorHints int default 0 not null,
						state int,
						score int default 0 not null,
						mode int,
						players int,
						gameStateData blob);
	create table gamePlayers (gameid text references games(id),
							playerid text references players(id),
							primary key (gameid, playerid));
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("Failed to create tables. %q: %s\n", err, sqlStmt)
	}
}

func insertPlayer(db *sql.DB, id string, name string) {
	
	_, err := db.Exec(playerInsertSql,id,name)
	if err != nil {
		log.Fatalf("Failed to insert player. %q: %s\n", err, playerInsertSql)
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
	
	if(gameScore < 0) {
		log.Fatalf("Couldn't find score for game.")
	}
	
	return gameScore
}

func determineFinalState(data StatLog, mode int, score int) int {
		highScore := 30
		if mode == ModeNormal { 
			highScore = 25
		}
					
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

func insertGame(db *sql.DB, id string, name string, data StatLog, state int, gameScore int, mode int, players int) {
	_, err := db.Exec(gameInsertSql,
					id,
					name,
					data.Turns,
					data.TimedTurns,
					data.Plays,
					data.Bombs,
					data.Discards,
					data.Hints,
					data.NumberHints,
					data.ColorHints,
					state,
					gameScore,
					mode,
					players)
					
	if err != nil {
		log.Fatalf("Failed to insert game. %q: %s\n", err, gameInsertSql)
	}
}