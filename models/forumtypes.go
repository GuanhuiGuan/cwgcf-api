package models

// GetForumPostsRequest is the request definition for mobile to get forum posts
type GetForumPostsRequest struct {
	Limit int64 `bson:"limit" json:"limit"`
}

// GetForumPostsResponse is the response definition for mobile to get forum posts
type GetForumPostsResponse struct {
	ForumPosts    []ForumPostV2
	ForumVotesMap map[string]ForumVote
}

// SaveForumPostsRequest is the request definition for mobile to get forum posts
type SaveForumPostsRequest struct {
	ForumPost DBForumPost
}

// BasicResponse is a basic response definition
type BasicResponse struct {
	Success  bool
	ErrorMsg string
}

// DBForumPost is the definition of forum post in DB
type DBForumPost struct {
	ID       string   `bson:"_id" json:"_id"`
	Title    string   `bson:"title" json:"title"`
	Content  string   `bson:"content" json:"content"`
	Image    string   `bson:"image" json:"image"`
	UserID   string   `bson:"userId" json:"userId"`
	VoteID   string   `bson:"voteId" json:"voteId"`
	Metadata Metadata `bson:"metadata" json:"metadata"`
}

// ForumPostV2 is the definition of a forum post sent back to mobile
type ForumPostV2 struct {
	ID          string   `bson:"_id" json:"_id"`
	Title       string   `bson:"title" json:"title"`
	Content     string   `bson:"content" json:"content"`
	Image       string   `bson:"image" json:"image"`
	UserProfile Profile  `bson:"userProfile" json:"userProfile"`
	VoteID      string   `bson:"voteId" json:"voteId"`
	Metadata    Metadata `bson:"metadata" json:"metadata"`
}

// DBForumVote is the definition of a forum vote in DB
type DBForumVote struct {
	ID       string   `bson:"_id" json:"_id"`
	Count    int64    `bson:"count" json:"count"`
	Metadata Metadata `bson:"metadata" json:"metadata"`
}

// ForumVote is the definition of a forum vote
type ForumVote struct {
	ID         string   `bson:"_id" json:"_id"`
	Count      int64    `bson:"count" json:"count"`
	VoteStatus int      `bson:"voteStatus" json:"voteStatus"`
	Metadata   Metadata `bson:"metadata" json:"metadata"`
}

type ForumVoteUpdateRequest struct {
	VoteID     string   `bson:"voteId" json:"voteId"`
	Offset     int64    `bson:"offset" json:"offset"`
	VoteStatus int      `bson:"voteStatus" json:"voteStatus"`
	UserID     string   `bson:"userId" json:"userId"`
	Metadata   Metadata `bson:"metadata" json:"metadata"`
}
