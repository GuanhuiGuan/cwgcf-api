package models

import (
	"context"
	"encoding/json"
	"fmt"
	"gguan/cwgcf_db/config"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AlbumServer is the definition of a REST API for photos
type AlbumServer struct {
	Client *mongo.Client
}

// NewAlbumServer creates a new Server instance
func NewAlbumServer() *AlbumServer {
	s := &AlbumServer{}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	// "mongodb+srv://<username>:<password>@<cluster-address>/test?w=majority"
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(
		config.MongoDBUrl,
	))
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Connected to MongoDB")

	s.Client = client
	return s
}

// GetAll handles getAll requests
func (s *AlbumServer) GetAll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	collection := s.Client.Database("cwgcf").Collection("album")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	cur, err := collection.Find(ctx, bson.D{})
	if err != nil {
		log.Printf("Error getting album: %v", err)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Not found"}`))
		return
	}
	defer cur.Close(ctx)
	res := []Photo{}
	for cur.Next(ctx) {
		var photo Photo

		err := cur.Decode(&photo)
		if err != nil {
			log.Print(err)
			continue
		}
		res = append(res, photo)
	}

	resBytes, _ := json.Marshal(res)

	w.WriteHeader(http.StatusOK)
	w.Write(resBytes)
}

// Put handles put requests
func (s *AlbumServer) Put(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var photo Photo

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&photo)
	if err != nil {
		log.Printf("Failed to decode body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Check your request"}`))
		return
	}

	collection := s.Client.Database("cwgcf").Collection("album")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	doc := bson.M{
		"url": photo.URL,
	}

	dbRes, err := collection.InsertOne(ctx, doc)
	if err != nil {
		log.Printf("Failed to decode body: %v", err)
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error": "Photo exists"}`))
		return
	}
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(fmt.Sprintf(`{"insertID": %v}`, dbRes.InsertedID)))
}
