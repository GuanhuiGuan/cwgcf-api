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
	Timestamp   int64      `bson:"timestamp" json:"timestamp"`
	UpdatedAt   int64      `bson:"updatedAt" json:"updatedAt"`
	UserID      string     `bson:"userId" json:"userId"`
	UserProfile Profile    `bson:"userProfile" json:"userProfile"`
	ForumVotes  ForumVotes `bson:"forumVotes" json:"forumVotes"`
}

// ForumSubComments is the definition of subcomment ids of a post/comment
type ForumSubComments struct {
	ParentID   string   `bson:"parentId" json:"parentId"`
	CommentIDs []string `bson:"commentIds" json:"commentIds"`
}

// ForumVotes is the definition of votes of a forum post/comment
type ForumVotes struct {
	Upvotes   int64 `bson:"upvotes" json:"upvotes"`
	Downvotes int64 `bson:"downvotes" json:"downvotes"`
}

// ForumComment is the definition of a forum comment
// Subcomments are usually fetched alongside with parent comments
type ForumComment struct {
	ID          string         `bson:"_id" json:"_id"`
	ParentID    string         `bson:"parentId" json:"parentId"`
	Content     string         `bson:"content" json:"content"`
	Timestamp   int64          `bson:"timestamp" json:"timestamp"`
	UpdatedAt   int64          `bson:"updatedAt" json:"updatedAt"`
	UserID      string         `bson:"userId" json:"userId"`
	UserProfile Profile        `bson:"userProfile" json:"userProfile"`
	ForumVotes  ForumVotes     `bson:"forumVotes" json:"forumVotes"`
	Comments    []ForumComment `bson:"comments" json:"comments"`
}
