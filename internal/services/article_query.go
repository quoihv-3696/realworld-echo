package services

import (
	"net/url"
	"strconv"
)

const MaxLimit = 100

type ArticleQuery struct {
	Tag       string
	Author    string
	Favorited string
	Limit     int
	Offset    int
}

func (q ArticleQuery) cacheKey() string {
	return "articles?" + url.Values{
		"tag":       {q.Tag},
		"author":    {q.Author},
		"favorited": {q.Favorited},
		"limit":     {strconv.Itoa(q.Limit)},
		"offset":    {strconv.Itoa(q.Offset)},
	}.Encode()
}
