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
		// nuttin
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

		// TODO: replace this to be some sort of JSON message format rather than Logger
		statsLog := s.db.CreateStatsLog()

		json, err := lib.EncodeStatsLog(statsLog)
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
	var game lib.Game
	// TODO: function to look up game in DB

	// Authenticate user
	authResponse, authError := s.auth.Authenticate(m.Token)
	if authError != "" {
		log.Printf("Failed to authenticate player '%s' in game '%s'. Error: %s\n", m.Player, m.Game, authError)
		fmt.Fprintf(w, jsonError("You appear to be signed out. Please refresh and try signing in again."))
		return
	}
	if authResponse.GetGoogleID() != m.Player {
		log.Printf("Authenticated player '%s' submitted move as player '%s' in game '%s'.", authResponse.GetGoogleID(), m.Player, m.Game)
		fmt.Fprintf(w, jsonError("Authenticated as a different user."))
		return
	}

	if command == "list" {
		list := lib.GamesList{}
		games := s.db.GetGameListForPlayer(m.Player)
		for _, game := range s.games {
			if (game.State == lib.StateNotStarted) ||
				(contains(games, game.ID) && !lib.GameStateIsFinished(game.State) &&
					len(game.Players) < lib.MaxPlayers && game.Public == true) {

				playerList := ""
				for player, _ := range s.games[game.ID].Players {
					playerList += s.games[game.ID].Players[player].Name + ", "
				}

				if playerList != "" {
					playerList = playerList[:len(playerList)-2]
				}
				game := lib.MinimalGame{ID: game.ID, Name: game.Name, Players: playerList, Mode: game.Mode}

				if contains(games, game.ID) {
					list.PlayersGames = append(list.PlayersGames, game)
				} else {
					list.OpenGames = append(list.OpenGames, game)
				}
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
		// TODO: store this in database!!! the correct way
		game = lib.Game{}
		game.Name = sanitizeAndTrim(m.Game, lib.MaxGameNameLength, false)
		game.ID = game.Name + "-" + strconv.FormatInt(time.Now().Unix(), 10)

		var initializationError = game.Initialize(m.Public, m.IgnoreTime, m.SighButton, m.GameMode)
		if initializationError != "" {
			log.Printf("Failed to initialize game '%s'. Error: %s\n", m.Game, initializationError)
			fmt.Fprintf(w, jsonError("Could not initialize game."))
			return
		}
		s.games[game.ID] = game
		log.Printf("Created new game '%s'\n", m.Game)

		command = "join"
	}

	if containsKey(s.games, m.Game) {
		game = s.games[m.Game]
	} else {
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
			s.db.AddPlayer(m.Player, game.ID)
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

	if command == "announce" {
		var processError = game.ProcessAnnouncement(&m)
		if processError != "" {
			log.Printf("Failed to process announcement for game '%s'. Error: %s\n", m.Game, processError)
			fmt.Fprintf(w, jsonError("Could not process announcement."))
			return
		}
		log.Printf("Processed announcement by player '%s' in game '%s'\n", m.Player, m.Game)
	}

	if command == "move" {

		var processError = game.ProcessMove(&m)
		if processError != "" {
			log.Printf("Failed to process move for game '%s'. Error: %s\n", m.Game, processError)
			fmt.Fprintf(w, jsonError("Could not process move."))
			return
		}
		logError := s.db.LogMove(game, m, time.Now().Unix())
		if logError != "" {
			log.Printf("Failed to log move for game '%s'. Error: %s\n", m.Game, logError)
			fmt.Fprintf(w, jsonError("Could not log move."))
			return
		}
		s.db.SaveGameToDatabase(game)
		player.PushToken = m.PushToken
		log.Printf("Processed and logged move by player '%s' in game '%s'\n", m.Player, m.Game)
	}

	if command == "status" {
		if m.LastTurn == game.Table.Turn && m.UpdateTime == game.LastUpdateTime {
			fmt.Fprintf(w, "")
			return
		}
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

// TODO: get rid of the logger altogether
type Server struct {
	games           map[string]lib.Game
	logger          *lib.Logger
	db              *lib.Database
	fileServer      bool
	auth            lib.Authenticator
	clientDirectory string
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	// initialize server
	s := Server{}
	s.games = make(map[string]lib.Game)
	s.auth.Initialize()

	fileServer := flag.Bool("file-server", false, "Whether to serve files in addition to game API.")
	https := flag.Bool("https", false, "Whether to serve everything over HTTPS instead of HTTP")
	clientDirectory := flag.String("client-directory", lib.DefaultClientDirectory, "Directory to serve HTTP responses from (fireworks-client directory)")
	port := flag.Int("port", lib.DefaultPort, "Port to listen for connections from client.")
	cert := flag.String("certificate", lib.DefaultCertificate, "Path to SSL certificate file, only used if using --http")
	key := flag.String("key", lib.DefaultKey, "Path to SSL key file, only used if using --http")
	databaseFile := flag.String("database", lib.DefaultDatabaseFile, "File to use as database, defaults to "+lib.DefaultDatabaseFile)
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	s.fileServer = *fileServer
	s.clientDirectory = *clientDirectory
	http.HandleFunc("/", s.handler)
	portString := ":" + strconv.Itoa(*port)

	// set up the logger and reconsitute games in progress
	log.Println("Loading database...")
	s.logger = new(lib.Logger)
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

//delete when you find the right one...

func contains(list []string, element string) bool {
	for _, elementToCheck := range list {
		if elementToCheck == element {
			return true
		}
	}
	return false
}

/*func contains(list []int, element int) bool {
	for _, elementToCheck := range list {
		if elementToCheck == element {
			return true
		}
	}
	return false
}*/

func containsKey(list map[string]lib.Game, element string) bool {
	for key, _ := range list {
		if key == element {
			return true
		}
	}
	return false
}
