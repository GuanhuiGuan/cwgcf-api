package main

import (
	"gguan/cwgcf_db/models"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()

	mongoAPI := router.PathPrefix("/mongo/v1").Subrouter()

	ps := models.NewProfileServer()
	mongoAPI.HandleFunc("/profiles", ps.GetAll).Methods(http.MethodGet)
	mongoAPI.HandleFunc("/profiles/{userID}", ps.Get).Methods(http.MethodGet)
	mongoAPI.HandleFunc("/profiles/{userID}", ps.Post).Methods(http.MethodPost)
	mongoAPI.HandleFunc("/profiles", ps.Put).Methods(http.MethodPut)
	mongoAPI.HandleFunc("/profiles/{userID}", ps.Delete).Methods(http.MethodDelete)

	albumServer := models.NewAlbumServer()
	mongoAPI.HandleFunc("/album", albumServer.GetAll).Methods(http.MethodGet)
	mongoAPI.HandleFunc("/album", albumServer.Put).Methods(http.MethodPut)

	log.Fatal(http.ListenAndServe(":8080", router))
}
