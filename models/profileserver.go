package models

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"gguan/cwgcf_db/config"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ProfileServer is the definition of a REST API for user profiles
type ProfileServer struct {
	Profiles map[string]Profile
	Client   *mongo.Client
}

// NewProfileServer creates a new Server instance
func NewProfileServer() *ProfileServer {
	s := &ProfileServer{}
	// jsonFile, err := os.Open("fixtures/mock_profiles.json")
	// if err != nil {
	// 	log.Printf("Error opening file: %v", err)
	// 	return s
	// }
	// defer jsonFile.Close()

	// bytes, err := ioutil.ReadAll(jsonFile)
	// if err != nil {
	// 	log.Printf("Error reading file: %v", err)
	// 	return s
	// }

	// var profiles map[string]Profile
	// err = json.Unmarshal(bytes, &profiles)
	// if err != nil {
	// 	log.Printf("Error unmarshalling file: %v", err)
	// 	return s
	// }

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

// Get handles get requests
func (s *ProfileServer) Get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathParams := mux.Vars(r)

	if userID, ok := pathParams["userID"]; ok {
		header := http.StatusOK
		res := []byte{}

		profile, err := s.GetProfile(userID)
		if err != nil {
			header = http.StatusNotFound
			log.Printf("Error getting profile from DB: %v", err)
		} else {
			res, _ = json.Marshal(profile)
		}

		w.WriteHeader(header)
		w.Write(res)
	}
}

// GetProfile finds a profile with given userID
func (s *ProfileServer) GetProfile(userID string) (profile Profile, err error) {
	collection := s.Client.Database("cwgcf").Collection("profiles")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	objectID, _ := primitive.ObjectIDFromHex(userID)
	filter := bson.M{"_id": objectID}
	err = collection.FindOne(ctx, filter).Decode(&profile)
	return profile, err
}

// GetAll handles getAll requests
func (s *ProfileServer) GetAll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	collection := s.Client.Database("cwgcf").Collection("profiles")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	cur, err := collection.Find(ctx, bson.D{})
	if err != nil {
		log.Printf("Error getting profiles: %v", err)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Not found"}`))
		return
	}
	defer cur.Close(ctx)
	res := []Profile{}
	for cur.Next(ctx) {
		var profile Profile

		err := cur.Decode(&profile)
		if err != nil {
			log.Print(err)
			continue
		}
		res = append(res, profile)
	}

	resBytes, _ := json.Marshal(res)

	w.WriteHeader(http.StatusOK)
	w.Write(resBytes)
}

// NOT IMPLEMENTED
// Post handles post requests
func (s *ProfileServer) Post(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message": "post called"}`))
}

// Put handles put requests
func (s *ProfileServer) Put(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var profile Profile

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&profile)
	if err != nil {
		log.Printf("Failed to decode body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Check your request"}`))
		return
	}

	collection := s.Client.Database("cwgcf").Collection("profiles")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	doc := bson.M{
		"name":        profile.Name,
		"title":       profile.Title,
		"description": profile.Description,
		"avatarUrl":   profile.AvatarURL,
	}

	dbRes, err := collection.InsertOne(ctx, doc)
	if err != nil {
		log.Printf("Failed to decode body: %v", err)
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error": "User exists"}`))
		return
	}
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(fmt.Sprintf(`{"insertID": %v}`, dbRes.InsertedID)))
}

// NOT IMPLEMENTED
// Delete handles delete requests
func (s *ProfileServer) Delete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "delete called"}`))
}

// NOT IMPLEMENTED
// NotFound handles notFound requests
func (s *ProfileServer) NotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"message": "not found"}`))
}
