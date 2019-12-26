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
	s := models.NewProfileServer()
	mongoAPI.HandleFunc("/profiles", s.GetAll).Methods(http.MethodGet)
	mongoAPI.HandleFunc("/profiles/{userID}", s.Get).Methods(http.MethodGet)
	mongoAPI.HandleFunc("/profiles/{userID}", s.Post).Methods(http.MethodPost)
	mongoAPI.HandleFunc("/profiles", s.Put).Methods(http.MethodPut)
	mongoAPI.HandleFunc("/profiles/{userID}", s.Delete).Methods(http.MethodDelete)

	log.Fatal(http.ListenAndServe(":8080", router))
}
