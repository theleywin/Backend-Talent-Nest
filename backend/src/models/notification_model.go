package models

import (
	"gorm.io/gorm"
)

type Notification struct {
	gorm.Model
	RecipientID   uint   `json:"recipient" gorm:"index"`
	Type          string `json:"type" gorm:"type:varchar(50)"`
	RelatedUserID uint  `json:"related_user_id" gorm:"default:null"`
	RelatedPostID uint  `json:"related_post_id" gorm:"default:null"`
	Read          bool   `json:"read" gorm:"default:false"`
	Recipient     User   `json:"-" gorm:"foreignKey:RecipientID"`
	RelatedUser   *User  `json:"-" gorm:"foreignKey:RelatedUserID"`
	RelatedPost   *Post  `json:"-" gorm:"foreignKey:RelatedPostID"`
}
