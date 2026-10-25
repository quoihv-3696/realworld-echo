package services

import (
	"net/url"
	"slices"
	"strconv"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
)

const MaxLimit = 100

type ArticleQuery struct {
	Tag       string
	Author    string
	Favorited string
	Sort      string
	Limit     int
	Offset    int
}

func (q ArticleQuery) cacheKey() string {
	return "articles?" + url.Values{
		"tag":       {q.Tag},
		"author":    {q.Author},
		"favorited": {q.Favorited},
		"sort":      {q.Sort},
		"limit":     {strconv.Itoa(q.Limit)},
		"offset":    {strconv.Itoa(q.Offset)},
	}.Encode()
}

var articleSorts = []string{repository.SortNewest, repository.SortOldest, repository.SortPopular}

func normalizeSort(sort string) (string, error) {
	if sort == "" {
		return repository.SortNewest, nil
	}
	if !slices.Contains(articleSorts, sort) {
		return "", apperr.Invalid("sort", "must be one of newest, oldest, popular")
	}
	return sort, nil
}
