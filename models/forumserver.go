package models

import (
	"context"
	"encoding/json"
	"fmt"
	"gguan/cwgcf_db/config"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ForumServer is the definition of a REST API for forum
type ForumServer struct {
	Client *mongo.Client
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

	s.Client = client
	return s
}

// GetAllPosts handles getAll requests
func (s *ForumServer) GetAllPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	collection := s.Client.Database("cwgcf").Collection("forumPosts")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	cur, err := collection.Find(ctx, bson.D{})
	if err != nil {
		log.Printf("Error getting forum posts: %v", err)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Not found"}`))
		return
	}
	defer cur.Close(ctx)
	res := []ForumPost{}
	for cur.Next(ctx) {
		var post ForumPost

		err := cur.Decode(&post)
		if err != nil {
			log.Print(err)
			continue
		}
		res = append(res, post)
	}

	resBytes, _ := json.Marshal(res)

	w.WriteHeader(http.StatusOK)
	w.Write(resBytes)
}

// GetCommentsForPost gets all comment for given post id
func (s *ForumServer) GetCommentsForPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathParams := mux.Vars(r)

	if postID, ok := pathParams["postID"]; ok {
		// Get sub comment IDs
		var forumSubComments ForumSubComments
		collection := s.Client.Database("cwgcf").Collection("forumSubComments")
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		filter := bson.M{"parentId": postID}
		err := collection.FindOne(ctx, filter).Decode(&forumSubComments)
		if err != nil {
			log.Printf("Error getting forum comments of parentID %s from DB: %v", postID, err)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		comments := []ForumComment{}
		for _, subID := range forumSubComments.CommentIDs {
			c := s.queryComment(subID)
			if c != nil {
				comments = append(comments, *c)
			}
		}

		res, _ := json.Marshal(comments)

		w.WriteHeader(http.StatusOK)
		w.Write(res)
	}
}

// PutPost handles forumPost put requests
func (s *ForumServer) PutPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var forumPost ForumPost

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&forumPost)
	if err != nil {
		log.Printf("Failed to decode body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Check your request"}`))
		return
	}

	collection := s.Client.Database("cwgcf").Collection("forumPosts")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	doc := bson.M{
		"title":      forumPost.Title,
		"content":    forumPost.Content,
		"image":      forumPost.Image,
		"timestamp":  forumPost.Timestamp,
		"userId":     forumPost.UserID,
		"forumVotes": forumPost.ForumVotes,
	}

	dbRes, err := collection.InsertOne(ctx, doc)
	if err != nil {
		log.Printf("Failed to insert post: %v", err)
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error": "Internal error"}`))
		return
	}

	objectID, _ := dbRes.InsertedID.(primitive.ObjectID)

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(fmt.Sprintf(`{"insertID": %v}`, objectID.Hex())))
}

// AddComment handles request to add a comment
func (s *ForumServer) AddComment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathParams := mux.Vars(r)

	if parentID, ok := pathParams["parentID"]; ok {
		// ensure the object exists
		s.insertSubCommentIdsArray(parentID)

		var forumComment ForumComment

		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&forumComment)
		if err != nil {
			log.Printf("Failed to decode body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Check your request"}`))
			return
		}

		collection := s.Client.Database("cwgcf").Collection("forumComments")
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		doc := bson.M{
			"content":    forumComment.Content,
			"timestamp":  forumComment.Timestamp,
			"userId":     forumComment.UserID,
			"forumVotes": forumComment.ForumVotes,
		}

		dbRes, err := collection.InsertOne(ctx, doc)
		if err != nil {
			log.Printf("Failed to insert comment: %v", err)
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(`{"error": "Failed to insert comment"}`))
			return
		}
		objectID, _ := dbRes.InsertedID.(primitive.ObjectID)
		commentID := objectID.Hex()

		collection = s.Client.Database("cwgcf").Collection("forumSubComments")
		filter := bson.M{"parentId": parentID}
		update := bson.M{"$push": bson.M{"commentIds": commentID}}
		_, err = collection.UpdateOne(ctx, filter, update)
		if err != nil {
			log.Printf("Failed to link comment and parent: %v", err)
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(`{"error": "Failed to link comment and parent"}`))
			return
		}

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(fmt.Sprintf(`{"insertID": %v}`, commentID)))
	}
}

// VotePost handles request to vote post
func (s *ForumServer) VotePost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathParams := mux.Vars(r)
	if id, ok := pathParams["id"]; ok {
		if score, ok := pathParams["score"]; ok {
			collection := s.Client.Database("cwgcf").Collection("forumPosts")
			ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
			objectID, _ := primitive.ObjectIDFromHex(id)
			filter := bson.M{"_id": objectID}
			var update map[string]interface{}
			if score == "1" {
				update = bson.M{"$inc": bson.M{"forumVotes.upvotes": 1}}
			} else {
				update = bson.M{"$inc": bson.M{"forumVotes.downvotes": 1}}
			}
			_, err := collection.UpdateOne(ctx, filter, update)
			if err != nil {
				log.Printf("Failed to update voting: %v", err)
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"error": "Failed to update voting"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
		}
	}
}

// VoteComment handles request to vote comment
func (s *ForumServer) VoteComment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathParams := mux.Vars(r)
	if id, ok := pathParams["id"]; ok {
		if score, ok := pathParams["score"]; ok {
			collection := s.Client.Database("cwgcf").Collection("forumComments")
			ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
			objectID, _ := primitive.ObjectIDFromHex(id)
			filter := bson.M{"_id": objectID}
			var update map[string]interface{}
			if score == "1" {
				update = bson.M{"$inc": bson.M{"forumVotes.upvotes": 1}}
			} else {
				update = bson.M{"$inc": bson.M{"forumVotes.downvotes": 1}}
			}
			_, err := collection.UpdateOne(ctx, filter, update)
			if err != nil {
				log.Printf("Failed to update voting: %v", err)
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"error": "Failed to update voting"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
		}
	}
}

// queryComment queries comments recursively
func (s *ForumServer) queryComment(id string) *ForumComment {
	// Get sub comment IDs
	var forumSubComments ForumSubComments
	collection := s.Client.Database("cwgcf").Collection("forumSubComments")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	filter := bson.M{"parentId": id}
	err := collection.FindOne(ctx, filter).Decode(&forumSubComments)
	if err != nil {
		log.Printf("Error getting forum comments of parentId %s from DB: %v", id, err)
	}

	// Get content for each sub comment
	subComments := []ForumComment{}
	if len(forumSubComments.CommentIDs) > 0 {
		for _, subID := range forumSubComments.CommentIDs {
			c := s.queryComment(subID)
			if c != nil {
				subComments = append(subComments, *c)
			}
		}
	}

	// Get content of current id
	var forumComment ForumComment
	collection = s.Client.Database("cwgcf").Collection("forumComments")
	objectID, _ := primitive.ObjectIDFromHex(id)
	filter = bson.M{"_id": objectID}
	err = collection.FindOne(ctx, filter).Decode(&forumComment)
	if err != nil {
		log.Printf("Error getting comment from DB: %v", err)
		return nil
	}
	forumComment.Comments = subComments
	return &forumComment
}

func (s *ForumServer) insertSubCommentIdsArray(id string) {
	// Insert empty comment array
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	collection := s.Client.Database("cwgcf").Collection("forumSubComments")
	doc := bson.M{
		"parentId": id,
	}

	_, err := collection.InsertOne(ctx, doc)
	if err != nil {
		log.Printf("Failed to create subcomment array for id %s: %v", id, err)
		return
	}
}