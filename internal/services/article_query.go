package services

const MaxLimit = 100

type ArticleQuery struct {
	Tag    string
	Author string
	Limit  int
	Offset int
}
