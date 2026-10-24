package services

import (
	"context"
	"errors"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/dto"
	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
)

type CommentService struct {
	articles repository.ArticleRepository
	comments repository.CommentRepository
	follows  repository.FollowRepository
}

func NewCommentService(articles repository.ArticleRepository, comments repository.CommentRepository, follows repository.FollowRepository) *CommentService {
	return &CommentService{articles: articles, comments: comments, follows: follows}
}

func (s *CommentService) List(ctx context.Context, viewerID uint, articleSlug string) (dto.MultipleCommentsResponse, error) {
	a, err := findArticle(ctx, s.articles, articleSlug)
	if err != nil {
		return dto.MultipleCommentsResponse{}, err
	}
	comments, err := s.comments.ListByArticle(ctx, a.ID)
	if err != nil {
		return dto.MultipleCommentsResponse{}, err
	}
	authorIDs := make([]uint, 0, len(comments))
	for _, c := range comments {
		authorIDs = append(authorIDs, c.AuthorID)
	}
	following, err := followingAmong(ctx, s.follows, viewerID, authorIDs)
	if err != nil {
		return dto.MultipleCommentsResponse{}, err
	}
	out := make([]dto.Comment, 0, len(comments))
	for _, c := range comments {
		item := toComment(c)
		item.Author.Following = following[c.AuthorID]
		out = append(out, item)
	}
	return dto.MultipleCommentsResponse{Comments: out}, nil
}

func (s *CommentService) Create(ctx context.Context, authorID uint, articleSlug string, req dto.CreateCommentRequest) (dto.SingleCommentResponse, error) {
	if req.Comment.Body == "" {
		return dto.SingleCommentResponse{}, apperr.Blank("body")
	}
	a, err := findArticle(ctx, s.articles, articleSlug)
	if err != nil {
		return dto.SingleCommentResponse{}, err
	}
	c := models.Comment{Body: req.Comment.Body, ArticleID: a.ID, AuthorID: authorID}
	if err := s.comments.Create(ctx, &c); err != nil {
		return dto.SingleCommentResponse{}, err
	}
	return dto.SingleCommentResponse{Comment: toComment(c)}, nil
}

func (s *CommentService) Delete(ctx context.Context, viewerID uint, articleSlug string, commentID uint) error {
	a, err := findArticle(ctx, s.articles, articleSlug)
	if err != nil {
		return err
	}
	c, err := s.comments.FindByID(ctx, commentID)
	if errors.Is(err, repository.ErrNotFound) {
		return apperr.NotFound("comment")
	}
	if err != nil {
		return err
	}
	if c.ArticleID != a.ID {
		return apperr.NotFound("comment")
	}
	if c.AuthorID != viewerID {
		return apperr.Forbidden("comment")
	}
	return s.comments.Delete(ctx, c.ID)
}
