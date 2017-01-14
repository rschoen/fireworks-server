package main

import (
	"fireworks-server/lib"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	// allow requests to come from anywhere, since clients can be wherever
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// serve client HTTP responses, if it's turned on
	if len(r.URL.Path) < 5 || r.URL.Path[1:5] != "api/" {
		if s.fileServer {
			http.FileServer(http.Dir(s.clientDirectory)).ServeHTTP(w, r)
		}
		return
	}

	var command = r.URL.Path[5:]

	if command == "stats" {
		json, err := lib.EncodeStatsLog(s.logger.CreateStatsLog())
		if err != "" {
			log.Printf("Failed to encode stats log. Error: %s\n", err)
			return
		}
		fmt.Fprintf(w, json)
		return
	}

	m, err := lib.DecodeMove(r.PostFormValue("data"))
	if err != "" {
		log.Println("Discarding malformed JSON message. Error: " + err)
		fmt.Fprintf(w, jsonError("Data sent was malformed."))
		return
	}
	var game *lib.Game
	for _, ongoingGame := range s.games {
		if ongoingGame.ID == m.Game {
			game = ongoingGame
		}
	}

	// Authenticate user
	authResponse, authError := lib.Authenticate(m.Token)
	if authError != "" {
		log.Printf("Failed to authenticate player '%s' in game '%s'. Error: %s\n", m.Player, m.Game, authError)
		fmt.Fprintf(w, jsonError("You appear to be signed out. Please refresh and try signing in again."))
		return
	}
	if authResponse.GetGoogleID() != m.Player {
		log.Printf("Authenticated player '%s' submitted move as player '%s' in game '%s'.", authResponse.GetGoogleID(), m.Player, m.Game, m.Game)
		fmt.Fprintf(w, jsonError("Authenticated as a different user."))
		return
	}

	if command == "list" {
		list := lib.GamesList{}
		for i, _ := range s.games {
		 state := s.games[i].State
		 if state != lib.StateNotStarted && state != lib.StateStarted {
		  continue
		 }
		 
			playerList := ""
			addGame := false
			for player, _ := range s.games[i].Players {
				playerList += s.games[i].Players[player].Name + ", "
				if s.games[i].Players[player].GoogleID == m.Player {
					addGame = true
				}
			}
			if playerList != "" {
				playerList = playerList[:len(playerList)-2]
			}
			game := lib.MinimalGame{ID: s.games[i].ID, Name: s.games[i].Name, Players: playerList, Mode: s.games[i].Mode}

			if addGame {
				list.PlayersGames = append(list.PlayersGames, game)
			} else if state == lib.StateNotStarted && len(s.games[i].Players) < lib.MaxPlayers && s.games[i].Public == true {
				list.OpenGames = append(list.OpenGames, game)
			}
		}

		encodedList, err := lib.EncodeList(list)
		if err != "" {
			log.Printf("Failed to encode game list. Error: %s\n", err)
			fmt.Fprintf(w, jsonError("Could not transmit game list to client."))
			return
		}
		fmt.Fprintf(w, encodedList)
		return
	}

	if command == "create" {
		game = new(lib.Game)
		game.Name = sanitizeAndTrim(m.Game, lib.MaxGameNameLength, false)
		game.ID = game.Name + "-" + strconv.FormatInt(time.Now().Unix(), 10)

		var initializationError = game.Initialize(m.Public, m.GameMode, m.StartingHints, m.MaxHints, m.StartingBombs)
		if initializationError != "" {
			log.Printf("Failed to initialize game '%s'. Error: %s\n", m.Game, initializationError)
			fmt.Fprintf(w, jsonError("Could not initialize game."))
			return
		}
		s.games = append(s.games, game)
		log.Printf("Created new game '%s'\n", m.Game)

		command = "join"
	}
    
    if game == nil {
		log.Printf("Attempting to make a move on a nonexistent game '%s'\n", m.Game)
		fmt.Fprintf(w, jsonError("The game you're attempting to play no longer exists."))
		return
	}

	if command == "join" {
		player := game.GetPlayerByGoogleID(m.Player)
		// add player if it doesn't exist
		if player == nil {
			if len(game.Players) >= lib.MaxPlayers {
				log.Printf("Attempting to add player '%s' to full game '%s'\n", m.Player, m.Game)
				fmt.Fprintf(w, jsonError("This game is now full."))
				return
			}
			playerName := sanitizeAndTrim(authResponse.GetGivenName(), lib.MaxPlayerNameLength, true)
			addError := game.AddPlayer(m.Player, playerName)
			if addError != "" {
				log.Printf("Error adding player '%s' to game '%s'. Error: %s\n", m.Player, m.Game, addError)
				fmt.Fprintf(w, jsonError("Unable to join this game."))
				return
			}
			log.Printf("Added player '%s' to game '%s'\n", playerName, game.Name)
		}
	}

	player := game.GetPlayerByGoogleID(m.Player)

	if player == nil {
		log.Printf("Attempting to make a move with nonexistent player '%s'\n", m.Player)
		fmt.Fprintf(w, jsonError("You're not a member of this game."))
		return
	}

	if command == "start" {
		if game.State != lib.StateNotStarted {
			log.Printf("Attempting to start already started game '%s'\n", m.Game)
			fmt.Fprintf(w, jsonError("This game has already started."))
			return
		}
		var startError = game.Start()
		if startError != "" {
			log.Printf("Failed to start game '%s'. Error: %s\n", m.Game, startError)
			fmt.Fprintf(w, jsonError("Could not start game."))
			return
		}
		log.Printf("Started game '%s'\n", m.Game)
	}

	if command == "move" {
		if m.MoveType == lib.MoveHint && game.Hints <= 0 {
			fmt.Fprintf(w, jsonError("There are no hints left. Discard to earn more hints."))
			return
		}
		var processError = game.ProcessMove(&m)
		if processError != "" {
			log.Printf("Failed to process move for game '%s'. Error: %s\n", m.Game, processError)
			fmt.Fprintf(w, jsonError("Could not process move."))
			return
		}
		logError := s.logger.LogMove(*game, m, time.Now().Unix(), true)
		if logError != "" {
			log.Printf("Failed to log move for game '%s'. Error: %s\n", m.Game, logError)
			fmt.Fprintf(w, jsonError("Could not log move."))
			return
		}
		log.Printf("Processed and logged move by player '%s' in game '%s'\n", m.Player, m.Game)
	}

	encodedGame, err := lib.EncodeGame(game.CreateState(m.Player))
	if err != "" {
		log.Printf("Failed to encode game '%s'. Error: %s\n", m.Game, err)
		fmt.Fprintf(w, jsonError("Could not transmit game state to client."))
		return
	}
	fmt.Fprintf(w, encodedGame)
}

func jsonError(err string) string {
	return "{\"error\":\"" + strings.Replace(err, "\"", "\\\"", -1) + "\"}"
}

func sanitizeAndTrim(text string, limit int, oneword bool) string {
	re := regexp.MustCompile("[^A-Za-z0-9 _!,\\.-]+")
	text = re.ReplaceAllString(text, "")
	if oneword && strings.Index(text, " ") > -1 {
		text = text[:strings.Index(text, " ")]
	}
	if len(text) > limit {
		return text[:limit]
	}
	return text
}

type Server struct {
	games           []*lib.Game
	logger          *lib.Logger
	fileServer      bool
	clientDirectory string
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	// initialize server
	s := Server{}
	s.games = make([]*lib.Game, 0, lib.MaxConcurrentGames)

	fileServer := flag.Bool("file-server", false, "Whether to serve files in addition to game API.")
	https := flag.Bool("https", false, "Whether to serve everything over HTTPS instead of HTTP")
	clientDirectory := flag.String("client-directory", lib.DefaultClientDirectory, "Directory to serve HTTP responses from (fireworks-client directory)")
	port := flag.Int("port", lib.DefaultPort, "Port to listen for connections from client.")
	cert := flag.String("certificate", lib.DefaultCertificate, "Path to SSL certificate file, only used if using --http")
	key := flag.String("key", lib.DefaultKey, "Path to SSL key file, only used if using --http")
	logDir := flag.String("logdir", lib.DefaultLogDirectory, "Path to log directory, defaults to ./log/")
	flag.Parse()

	s.fileServer = *fileServer
	s.clientDirectory = *clientDirectory
	http.HandleFunc("/", s.handler)
	portString := ":" + strconv.Itoa(*port)

	// set up the logger and reconsitute games in progress
	s.logger = new(lib.Logger)
	s.logger.Directory = *logDir
	fmt.Println("Re-constituting games in progress.")
	games, loggerError := s.logger.Initialize()
	if loggerError != "" {
		log.Fatal("Failed to initialize logger. Error: " + loggerError)
	}
	s.games = append(s.games, games...)
	fmt.Printf("Re-constituted %d games.\n", len(games))

	if *https {
		log.Fatal(http.ListenAndServeTLS(portString, *cert, *key, nil))
	} else {
		log.Fatal(http.ListenAndServe(portString, nil))
	}

}
