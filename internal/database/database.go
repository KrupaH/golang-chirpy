package database

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

type DB struct {
	path   string
	mux    *sync.RWMutex
	lastid int
}

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

type User struct {
	Id       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
	Users  map[int]User  `json:"users"`
}

// NewDB creates a new database connection
// and creates the database file if it doesn't exist
func NewDB(path string) (*DB, error) {
	mux := &sync.RWMutex{}
	db := DB{path: path, mux: mux}
	db.ensureDB()

	return &db, nil
}

// ensureDB creates a new database file if it doesn't exist
func (db *DB) ensureDB() error {
	emptyData := DBStructure{}
	filecontent, _ := json.Marshal(emptyData)

	if _, err := os.Stat(db.path); errors.Is(err, os.ErrNotExist) {
		err := os.WriteFile(db.path, filecontent, 0666)
		if err != nil {
			log.Fatal("Reached writefile")
			panic(err)
		}
	} else if err != nil {
		log.Fatal("Did not reach writefile")
		panic(err)
	}
	return nil
}

// loadDB reads the database file into memory
func (db *DB) loadDB() (DBStructure, error) {
	err := db.ensureDB()
	if err != nil {
		panic("Unable to ensure db")
	}
	dbData := DBStructure{Chirps: make(map[int]Chirp)}
	fileData, err := os.ReadFile(db.path)
	if err != nil {
		log.Printf("Unable to read db: %v", err)
		return DBStructure{}, err
	}

	if len(fileData) > 0 {
		err = json.Unmarshal(fileData, &dbData)
		if err != nil {
			log.Println("Error reading db data: possibly malformed")
			return DBStructure{}, err
		}
	}
	return dbData, nil

}

// writeDB writes the database file to disk
func (db *DB) writeDB(dbData DBStructure) error {
	db.ensureDB()
	fdata, err := json.Marshal(dbData)
	if err != nil {
		log.Println("Error marshalling db data: possibly malformed")
		return err
	}
	err = os.WriteFile(db.path, fdata, 0666)
	if err != nil {
		log.Println("Error writing file")
		return err
	}
	return nil
}

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateChirp(body string) (Chirp, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	dbData, err := db.loadDB()
	log.Printf("DB DATA: %v", dbData)
	if err != nil {
		return Chirp{}, err
	}

	last_id := db.lastid
	chirp := Chirp{Body: body, Id: last_id + 1}
	if dbData.Chirps == nil {
		dbData.Chirps = make(map[int]Chirp)
	}
	dbData.Chirps[chirp.Id] = chirp
	err = db.writeDB(dbData)
	if err != nil {
		return Chirp{}, err
	} else {
		db.lastid += 1
		return chirp, nil
	}
}

func (db *DB) GetUserIdByEmail(email string) (User, bool) {
	dbData, _ := db.loadDB()
	for _, dbuser := range dbData.Users {
		if dbuser.Email == email {
			return dbuser, true
		}
	}
	return User{}, false
}

// CreateUser creates a new user and saves to disk
func (db *DB) CreateUser(user User) (User, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	dbData, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	// if user email exists, raise error - can't create duplicate
	_, ok := db.GetUserIdByEmail(user.Email)
	if ok {
		return User{}, errors.New("user already exists")
	}

	last_id := db.lastid
	user.Id = last_id + 1
	pwhash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	user.Password = string(pwhash)

	if dbData.Users == nil {
		dbData.Users = make(map[int]User)
	}
	dbData.Users[user.Id] = user
	err = db.writeDB(dbData)
	if err != nil {
		return User{}, err
	} else {
		db.lastid += 1
		user.Password = ""
		return user, nil
	}
}

// UpdateUser updates password for a user
func (db *DB) UpdateUser(user User) (User, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	dbData, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	_, ok := dbData.Users[user.Id]
	if !ok {
		return User{}, errors.New("user does not exist")
	}

	pwhash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	user.Password = string(pwhash)

	if dbData.Users == nil {
		dbData.Users = make(map[int]User)
	}
	dbData.Users[user.Id] = user
	err = db.writeDB(dbData)
	if err != nil {
		return User{}, err
	} else {
		user.Password = ""
		return user, nil
	}
}

func CheckPasswordsEqual(hashedPw, plaintextPw string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPw), []byte(plaintextPw))
	return err == nil
}

// GetChirps returns all chirps in the database
func (db *DB) GetChirps() ([]Chirp, error) {
	db.mux.RLock()
	defer db.mux.RUnlock()

	dbData, err := db.loadDB()
	if err != nil {
		return nil, err
	}

	var chirps []Chirp
	for _, chirp := range dbData.Chirps {
		chirps = append(chirps, chirp)
	}
	return chirps, nil
}

// GetChirpById returns a specific chirl in the database
func (db *DB) GetChirpById(id int) (Chirp, error) {
	db.mux.RLock()
	defer db.mux.RUnlock()

	dbData, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}

	chirp, ok := dbData.Chirps[id]
	if !ok {
		return Chirp{}, errors.New("Chirp not found")
	}
	return chirp, nil
}
