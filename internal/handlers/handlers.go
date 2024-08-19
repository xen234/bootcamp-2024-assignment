package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/xen234/bootcamp-2024-assignment/api"
	"github.com/xen234/bootcamp-2024-assignment/internal/storage/sqlite"
)

type MyServer struct {
	Storage *sqlite.Storage
}

// ожидается формат Bearer TYPE_OF_USER-token
func (s *MyServer) isModerator(r *http.Request) bool {
	token := r.Header.Get("Authorization")
	log.Println("isModerator raw token:", token)

	const bearerPrefix = "Bearer "
	if len(token) > len(bearerPrefix) && token[:len(bearerPrefix)] == bearerPrefix {
		token = token[len(bearerPrefix):]
	}

	log.Println("isModerator token without Bearer:", token)
	return token == "moderator-token"
}

func (s *MyServer) GetDummyLogin(w http.ResponseWriter, r *http.Request, params api.GetDummyLoginParams) {
	var token string

	switch params.UserType {
	case api.Client:
		token = "client"
	case api.Moderator:
		token = "moderator"
	default:
		http.Error(w, "Invalid user type", http.StatusBadRequest)
		return
	}

	response := map[string]string{
		"token": token,
	}

	log.Printf("user authorized as %s", token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *MyServer) PostFlatCreate(w http.ResponseWriter, r *http.Request) {
	var flat api.Flat

	if err := json.NewDecoder(r.Body).Decode(&flat); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	createdFlat, err := s.Storage.CreateFlat(flat)
	if err != nil {
		http.Error(w, "Failed to create flat", http.StatusInternalServerError)
		log.Println("Failed to create flat:", err)
		return
	}

	err = s.Storage.UpdateHouseTimestamp(flat.HouseId)
	if err != nil {
		http.Error(w, "Failed to update house timestamp", http.StatusInternalServerError)
		log.Println("Failed to update house timestamp:", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdFlat)
}

func (s *MyServer) PostFlatUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.isModerator(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var flat api.Flat

	if err := json.NewDecoder(r.Body).Decode(&flat); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		log.Println("Invalid request payload:", err)
		return
	}

	updatedFlat, err := s.Storage.UpdateFlat(flat)
	if err != nil {
		http.Error(w, "Failed to update flat", http.StatusInternalServerError)
		log.Println("Failed to update flat:", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedFlat)
}

func (s *MyServer) PostHouseCreate(w http.ResponseWriter, r *http.Request) {
	log.Println("PostHouseCreate called")

	if s.Storage == nil {
		log.Println("Storage is nil")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !s.isModerator(r) {
		log.Println("User is not a moderator")
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var house api.House
	if err := json.NewDecoder(r.Body).Decode(&house); err != nil {
		log.Println("Failed to decode house:", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	createdHouse, err := s.Storage.CreateHouse(house)
	if err != nil {
		log.Println("Failed to create house:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdHouse)
}

func (s *MyServer) GetHouseId(w http.ResponseWriter, r *http.Request, id api.HouseId) {
	var flats []api.Flat
	var err error

	if s.isModerator(r) {
		log.Println("GetHouseId: moderator")
		flats, err = s.Storage.GetAllFlatsByHouseId(id)
	} else {
		log.Println("GetHouseId: client")
		flats, err = s.Storage.GetApprovedFlatsByHouseId(id)
	}

	if err != nil {
		log.Println("Failed to get flats:", err)
		http.Error(w, "Failed to get flats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(flats)
}

func (s *MyServer) PostHouseIdSubscribe(w http.ResponseWriter, r *http.Request, HouseId int) {
	// unimplemented
}

func (s *MyServer) PostLogin(w http.ResponseWriter, r *http.Request) {
	// unimplemented
}

func (s *MyServer) PostRegister(w http.ResponseWriter, r *http.Request) {
	// unimplemented
}
