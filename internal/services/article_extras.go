package services

import (
	"context"

	"github.com/quoihv-3696/realworld-echo/internal/dto"
	"github.com/quoihv-3696/realworld-echo/internal/models"
)

type articleExtras struct {
	following map[uint]bool
	favorited map[uint]bool
	counts    map[uint]int64
}

func (s *ArticleService) loadExtras(ctx context.Context, viewerID uint, articles []models.Article) (articleExtras, error) {
	articleIDs := make([]uint, 0, len(articles))
	authorIDs := make([]uint, 0, len(articles))
	for _, a := range articles {
		articleIDs = append(articleIDs, a.ID)
		authorIDs = append(authorIDs, a.AuthorID)
	}

	counts, err := s.favorites.CountByArticle(ctx, articleIDs)
	if err != nil {
		return articleExtras{}, err
	}
	following, err := followingAmong(ctx, s.follows, viewerID, authorIDs)
	if err != nil {
		return articleExtras{}, err
	}
	favorited := map[uint]bool{}
	if viewerID != 0 {
		favorited, err = s.favorites.FavoritedAmong(ctx, viewerID, articleIDs)
		if err != nil {
			return articleExtras{}, err
		}
	}
	return articleExtras{following: following, favorited: favorited, counts: counts}, nil
}

func (x articleExtras) fill(item *dto.ArticleListItem, a models.Article) {
	item.Author.Following = x.following[a.AuthorID]
	item.Favorited = x.favorited[a.ID]
	item.FavoritesCount = x.counts[a.ID]
}
