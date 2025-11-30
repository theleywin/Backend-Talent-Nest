package models

import (
	"time"

	"gorm.io/gorm"
)

type Post struct {
	gorm.Model
	AuthorID uint      `json:"author" gorm:"index"`
	Content  string    `json:"content" gorm:"type:text"`
	Image    string    `json:"image"`
	RepostID *uint     `json:"repost"`
	Likes    []Like    `json:"likes" gorm:"foreignKey:PostID"`
	Comments []Comment `json:"comments" gorm:"foreignKey:PostID"`
	Author   User      `json:"-" gorm:"foreignKey:AuthorID"`
	Repost   *Post     `json:"-" gorm:"foreignKey:RepostID"`
}

type PostDto struct {
	ID        uint         `json:"_id"`
	Author    UserDto      `json:"author"`
	Content   string       `json:"content"`
	Image     string       `json:"image"`
	Repost    *PostDto     `json:"repost,omitempty"`
	Likes     []UserDto    `json:"likes"`
	Comments  []CommentDto `json:"comments"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
}

type Comment struct {
	gorm.Model
	PostID  uint   `json:"post_id" gorm:"index"`
	UserID  uint   `json:"user_id" gorm:"index"`
	Content string `json:"content" gorm:"type:text"`
	User    User   `json:"-" gorm:"foreignKey:UserID"`
}

type CommentDto struct {
	ID        uint      `json:"_id"`
	Content   string    `json:"content"`
	User      UserDto   `json:"user"`
	CreatedAt time.Time `json:"createdAt"`
}

type Like struct {
	gorm.Model
	PostID uint `json:"post_id" gorm:"index"`
	UserID uint `json:"user_id" gorm:"index"`
	User   User `json:"-" gorm:"foreignKey:UserID"`
}
