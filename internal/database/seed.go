package database

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/quoihv-3696/realworld-echo/internal/models"
)

const SeedPassword = "password123"

func Seed(ctx context.Context, db *gorm.DB) (bool, error) {
	var n int64
	if err := db.WithContext(ctx).Model(&models.User{}).Count(&n).Error; err != nil {
		return false, err
	}
	if n > 0 {
		return false, nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(SeedPassword), bcrypt.DefaultCost)
	if err != nil {
		return false, fmt.Errorf("hash mật khẩu mẫu: %w", err)
	}
	bio := "Tôi thích viết về Go."
	users := []models.User{
		{Username: "jake", Email: "jake@example.com", Password: string(hash), Bio: &bio},
		{Username: "anna", Email: "anna@example.com", Password: string(hash)},
		{Username: "minh", Email: "minh@example.com", Password: string(hash)},
	}

	base := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	tag := func(name string) models.Tag { return models.Tag{Name: name} }
	articles := []struct {
		author int
		a      models.Article
	}{
		{0, models.Article{Slug: "how-to-train-your-dragon", Title: "How to train your dragon", Description: "Ever wonder how?", Body: "You have to believe.", Tags: []models.Tag{tag("dragons"), tag("training")}}},
		{1, models.Article{Slug: "hoc-go-trong-30-ngay", Title: "Học Go trong 30 ngày", Description: "Lộ trình tự học", Body: "Bắt đầu từ cú pháp cơ bản.", Tags: []models.Tag{tag("go"), tag("beginner")}}},
		{0, models.Article{Slug: "echo-v5-co-gi-moi", Title: "Echo v5 có gì mới", Description: "So sánh với v4", Body: "Handler nhận *echo.Context.", Tags: []models.Tag{tag("go"), tag("echo")}}},
		{2, models.Article{Slug: "gorm-va-n-plus-1", Title: "GORM và N+1", Description: "Preload cứu bạn", Body: "Bật logger để thấy câu SQL thật.", Tags: []models.Tag{tag("go"), tag("gorm")}}},
		{1, models.Article{Slug: "bai-viet-khong-co-tag", Title: "Bài viết không có tag", Description: "tagList rỗng", Body: "tagList phải là [] chứ không phải null."}},
	}

	return true, db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&users).Error; err != nil {
			return err
		}
		tags := map[string]*models.Tag{}
		for i, item := range articles {
			a := item.a
			a.AuthorID = users[item.author].ID
			a.CreatedAt = base.Add(time.Duration(i) * time.Hour)
			a.UpdatedAt = a.CreatedAt
			a.Tags = nil
			for _, t := range item.a.Tags {
				if tags[t.Name] == nil {
					tags[t.Name] = &models.Tag{Name: t.Name}
					if err := tx.Create(tags[t.Name]).Error; err != nil {
						return err
					}
				}
				a.Tags = append(a.Tags, *tags[t.Name])
			}
			if err := tx.Create(&a).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
