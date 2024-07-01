package postgres

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMock(t *testing.T) (*PostgresDB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "postgres")
	return &PostgresDB{db: sqlxDB}, mock
}

func TestCreatePost(t *testing.T) {
	db, mock := setupMock(t)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO posts (id, title, content, commentsEnabled) VALUES ($1, $2, $3, $4)")).
		WithArgs(sqlmock.AnyArg(), "Test Title", "Test Content", true).
		WillReturnResult(sqlmock.NewResult(1, 1))

	post, err := db.CreatePost(ctx, "Test Title", "Test Content", true)
	require.NoError(t, err)

	assert.Equal(t, "Test Title", post.Title)
	assert.Equal(t, "Test Content", post.Content)
	assert.True(t, post.CommentsEnabled)
}

func TestCreateComment(t *testing.T) {
	db, mock := setupMock(t)
	ctx := context.Background()

	postID := "post_id"
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO comments (id, postId, parentId, content, createdAt) VALUES ($1, $2, $3, $4, $5)")).
		WithArgs(sqlmock.AnyArg(), postID, nil, "Test Comment", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	comment, err := db.CreateComment(ctx, postID, nil, "Test Comment")
	require.NoError(t, err)

	assert.Equal(t, postID, comment.PostID)
	assert.Nil(t, comment.ParentID)
	assert.Equal(t, "Test Comment", comment.Content)
}

func TestGetPosts(t *testing.T) {
	db, mock := setupMock(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "title", "content", "commentsenabled"}).
		AddRow("1", "Test Title 1", "Test Content 1", true).
		AddRow("2", "Test Title 2", "Test Content 2", false)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM posts")).
		WillReturnRows(rows)

	posts, err := db.GetPosts(ctx)
	require.NoError(t, err)

	assert.Len(t, posts, 2)
	assert.Equal(t, "Test Title 1", posts[0].Title)
	assert.Equal(t, "Test Title 2", posts[1].Title)
}

func TestGetPostByID(t *testing.T) {
	db, mock := setupMock(t)
	ctx := context.Background()

	postID := "post_id"
	rows := sqlmock.NewRows([]string{"id", "title", "content", "commentsenabled"}).
		AddRow(postID, "Test Title", "Test Content", true)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM posts WHERE id = $1")).
		WithArgs(postID).
		WillReturnRows(rows)

	commentRows := sqlmock.NewRows([]string{"id", "postid", "parentid", "content", "createdat"}).
		AddRow("comment_id", postID, "parent_id", "Test Comment", "Rara")
	mock.ExpectQuery(regexp.QuoteMeta(`WITH RECURSIVE comment_tree AS (
                SELECT id, postId, parentId, content, createdAt
                FROM comments
                WHERE postId = $1 AND parentId IS NULL
                UNION ALL
                SELECT c.id, c.postId, c.parentId, c.content, c.createdAt
                FROM comments c
                INNER JOIN comment_tree ct ON ct.id = c.parentId
              )
              SELECT id, postId, parentId, content, createdAt FROM comment_tree`)).
		WithArgs(postID).WillReturnRows(commentRows)

	fetchedPost, err := db.GetPostByID(ctx, postID)
	require.NoError(t, err)

	assert.Equal(t, postID, fetchedPost.ID)
	assert.Equal(t, "Test Title", fetchedPost.Title)
}
