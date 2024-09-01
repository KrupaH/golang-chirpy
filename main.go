package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	database "github.com/KrupaH/golang-chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits int
	dbClient       database.DB
	jwtSecret      string
}

type chirp struct {
	Body string `json:"body"`
	Id   int    `json:"id"`
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits += 1
		next.ServeHTTP(w, r)
	})
}

func ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (apiConfig *apiConfig) GetHits(w http.ResponseWriter, r *http.Request) {
	htmlContent, _ := os.ReadFile("metrics.html")
	responseText := fmt.Sprintf(string(htmlContent), apiConfig.fileserverHits)
	w.Write([]byte(responseText))
	w.Header().Set("Content-Type", "text/html")
}

func (apiConfig *apiConfig) ResetHits(w http.ResponseWriter, r *http.Request) {
	apiConfig.fileserverHits = 0
	w.Write([]byte("OK"))
}

func ResponseWithError(w http.ResponseWriter, code int, message string) {
	type errResp struct {
		Error string `json:"error"`
	}
	body, _ := json.Marshal(errResp{Error: message})
	w.WriteHeader(code)
	w.Write(body)
}

func ResponseWithSuccess(w http.ResponseWriter, code int, message []byte) {
	w.WriteHeader(code)
	w.Write(message)
}

func apiCfgHandler(apiCfg *apiConfig, fn func(http.ResponseWriter, *http.Request, *apiConfig)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, apiCfg)
	}
}

func main() {

	// by default, godotenv will look for a file named .env in the current directory
	godotenv.Load()
	jwtSecret := os.Getenv("JWT_SECRET")

	mux := http.NewServeMux()
	port := "8080"
	filepathRoot := "."
	dbFpath := "database.json"

	dbg := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()
	if *dbg {
		os.Remove(dbFpath)
	}

	db, err := database.NewDB(dbFpath)

	if err != nil {
		panic("Unable to connect db")
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	apiConfig := apiConfig{dbClient: *db, jwtSecret: jwtSecret}

	appHandler := http.FileServer(http.Dir(filepathRoot))
	mux.Handle("/app/", http.StripPrefix("/app/", apiConfig.middlewareMetricsInc(appHandler)))
	mux.HandleFunc("GET /api/healthz/", ReadinessCheck)
	mux.HandleFunc("GET /admin/metrics/", apiConfig.GetHits)
	mux.HandleFunc("GET /api/reset/", apiConfig.ResetHits)

	mux.HandleFunc("POST /api/chirps", apiCfgHandler(&apiConfig, ValidateAndWriteChirp))
	mux.HandleFunc("GET /api/chirps", apiCfgHandler(&apiConfig, GetChirps))
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfgHandler(&apiConfig, GetChirpById))
	mux.HandleFunc("POST /api/users", apiCfgHandler(&apiConfig, WriteUser))
	mux.HandleFunc("POST /api/login", apiCfgHandler(&apiConfig, LoginUser))
	mux.HandleFunc("PUT /api/users", apiCfgHandler(&apiConfig, UpdateUser))

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())
}
