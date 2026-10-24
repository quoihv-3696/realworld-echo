package models

import "time"

type Favorite struct {
	UserID    uint    `gorm:"primaryKey"`
	ArticleID uint    `gorm:"primaryKey;index"`
	User      User    `gorm:"constraint:OnDelete:CASCADE"`
	Article   Article `gorm:"constraint:OnDelete:CASCADE"`
	CreatedAt time.Time
}
