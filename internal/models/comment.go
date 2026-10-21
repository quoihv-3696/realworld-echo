package models

import "time"

type Comment struct {
	ID        uint   `gorm:"primaryKey"`
	Body      string `gorm:"not null"`
	ArticleID uint   `gorm:"not null;index"`
	AuthorID  uint   `gorm:"not null;index"`
	Author    User
	CreatedAt time.Time
	UpdatedAt time.Time
}
