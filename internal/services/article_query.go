package services

const MaxLimit = 100

type ArticleQuery struct {
	Tag       string
	Author    string
	Favorited string
	Limit     int
	Offset    int
}
