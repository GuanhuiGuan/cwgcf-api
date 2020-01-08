package models

// Profile is the definition of a user profile
type Profile struct {
	ID          string `bson:"_id" json:"_id"`
	Name        string `bson:"name" json:"name"`
	Title       string `bson:"title" json:"title"`
	Description string `bson:"description" json:"description"`
	AvatarURL   string `bson:"avatarUrl" json:"avatarUrl"`
}

// Photo is the definition of a photo
type Photo struct {
	ID  string `bson:"_id" json:"_id"`
	URL string `bson:"url" json:"url"`
}

// ForumPost is the definition of a post in forum
// Save comments in a different table with key being post ID because comments are usually not fetched at the same time the content is fetched
type ForumPost struct {
	ID          string     `bson:"_id" json:"_id"`
	Title       string     `bson:"title" json:"title"`
	Content     string     `bson:"content" json:"content"`
	Image       string     `bson:"image" json:"image"`
	CreatedAt   int64      `bson:"createdAt" json:"createdAt"`
	UpdatedAt   int64      `bson:"updatedAt" json:"updatedAt"`
	UserID      string     `bson:"userId" json:"userId"`
	UserProfile Profile    `bson:"userProfile" json:"userProfile"`
	ForumVotes  ForumVotes `bson:"forumVotes" json:"forumVotes"`
}

// ForumVotes is the definition of votes of a forum post/comment
// Upvotes/downvotes not used at the moment
type ForumVotes struct {
	Upvotes   int64 `bson:"upvotes" json:"upvotes"`
	Downvotes int64 `bson:"downvotes" json:"downvotes"`
	VotesSum  int64 `bson:"votesSum" json:"votesSum"`
}

// ForumComment is the definition of a forum comment
// Subcomments are usually fetched alongside with parent comments
type ForumComment struct {
	ID          string         `bson:"_id" json:"_id"`
	ParentID    string         `bson:"parentId" json:"parentId"`
	Content     string         `bson:"content" json:"content"`
	CreatedAt   int64          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   int64          `bson:"updatedAt" json:"updatedAt"`
	UserID      string         `bson:"userId" json:"userId"`
	UserProfile Profile        `bson:"userProfile" json:"userProfile"`
	ForumVotes  ForumVotes     `bson:"forumVotes" json:"forumVotes"`
	Comments    []ForumComment `bson:"comments" json:"comments"`
}

//ForumVoteRequest is the definition for vote request
/*
	IsPost indicates if vote is for forumPost or forumComment
	TapUpvote indicates if tapped on upvote or downvote (could be vote or unvote, will be decided in Backend)
*/
type ForumVoteRequest struct {
	VoteID    string `bson:"voteId" json:"voteId"`
	UserID    string `bson:"userId" json:"userId"`
	IsPost    bool   `bson:"isPost" json:"isPost"`
	UpdatedAt int64  `bson:"updatedAt" json:"updatedAt"`
	TapUpvote bool   `bson:"tapUpvote" json:"tapUpvote"`
}

// ForumUserVotes is the definition to record what the user voted
type ForumUserVotes struct {
	UserID  string                       `bson:"userId" json:"userId"`
	VoteMap map[string]ForumVoteWithTime `bson:"voteMap" json:"voteMap"`
}

// ForumVoteWithTime is the definition to record user's vote and the timestamp
// VoteStatus: 0: unvoted, 1: upvoted, -1: downvoted
type ForumVoteWithTime struct {
	VoteStatus int   `bson:"voteStatus" json:"voteStatus"`
	UpdatedAt  int64 `bson:"updatedAt" json:"updatedAt"`
}
