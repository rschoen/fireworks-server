package main

import (
	"fireworks-server/lib"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"
)

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	// allow requests to come from anywhere, since clients can be wherever
	w.Header().Set("Access-Control-Allow-Origin", "*")

	apiPrefix := "/apiv2/"
	if len(r.URL.Path) >= len(apiPrefix) && r.URL.Path[:len(apiPrefix)] == apiPrefix {
		// nothing
	} else if s.fileServer {
		// serve client HTTP responses, if it's turned on
		http.FileServer(http.Dir(s.clientDirectory)).ServeHTTP(w, r)
		return
	} else {
		return
	}

	var command = r.URL.Path[len(apiPrefix):]
	if command == "version" {
		fmt.Fprintf(w, lib.VERSION)
		return
	}
	if command == "stats" {

		statsLog := s.db.CreateStatsMessage()

		json, err := lib.EncodeStatsMessage(statsLog)
		if err != "" {
			log.Printf("Failed to encode stats log. Error: %s\n", err)
			return
		}
		fmt.Fprint(w, json)
		return
	}
	if command == "clean" {
		s.db.CleanupUnstartedGames()
		fmt.Fprint(w, "")
		return
	}

	m, err := lib.DecodeMove(r.PostFormValue("data"))
	if err != "" {
		log.Println("Discarding malformed JSON message. Error: " + err)
		fmt.Fprint(w, jsonError("Data sent was malformed."))
		return
	}
	var selectedGame *lib.Game

	// Authenticate user
	authResponse, authError := s.auth.Authenticate(m.Token)
	if authError != "" {
		log.Printf("Failed to authenticate player '%s' in game '%s'. Error: %s\n", m.Player, m.Game, authError)
		fmt.Fprint(w, jsonError("You appear to be signed out. Please refresh and try signing in again."))
		return
	}
	if authResponse.GetGoogleID() != m.Player && !s.disableAuth {
		log.Printf("Authenticated player '%s' submitted move as player '%s' in game '%s'.", authResponse.GetGoogleID(), m.Player, m.Game)
		fmt.Fprint(w, jsonError("Authenticated as a different user."))
		return
	}

	if command == "list" {
		list := lib.GamesList{}
		playersGames := s.db.GetGamesPlayerIsIn(m.Player)
		for _, gameId := range playersGames {
			game := s.games[gameId]
			if !lib.GameStateIsFinished(game.State) {

				playerList := ""
				for player := range game.Players {
					playerList += game.Players[player].Name + ", "
				}

				if playerList != "" {
					playerList = playerList[:len(playerList)-2]
				}
				gameMessage := lib.MinimalGame{ID: game.ID, Name: game.Name, Players: playerList, Mode: game.Mode}
				list.PlayersGames = append(list.PlayersGames, gameMessage)
			}
		}

		joinableGames := s.db.GetJoinableGames()
		for _, gameId := range joinableGames {
			game := s.games[gameId]
			if game.State == lib.StateNotStarted && len(game.Players) < lib.MaxPlayers && game.Public {

				playerList := ""
				for player := range game.Players {
					playerList += game.Players[player].Name + ", "
				}

				if playerList != "" {
					playerList = playerList[:len(playerList)-2]
				}
				gameMessage := lib.MinimalGame{ID: game.ID, Name: game.Name, Players: playerList, Mode: game.Mode}
				list.OpenGames = append(list.OpenGames, gameMessage)
			}
		}

		encodedList, err := lib.EncodeList(list)
		if err != "" {
			log.Printf("Failed to encode game list. Error: %s\n", err)
			fmt.Fprint(w, jsonError("Could not transmit game list to client."))
			return
		}
		fmt.Fprint(w, encodedList)
		return
	}

	if command == "create" {
		selectedGame = new(lib.Game)
		selectedGame.Name = sanitizeAndTrim(m.Game, lib.MaxGameNameLength, false)
		selectedGame.ID = selectedGame.Name + "-" + strconv.FormatInt(time.Now().Unix(), 10)

		var initializationError = selectedGame.Initialize(m.Public, m.IgnoreTime, m.SighButton, m.GameMode)
		if initializationError != "" {
			log.Printf("Failed to initialize game '%s'. Error: %s\n", m.Game, initializationError)
			fmt.Fprint(w, jsonError("Could not initialize game."))
			return
		}
		s.db.CreateGame(*selectedGame)
		s.games[selectedGame.ID] = selectedGame
		m.Game = selectedGame.ID
		log.Printf("Created new game '%s'\n", m.Game)

		command = "join"
	}

	if _, ok := s.games[m.Game]; ok {
		selectedGame = s.games[m.Game]
	} else {
		log.Printf("Attempting to make a move on a nonexistent game '%s'\n", m.Game)
		fmt.Fprint(w, jsonError("The game you're attempting to play no longer exists."))
		return
	}

	if command == "join" {
		player := selectedGame.GetPlayerByGoogleID(m.Player)
		// add player if it doesn't exist
		if player == nil {
			log.Printf("Player not in game already, adding now!")
			if len(selectedGame.Players) >= lib.MaxPlayers {
				log.Printf("Attempting to add player '%s' to full game '%s'\n", m.Player, m.Game)
				fmt.Fprint(w, jsonError("This game is now full."))
				return
			}
			playerName := sanitizeAndTrim(authResponse.GetGivenName(), lib.MaxPlayerNameLength, true)
			addError := selectedGame.AddPlayer(m.Player, playerName)
			s.db.CreatePlayerIfNotExists(m.Player, playerName)
			s.db.AddPlayer(m.Player, selectedGame.ID)
			if addError != "" {
				log.Printf("Error adding player '%s' to game '%s'. Error: %s\n", m.Player, m.Game, addError)
				fmt.Fprint(w, jsonError("Unable to join this game."))
				return
			}
			log.Printf("Added player '%s' to game '%s'\n", playerName, selectedGame.Name)
		}
	}

	player := selectedGame.GetPlayerByGoogleID(m.Player)

	if player == nil {
		log.Printf("Attempting to make a move with nonexistent player '%s'\n", m.Player)
		fmt.Fprint(w, jsonError("You're not a member of this game."))
		return
	}

	if command == "start" {
		if selectedGame.State != lib.StateNotStarted {
			log.Printf("Attempting to start already started game '%s'\n", m.Game)
			fmt.Fprint(w, jsonError("This game has already started."))
			return
		}
		log.Printf("Gonna start game %s with table %+v", selectedGame.ID, selectedGame.Table)
		var startError = selectedGame.Start()
		if startError != "" {
			log.Printf("Failed to start game '%s'. Error: %s\n", m.Game, startError)
			fmt.Fprint(w, jsonError("Could not start game."))
			return
		}
		s.db.SaveGameToDatabase(selectedGame)
		log.Printf("Started game '%s'\n", m.Game)
	}

	if command == "announce" {
		var processError = selectedGame.ProcessAnnouncement(&m)
		if processError != "" {
			log.Printf("Failed to process announcement for game '%s'. Error: %s\n", m.Game, processError)
			fmt.Fprint(w, jsonError("Could not process announcement."))
			return
		}
		log.Printf("Processed announcement by player '%s' in game '%s'\n", m.Player, m.Game)
	}

	if command == "move" {

		var processError = selectedGame.ProcessMove(&m)
		if processError != "" {
			log.Printf("Failed to process move for game '%s'. Error: %s\n", m.Game, processError)
			fmt.Fprint(w, jsonError("Could not process move."))
			return
		}
		logError := s.db.LogMove(*selectedGame, m, time.Now().Unix())
		if logError != "" {
			log.Printf("Failed to log move for game '%s'. Error: %s\n", m.Game, logError)
			fmt.Fprint(w, jsonError("Could not log move."))
			return
		}
		s.db.SaveGameToDatabase(selectedGame)
		player.PushToken = m.PushToken
		log.Printf("Processed and logged move by player '%s' in game '%s'\n", m.Player, m.Game)
	}

	if command == "status" {
		if m.LastTurn == selectedGame.Table.Turn && m.UpdateTime == selectedGame.LastUpdateTime {
			fmt.Fprint(w, "")
			return
		}
	}

	encodedGame, err := lib.EncodeGame(selectedGame.CreateState(m.Player))
	if err != "" {
		log.Printf("Failed to encode game '%s'. Error: %s\n", m.Game, err)
		fmt.Fprint(w, jsonError("Could not transmit game state to client."))
		return
	}
	fmt.Fprint(w, encodedGame)
}

func jsonError(err string) string {
	return "{\"error\":\"" + strings.Replace(err, "\"", "\\\"", -1) + "\"}"
}

func sanitizeAndTrim(text string, limit int, oneword bool) string {
	re := regexp.MustCompile(`[^A-Za-z0-9 _!,\.-]+`)
	text = re.ReplaceAllString(text, "")
	if oneword && strings.Contains(text, " ") {
		text = text[:strings.Index(text, " ")]
	}
	if len(text) > limit {
		return text[:limit]
	}
	return text
}

type Server struct {
	games           map[string]*lib.Game
	db              *lib.Database
	fileServer      bool
	auth            lib.Authenticator
	disableAuth     bool
	clientDirectory string
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	// initialize server
	s := Server{}
	s.games = make(map[string]*lib.Game)

	fileServer := flag.Bool("file-server", false, "Whether to serve files in addition to game API.")
	https := flag.Bool("https", false, "Whether to serve everything over HTTPS instead of HTTP")
	clientDirectory := flag.String("client-directory", lib.DefaultClientDirectory, "Directory to serve HTTP responses from (fireworks-client directory)")
	port := flag.Int("port", lib.DefaultPort, "Port to listen for connections from client.")
	cert := flag.String("certificate", lib.DefaultCertificate, "Path to SSL certificate file, only used if using --http")
	key := flag.String("key", lib.DefaultKey, "Path to SSL key file, only used if using --http")
	databaseFile := flag.String("database", lib.DefaultDatabaseFile, "File to use as database, defaults to "+lib.DefaultDatabaseFile)
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	disableAuth := flag.Bool("disable-auth", false, "Disable authentication for testing")
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	s.disableAuth = *disableAuth
	s.auth.Initialize(*disableAuth)

	s.fileServer = *fileServer
	s.clientDirectory = *clientDirectory
	http.HandleFunc("/", s.handler)
	portString := ":" + strconv.Itoa(*port)

	log.Println("Loading database...")
	s.db = new(lib.Database)
	s.db.Connect(*databaseFile)
	s.games = s.db.GetActiveGames()

	log.Println("Ready to go!")

	if *https {
		log.Fatal(http.ListenAndServeTLS(portString, *cert, *key, nil))
	} else {
		log.Fatal(http.ListenAndServe(portString, nil))
	}

}
