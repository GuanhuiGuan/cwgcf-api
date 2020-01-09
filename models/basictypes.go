package models

// Metadata stores userID and timestamp of creation and update
type Metadata struct {
	CreatedBy string `bson:"createdBy" json:"createdBy"`
	CreatedAt int64  `bson:"createdAt" json:"createdAt"`
	UpdatedBy string `bson:"updatedBy" json:"updatedBy"`
	UpdatedAt int64  `bson:"updatedAt" json:"updatedAt"`
}
