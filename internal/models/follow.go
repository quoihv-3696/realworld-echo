package models

import "time"

type Follow struct {
	FollowerID  uint `gorm:"primaryKey"`
	FollowingID uint `gorm:"primaryKey;index"`
	Follower    User `gorm:"constraint:OnDelete:CASCADE"`
	Following   User `gorm:"constraint:OnDelete:CASCADE"`
	CreatedAt   time.Time
}
