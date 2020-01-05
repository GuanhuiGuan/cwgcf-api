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
	mongoAPI.HandleFunc("/profile", ps.GetAll).Methods(http.MethodGet)
	mongoAPI.HandleFunc("/profile/{userID}", ps.Get).Methods(http.MethodGet)
	mongoAPI.HandleFunc("/profile/{userID}", ps.Post).Methods(http.MethodPost)
	mongoAPI.HandleFunc("/profile", ps.Put).Methods(http.MethodPut)
	mongoAPI.HandleFunc("/profile/{userID}", ps.Delete).Methods(http.MethodDelete)

	albumServer := models.NewAlbumServer()
	mongoAPI.HandleFunc("/album", albumServer.GetAll).Methods(http.MethodGet)
	mongoAPI.HandleFunc("/album", albumServer.Put).Methods(http.MethodPut)

	forumServer := models.NewForumServer()
	mongoAPI.HandleFunc("/forum/post", forumServer.GetAllPosts).Methods(http.MethodGet)
	mongoAPI.HandleFunc("/forum/post/{postID}", forumServer.GetPost).Methods(http.MethodGet)
	mongoAPI.HandleFunc("/forum/commentsofpost/{postID}", forumServer.GetCommentsForPostV2).Methods(http.MethodGet)
	mongoAPI.HandleFunc("/forum/post", forumServer.PutPost).Methods(http.MethodPut)
	mongoAPI.HandleFunc("/forum/comment/{parentID}", forumServer.AddCommentV2).Methods(http.MethodPost)
	mongoAPI.HandleFunc("/forum/post/{id}/{score}", forumServer.VotePost).Methods(http.MethodPost)
	mongoAPI.HandleFunc("/forum/comment/{id}/{score}", forumServer.VoteComment).Methods(http.MethodPost)

	log.Fatal(http.ListenAndServe(":8080", router))
}
