package models

// Profile is the definition of a user profile
type Profile struct {
	ID          string `bson:"_id" json:"_id"`
	Name        string `bson:"name" json:"name"`
	Title       string `bson:"title" json:"title"`
	Description string `bson:"description" json:"description"`
	AvatarURL   string `bson:"avatarUrl" json:"avatarUrl"`
}
