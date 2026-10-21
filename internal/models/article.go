package models

import "time"

type Article struct {
	ID          uint   `gorm:"primaryKey"`
	Slug        string `gorm:"uniqueIndex;not null"`
	Title       string `gorm:"not null"`
	Description string `gorm:"not null"`
	Body        string `gorm:"not null"`
	AuthorID    uint   `gorm:"not null;index"`
	Author      User
	Tags        []Tag     `gorm:"many2many:article_tags;constraint:OnDelete:CASCADE"`
	Comments    []Comment `gorm:"constraint:OnDelete:CASCADE"`
	CreatedAt   time.Time `gorm:"index"`
	UpdatedAt   time.Time
}
