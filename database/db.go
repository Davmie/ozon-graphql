package database

import (
	"context"
	"ozon-graphql/graph/model"
)

type DB interface {
	CreatePost(ctx context.Context, title string, content string, commentsEnabled bool) (*model.Post, error)
	CreateComment(ctx context.Context, postId string, parentId *string, content string) (*model.Comment, error)
	GetPosts(ctx context.Context) ([]*model.Post, error)
	GetPostByID(ctx context.Context, id string) (*model.Post, error)
	GetCommentsByPostID(ctx context.Context, postId string, limit, offset int) ([]*model.Comment, error)
	SubscribeToComments(ctx context.Context, postId string) (<-chan *model.Comment, error)
}
