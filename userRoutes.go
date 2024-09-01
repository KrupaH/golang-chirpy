package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	auth "github.com/KrupaH/golang-chirpy/internal/auth"
	"github.com/KrupaH/golang-chirpy/internal/database"
	"github.com/golang-jwt/jwt/v5"
)

func WriteUser(w http.ResponseWriter, r *http.Request, apiCfg *apiConfig) {
	user := database.User{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&user)
	if err != nil {
		ResponseWithError(w, 422, "Bad request: unable to decode json")
	}
	dbUser, err := apiCfg.dbClient.CreateUser(user)
	if err != nil {
		ResponseWithError(w, 500, "Internal error creating user: "+err.Error())
	} else {
		resp, _ := json.Marshal(dbUser)
		ResponseWithSuccess(w, 201, resp)
	}
}

func LoginUser(w http.ResponseWriter, r *http.Request, apiCfg *apiConfig) {

	type login struct {
		Email              string `json:"email"`
		Password           string `json:"password"`
		Expires_in_seconds int    `json:"expires_in_seconds"`
	}
	user := login{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&user)
	if err != nil {
		ResponseWithError(w, 422, "Bad request format")
	}

	existingUser, ok := apiCfg.dbClient.GetUserIdByEmail(user.Email)
	if !ok {
		ResponseWithError(w, 404, "User not found")
	}
	cmp := database.CheckPasswordsEqual(existingUser.Password, user.Password)
	if cmp {
		existingUser.Password = ""
		// Get JWT token
		token := auth.GetJWTToken(user.Expires_in_seconds, existingUser.Id, apiCfg.jwtSecret)
		type respFormat struct {
			Email string `json:"email"`
			Id    int    `json:"id"`
			Token string `json:"token"`
		}
		response := respFormat{Email: existingUser.Email, Id: existingUser.Id, Token: token}
		resp, _ := json.Marshal(response)
		ResponseWithSuccess(w, 200, resp)
	} else {
		ResponseWithError(w, 401, "Unauthorized")
	}

}

func UpdateUser(w http.ResponseWriter, r *http.Request, apiCfg *apiConfig) {
	type user struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	auth := r.Header.Get("Authorization")
	tokenString, _ := strings.CutPrefix(auth, "Bearer ")
	fmt.Println("token:", tokenString)

	jwtToken, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(apiCfg.jwtSecret), nil
		},
	)
	if err != nil {
		log.Print(err)
		ResponseWithError(w, 401, "Unauthorized Bearer token")
	} else {
		user_id, _ := jwtToken.Claims.GetSubject()

		user := database.User{}
		decoder := json.NewDecoder(r.Body)
		decoder.Decode(&user)
		user.Id, _ = strconv.Atoi(user_id)
		fmt.Print(user)

		user, _ = apiCfg.dbClient.UpdateUser(user)
		respData, _ := json.Marshal(user)
		ResponseWithSuccess(w, 200, respData)
	}
}
