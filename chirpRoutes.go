package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func ValidateAndWriteChirp(w http.ResponseWriter, r *http.Request, apiCfg *apiConfig) {
	decoder := json.NewDecoder(r.Body)
	chirps := chirp{}
	err := decoder.Decode(&chirps)
	if err != nil {
		// an error will be thrown if the JSON is invalid or has the wrong types
		// any missing fields will simply have their values in the struct set to their zero value
		log.Printf("Error decoding parameters: %s", err)
		ResponseWithError(w, 500, "Something went wrong")
	}

	if len(chirps.Body) > 140 {
		ResponseWithError(w, 400, "Chirp is too long")
	}

	// Split string and replace profanity
	words := strings.Split(chirps.Body, " ")
	profanities := []string{"kerfuffle", "sharbert", "fornax"}
	for idx, word := range words {
		for _, profanity := range profanities {
			if strings.ToLower(word) == profanity {
				words[idx] = "****"
			}
		}
	}
	chirps.Body = strings.Join(words, " ")
	log.Printf("Clean chirp:%s", chirps.Body)

	dbChirp, err := apiCfg.dbClient.CreateChirp(chirps.Body)
	if err != nil {
		ResponseWithError(w, 500, "Error creating chirp in db")
	} else {
		resp, _ := json.Marshal(dbChirp)
		ResponseWithSuccess(w, 201, resp)
	}
}

func GetChirps(w http.ResponseWriter, r *http.Request, apiCfg *apiConfig) {
	chirps, err := apiCfg.dbClient.GetChirps()
	if err != nil {
		ResponseWithError(w, 500, "Unable to get chirps from db")
	}

	responseAsStr, err := json.Marshal(chirps)
	ResponseWithSuccess(w, 200, responseAsStr)
}

func GetChirpById(w http.ResponseWriter, r *http.Request, apiCfg *apiConfig) {
	id := r.PathValue("chirpID")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		ResponseWithError(w, 422, "Invalid id "+id)
	}

	chirps, err := apiCfg.dbClient.GetChirpById(idInt)
	log.Printf("chirp:%v, err: %v", chirps, err)
	if err != nil {
		ResponseWithError(w, 404, "Unable to get chirp from db")
	} else {
		responseAsStr, err := json.Marshal(chirps)
		if err != nil {
			ResponseWithError(w, 500, "Unable to marshal json")
		} else {
			ResponseWithSuccess(w, 200, responseAsStr)
		}
	}
}
