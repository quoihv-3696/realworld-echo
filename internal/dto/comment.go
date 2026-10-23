package dto

import "time"

type Comment struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Body      string    `json:"body"`
	Author    Profile   `json:"author"`
}

type SingleCommentResponse struct {
	Comment Comment `json:"comment"`
}

type MultipleCommentsResponse struct {
	Comments []Comment `json:"comments"`
}

type NewComment struct {
	Body string `json:"body"`
}

type CreateCommentRequest struct {
	Comment NewComment `json:"comment"`
}
