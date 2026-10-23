package dto

import "time"

type ArticleListItem struct {
	Slug           string    `json:"slug"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	TagList        []string  `json:"tagList"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Favorited      bool      `json:"favorited"`
	FavoritesCount int64     `json:"favoritesCount"`
	Author         Profile   `json:"author"`
}

type Article struct {
	ArticleListItem
	Body string `json:"body"`
}

type SingleArticleResponse struct {
	Article Article `json:"article"`
}

type MultipleArticlesResponse struct {
	Articles      []ArticleListItem `json:"articles"`
	ArticlesCount int64             `json:"articlesCount"`
}

type NewArticle struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Body        string   `json:"body"`
	TagList     []string `json:"tagList"`
}

type CreateArticleRequest struct {
	Article NewArticle `json:"article"`
}

type UpdateArticle struct {
	Title       Optional[string]   `json:"title"`
	Description Optional[string]   `json:"description"`
	Body        Optional[string]   `json:"body"`
	TagList     Optional[[]string] `json:"tagList"`
}

type UpdateArticleRequest struct {
	Article UpdateArticle `json:"article"`
}
