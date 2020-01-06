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
	Client        *mongo.Client
	ProfileClient *ProfileServer
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

	s.ProfileClient = NewProfileServer()

	s.Client = client
	return s
}

// GetAllPosts handles getAll requests
func (s *ForumServer) GetAllPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	collection := s.Client.Database("cwgcf").Collection("forumPosts")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	opt := options.Find()
	opt.SetSort(bson.D{{"forumVotes.votesSum", -1}, {"updatedAt", -1}})
	cur, err := collection.Find(ctx, bson.D{}, opt)
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
			log.Printf("Error getting post: %v", err)
			continue
		}
		// Get user profile
		profile, err := s.ProfileClient.GetProfile(post.UserID)
		if err != nil {
			log.Printf("Error getting profile for ID %s: %v", post.UserID, err)
			continue
		}
		post.UserProfile = profile

		res = append(res, post)
	}

	resBytes, _ := json.Marshal(res)

	w.WriteHeader(http.StatusOK)
	w.Write(resBytes)
}

// GetPost handles get one post requests
func (s *ForumServer) GetPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathParams := mux.Vars(r)

	if postID, ok := pathParams["postID"]; ok {
		var forumPost ForumPost
		collection := s.Client.Database("cwgcf").Collection("forumPosts")
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		objectID, _ := primitive.ObjectIDFromHex(postID)
		filter := bson.M{"_id": objectID}
		err := collection.FindOne(ctx, filter).Decode(&forumPost)
		if err != nil {
			log.Printf("Error getting post with id %s from DB: %v", postID, err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Get user profile
		profile, err := s.ProfileClient.GetProfile(forumPost.UserID)
		if err != nil {
			log.Printf("Error getting profile for ID %s: %v", forumPost.UserID, err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		forumPost.UserProfile = profile
		res, _ := json.Marshal(forumPost)
		w.WriteHeader(http.StatusOK)
		w.Write(res)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

// GetCommentsForPostV2 gets all comment for given post id
func (s *ForumServer) GetCommentsForPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathParams := mux.Vars(r)

	if postID, ok := pathParams["postID"]; ok {
		comments := s.queryCommentByParent(postID)
		if comments == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		res, _ := json.Marshal(&comments)
		w.WriteHeader(http.StatusOK)
		w.Write(res)
		return
	}
	w.WriteHeader(http.StatusBadRequest)
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
		"createdAt":  forumPost.CreatedAt,
		"updatedAt":  forumPost.CreatedAt,
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

// AddCommentV2 handles request to add a comment and updates all parents' updatedAt
func (s *ForumServer) AddCommentV2(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathParams := mux.Vars(r)

	if parentID, ok := pathParams["parentID"]; ok {
		// decode request body
		var forumComment ForumComment
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&forumComment)
		if err != nil {
			log.Printf("Failed to decode body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Check your request"}`))
			return
		}
		// Insert comment
		collection := s.Client.Database("cwgcf").Collection("forumComments")
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		doc := bson.M{
			"parentId":   parentID,
			"content":    forumComment.Content,
			"createdAt":  forumComment.CreatedAt,
			"updatedAt":  forumComment.CreatedAt,
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

		// Update parents' updatedAt
		go func() {
			err = s.updateCommentUpdatedAt(parentID, forumComment.CreatedAt)
			if err != nil {
				log.Printf("Failed to update updatedAt: %v", err)
			}
		}()

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(fmt.Sprintf(`{"insertID": %v}`, commentID)))
		return
	}
	w.WriteHeader(http.StatusBadRequest)
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
			"createdAt":  forumComment.CreatedAt,
			"updatedAt":  forumComment.CreatedAt,
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

func (s *ForumServer) setUpdateKeyValue(unvote bool, upvote bool) (string, bool) {
	var prefix string
	var value bool
	if unvote {
		value = false
		if upvote {
			prefix = "forumVotes.upvotedIds."
		} else {
			prefix = "forumVotes.downvotedIds."
		}
	} else {
		value = true
		if upvote {
			prefix = "forumVotes.upvotedIds."
		} else {
			prefix = "forumVotes.downvotedIds."
		}
	}
	return prefix, value
}

// votePost is the internal core for VotePost and UnvotePost
func (s *ForumServer) _votePost(w http.ResponseWriter, r *http.Request, update map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	pathParams := mux.Vars(r)
	if id, ok := pathParams["id"]; ok {
		if score, ok := pathParams["score"]; ok {
			upvoted := false
			if score == "1" {
				upvoted = true
			}
			collection := s.Client.Database("cwgcf").Collection("forumPosts")
			ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
			objectID, _ := primitive.ObjectIDFromHex(id)
			filter := bson.M{"_id": objectID}
			var update map[string]interface{}
			if upvoted {
				update = bson.M{"$inc": bson.M{"forumVotes.upvotes": 1, "forumVotes.votesSum": 1}}
			} else {
				update = bson.M{"$inc": bson.M{"forumVotes.downvotes": 1, "forumVotes.votesSum": -1}}
			}
			var forumPost ForumPost
			err := collection.FindOneAndUpdate(ctx, filter, update).Decode(&forumPost)
			// _, err := collection.UpdateOne(ctx, filter, update)
			if err != nil {
				log.Printf("Failed to update voting: %v", err)
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"error": "Failed to update voting"}`))
				return
			}
			if upvoted {
				forumPost.ForumVotes.Upvotes++
				forumPost.ForumVotes.VotesSum++
			} else {
				forumPost.ForumVotes.Downvotes++
				forumPost.ForumVotes.VotesSum--
			}
			res, _ := json.Marshal(forumPost)
			w.WriteHeader(http.StatusOK)
			w.Write(res)
			return
		}
	}
	w.WriteHeader(http.StatusBadRequest)
}

// VoteComment handles request to vote comment
func (s *ForumServer) _VoteComment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathParams := mux.Vars(r)
	if id, ok := pathParams["id"]; ok {
		if score, ok := pathParams["score"]; ok {
			upvoted := false
			if score == "1" {
				upvoted = true
			}
			collection := s.Client.Database("cwgcf").Collection("forumComments")
			ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
			objectID, _ := primitive.ObjectIDFromHex(id)
			filter := bson.M{"_id": objectID}
			var update map[string]interface{}
			if upvoted {
				update = bson.M{"$inc": bson.M{"forumVotes.upvotes": 1, "forumVotes.votesSum": 1}}
			} else {
				update = bson.M{"$inc": bson.M{"forumVotes.downvotes": 1, "forumVotes.votesSum": -1}}
			}
			var forumComment ForumComment
			err := collection.FindOneAndUpdate(ctx, filter, update).Decode(&forumComment)
			if err != nil {
				log.Printf("Failed to update voting: %v", err)
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"error": "Failed to update voting"}`))
				return
			}
			if upvoted {
				forumComment.ForumVotes.Upvotes++
				forumComment.ForumVotes.VotesSum++
			} else {
				forumComment.ForumVotes.Downvotes++
				forumComment.ForumVotes.VotesSum--
			}
			res, _ := json.Marshal(forumComment)
			w.WriteHeader(http.StatusOK)
			w.Write(res)
			return
		}
	}
	w.WriteHeader(http.StatusBadRequest)
}

// queryCommentByParent queries comments recursively
func (s *ForumServer) queryCommentByParent(parentID string) []ForumComment {
	collection := s.Client.Database("cwgcf").Collection("forumComments")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	filter := bson.M{"parentId": parentID}
	opt := options.Find()
	opt.SetSort(bson.D{{"forumVotes.votesSum", -1}, {"updatedAt", -1}})
	cur, err := collection.Find(ctx, filter, opt)
	if err != nil {
		log.Printf("Error getting comments with parentID %s: %v", parentID, err)
		return nil
	}
	defer cur.Close(ctx)
	res := []ForumComment{}
	for cur.Next(ctx) {
		var comment ForumComment

		err := cur.Decode(&comment)
		if err != nil {
			log.Printf("Error getting comment: %v", err)
			continue
		}
		// Get user profile
		profile, err := s.ProfileClient.GetProfile(comment.UserID)
		if err != nil {
			log.Printf("Error getting profile for ID %s: %v", comment.UserID, err)
			continue
		}
		comment.UserProfile = profile

		// Find children comments
		subComments := s.queryCommentByParent(comment.ID)
		if subComments != nil {
			comment.Comments = subComments
		}

		res = append(res, comment)
	}
	return res
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

func (s *ForumServer) updatePostUpdatedAt(id string, updatedAt int64) (err error) {
	collection := s.Client.Database("cwgcf").Collection("forumPosts")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	objectID, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": bson.M{"updatedAt": updatedAt}}
	_, err = collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("Failed to update updatedAt for post id %s: %v", id, err)
	}
	return err
}

func (s *ForumServer) updateCommentUpdatedAt(id string, updatedAt int64) (err error) {
	// Find parent
	collection := s.Client.Database("cwgcf").Collection("forumComments")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	objectID, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"_id": objectID}
	var comment ForumComment
	err = collection.FindOne(ctx, filter).Decode(&comment)
	// Try post if not found
	if err != nil {
		log.Printf("Failed to get comment with ID %s: %v", id, err)
		return s.updatePostUpdatedAt(id, updatedAt)
	}
	// Update parents recursively
	if len(comment.ParentID) > 0 {
		s.updateCommentUpdatedAt(comment.ParentID, updatedAt)
	}

	// Update self
	update := bson.M{"$set": bson.M{"updatedAt": updatedAt}}
	_, err = collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("Failed to update updatedAt for comment id %s: %v", id, err)
	}
	return err
}

// GetUserVoteMap gets voteMap of a user
func (s *ForumServer) GetUserVoteMap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	header := http.StatusInternalServerError
	var res []byte
	var err error
	defer func() {
		if err != nil {
			log.Printf("Error getting voteMap: %v", err)
		}
		w.WriteHeader(header)
		w.Write(res)
	}()
	pathParams := mux.Vars(r)
	if id, ok := pathParams["id"]; ok {
		collection := s.Client.Database("cwgcf").Collection("forumUserVotes")
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		filter := bson.M{"userId": id}
		var forumUserVotes ForumUserVotes
		err := collection.FindOne(ctx, filter).Decode(&forumUserVotes)
		if err != nil {
			header = http.StatusInternalServerError
			return
		}
		res, _ = json.Marshal(forumUserVotes)
		header = http.StatusOK
	}
}

// Vote handles vote requests
func (s *ForumServer) Vote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	header := http.StatusInternalServerError
	var res []byte
	var err error
	defer func() {
		if err != nil {
			log.Printf("Error getting voteMap: %v", err)
		}
		w.WriteHeader(header)
		w.Write(res)
	}()

	// Parse request
	var request VoteRequest
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&request)
	if err != nil {
		header = http.StatusBadRequest
		return
	}
	// Update vote
	if request.Offset > 2 || request.Offset < -2 {
		request.Offset = 0
	}
	collectionName := "forumComments"
	if request.IsPost {
		collectionName = "forumPosts"
	}
	err = s.vote(request, request.Offset, collectionName)
	if err != nil {
		header = http.StatusInternalServerError
		return
	}
	// Update voteMap
	err = s.postUserVoteMap(request)
	if err != nil {
		header = http.StatusInternalServerError
		return
	}
	header = http.StatusOK
}

// vote changes the votesSum value
// offset: do upvote OR undo downvote = 1; do downvote OR undo upvote = -1;
func (s *ForumServer) vote(request VoteRequest, offset int, collectionName string) error {
	collection := s.Client.Database("cwgcf").Collection(collectionName)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	objectID, _ := primitive.ObjectIDFromHex(request.VoteID)
	filter := bson.M{"_id": objectID}
	update := bson.M{"$inc": bson.M{"forumVotes.votesSum": offset}}
	opt := options.Update()
	opt.SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opt)
	if err != nil {
		return err
	}
	return nil
}

// postUserVoteMap updates a user's voteMap
func (s *ForumServer) postUserVoteMap(request VoteRequest) error {
	var value int
	if request.Offset > 0 {
		value = 1
	} else if request.Offset < 0 {
		value = -1
	}
	collection := s.Client.Database("cwgcf").Collection("forumUserVotes")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	filter := bson.M{"userId": request.UserID}
	update := bson.M{"$set": bson.M{"voteMap." + request.VoteID: value}}
	opt := options.Update()
	opt.SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opt)
	if err != nil {
		return err
	}
	return nil
}
