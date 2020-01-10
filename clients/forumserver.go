package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"gguan/cwgcf_db/config"
	"gguan/cwgcf_db/models"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ForumServer is the definition of a REST API for forum
type ForumServer struct {
	Client        *mongo.Client
	ProfileClient *models.ProfileServer
}

// NewForumServer creates a new Server instance
func NewForumServer() *ForumServer {
	s := &ForumServer{}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	// "mongodb+srv://<username>:<password>@<cluster-address>/test?w=majority"
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(
		config.MongoDBUrl,
	))
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Connected to MongoDB")
	s.ProfileClient = models.NewProfileServer()
	s.Client = client
	return s
}

/*
	Endpoints
*/

// GetForumPosts returns an array of forum posts
func (s *ForumServer) GetForumPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Parse request
	var getForumPostsRequest models.GetForumPostsRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&getForumPostsRequest)
	if err != nil {
		log.Printf("Failed to decode request: %v", err)
	}
	// Fetch DBPosts
	collection := s.Client.Database("cwgcf").Collection("forumPosts")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	opt := options.Find()
	opt.SetSort(bson.D{{"metadata.updatedAt", -1}})
	cur, err := collection.Find(ctx, bson.D{}, opt)
	if err != nil {
		log.Printf("Error getting forum posts: %v", err)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf(`{"error": "%v"}`, err)))
		return
	}
	// Transform DBPosts to []ForumPostV2
	defer cur.Close(ctx)
	response := models.GetForumPostsResponse{}
	posts := []models.ForumPostV2{}
	votes := map[string]models.ForumVote{}
	for cur.Next(ctx) {
		var dbPost models.DBForumPost
		err := cur.Decode(&dbPost)
		if err != nil {
			log.Printf("Error getting post: %v", err)
			continue
		}
		// Get user profile
		profile, err := s.ProfileClient.GetProfile(dbPost.UserID)
		if err != nil {
			log.Printf("Error getting profile for ID %s: %v", dbPost.UserID, err)
			continue
		}
		post := s.dbPostToPost(dbPost, profile)
		posts = append(posts, post)

		// Get vote
		vote := s.getVote(dbPost.VoteID)
		votes[dbPost.VoteID] = vote
	}
	response.ForumPosts = posts
	response.ForumVotesMap = votes
	resBytes, _ := json.Marshal(response)

	w.WriteHeader(http.StatusOK)
	w.Write(resBytes)
}

// SaveForumPost saves a post in DB
func (s *ForumServer) SaveForumPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var err error
	defer func() {
		res := s.generateBasicResponse(err)
		resBytes, _ := json.Marshal(res)
		w.Write(resBytes)
	}()
	// Parse request
	var saveForumPostsRequest models.SaveForumPostsRequest
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&saveForumPostsRequest)
	if err != nil {
		log.Printf("Failed to decode request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	forumPost := saveForumPostsRequest.ForumPost
	// Create and get voteID
	voteID := s.createAndGetVoteID(forumPost.Metadata)
	log.Printf("VoteID: %s", voteID)
	// Upsert Post
	collection := s.Client.Database("cwgcf").Collection("forumPosts")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	doc := bson.M{
		"title":    forumPost.Title,
		"content":  forumPost.Content,
		"image":    forumPost.Image,
		"metadata": forumPost.Metadata,
		"userId":   forumPost.UserID,
		"voteId":   voteID,
	}
	dbRes, err := collection.InsertOne(ctx, doc)
	if err != nil {
		log.Printf("Error inserting forum post: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	objectID, _ := dbRes.InsertedID.(primitive.ObjectID)
	log.Printf("Insert ID: %s", objectID.Hex())
	w.WriteHeader(http.StatusOK)
}

// HandleVoteEvent updates vote object and userVoteMap
func (s *ForumServer) HandleVoteEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var err error
	defer func() {
		res := s.generateBasicResponse(err)
		resBytes, _ := json.Marshal(res)
		w.Write(resBytes)
	}()
	// Parse request
	var forumVoteUpdateRequest models.ForumVoteUpdateRequest
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&forumVoteUpdateRequest)
	if err != nil {
		log.Printf("Failed to decode request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Update vote
	s.updateVote(forumVoteUpdateRequest)
	// Update userVoteMap

	w.WriteHeader(http.StatusOK)
}

/*
	Helpers
*/

func (s *ForumServer) dbPostToPost(dbPost models.DBForumPost, profile models.Profile) models.ForumPostV2 {
	post := models.ForumPostV2{
		ID:          dbPost.ID,
		Title:       dbPost.Title,
		Content:     dbPost.Content,
		Image:       dbPost.Image,
		UserProfile: profile,
		VoteID:      dbPost.VoteID,
		Metadata:    dbPost.Metadata,
	}
	return post
}

func (s *ForumServer) generateBasicResponse(err error) models.BasicResponse {
	if err != nil {
		return models.BasicResponse{
			Success:  false,
			ErrorMsg: err.Error(),
		}
	} else {
		return models.BasicResponse{
			Success: true,
		}
	}
}

// createAndGetVoteID creates a new vote object and returns the id
func (s *ForumServer) createAndGetVoteID(metadata models.Metadata) string {
	collection := s.Client.Database("cwgcf").Collection("forumVotes")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	doc := bson.M{
		"count":    0,
		"metadata": metadata,
	}
	dbRes, err := collection.InsertOne(ctx, doc)
	if err != nil {
		log.Printf("Error inserting forum post: %v", err)
		return ""
	}
	objectID, _ := dbRes.InsertedID.(primitive.ObjectID)
	return objectID.Hex()
}

// getVote retrieves a vote object
func (s *ForumServer) getVote(id string) models.ForumVote {
	collection := s.Client.Database("cwgcf").Collection("forumVotes")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	objectID, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{
		"_id": objectID,
	}
	var vote models.ForumVote
	err := collection.FindOne(ctx, filter).Decode(&vote)
	if err != nil {
		log.Printf("Error getting vote with id %s: %v", id, err)
		return vote
	}
	return vote
}

// updateVote updates a vote object
func (s *ForumServer) updateVote(request models.ForumVoteUpdateRequest) {
	collection := s.Client.Database("cwgcf").Collection("forumVotes")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	objectID, _ := primitive.ObjectIDFromHex(request.VoteID)
	filter := bson.M{
		"_id": objectID,
	}
	update := bson.M{
		"$inc": bson.M{
			"count": request.Offset,
		},
		"$set": bson.M{
			"metadata.updatedAt": request.Metadata.UpdatedAt,
		},
	}
	opt := options.Update()
	opt.SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opt)
	if err != nil {
		log.Printf("Error updating vote with id %s: %v", request.VoteID, err)
	}
}
