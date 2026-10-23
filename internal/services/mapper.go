package services

import (
	"slices"

	"github.com/quoihv-3696/realworld-echo/internal/dto"
	"github.com/quoihv-3696/realworld-echo/internal/models"
)

func toProfile(u models.User) dto.Profile {
	return dto.Profile{
		Username:  u.Username,
		Bio:       u.Bio,
		Image:     u.Image,
		Following: false,
	}
}

func toArticleListItem(a models.Article) dto.ArticleListItem {
	tags := make([]string, 0, len(a.Tags))
	for _, t := range a.Tags {
		tags = append(tags, t.Name)
	}
	slices.Sort(tags)

	return dto.ArticleListItem{
		Slug:           a.Slug,
		Title:          a.Title,
		Description:    a.Description,
		TagList:        tags,
		CreatedAt:      a.CreatedAt.UTC(),
		UpdatedAt:      a.UpdatedAt.UTC(),
		Favorited:      false,
		FavoritesCount: 0,
		Author:         toProfile(a.Author),
	}
}

func toArticle(a models.Article) dto.Article {
	return dto.Article{ArticleListItem: toArticleListItem(a), Body: a.Body}
}

func toUser(u models.User, token string) dto.User {
	return dto.User{
		Email:    u.Email,
		Token:    token,
		Username: u.Username,
		Bio:      u.Bio,
		Image:    u.Image,
	}
}

func toComment(c models.Comment) dto.Comment {
	return dto.Comment{
		ID:        c.ID,
		CreatedAt: c.CreatedAt.UTC(),
		UpdatedAt: c.UpdatedAt.UTC(),
		Body:      c.Body,
		Author:    toProfile(c.Author),
	}
}
