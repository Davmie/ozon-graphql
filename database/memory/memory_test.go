package memory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMemoryDB() *InMemoryDB {
	return NewInMemoryDB()
}

func TestMemoryCreatePost(t *testing.T) {
	db := setupMemoryDB()
	ctx := context.Background()

	post, err := db.CreatePost(ctx, "Test Title", "Test Content", true)
	require.NoError(t, err)

	assert.Equal(t, "Test Title", post.Title)
	assert.Equal(t, "Test Content", post.Content)
	assert.True(t, post.CommentsEnabled)
}

func TestMemoryCreateComment(t *testing.T) {
	db := setupMemoryDB()
	ctx := context.Background()

	post, err := db.CreatePost(ctx, "Test Title", "Test Content", true)
	require.NoError(t, err)

	comment, err := db.CreateComment(ctx, post.ID, nil, "Test Comment")
	require.NoError(t, err)

	assert.Equal(t, post.ID, comment.PostID)
	assert.Nil(t, comment.ParentID)
	assert.Equal(t, "Test Comment", comment.Content)
}

func TestMemoryGetPosts(t *testing.T) {
	db := setupMemoryDB()
	ctx := context.Background()

	_, err := db.CreatePost(ctx, "Test Title 1", "Test Content 1", true)
	require.NoError(t, err)
	_, err = db.CreatePost(ctx, "Test Title 2", "Test Content 2", false)
	require.NoError(t, err)

	posts, err := db.GetPosts(ctx)
	require.NoError(t, err)

	assert.Len(t, posts, 2)
}

func TestMemoryGetPostByID(t *testing.T) {
	db := setupMemoryDB()
	ctx := context.Background()

	post, err := db.CreatePost(ctx, "Test Title", "Test Content", true)
	require.NoError(t, err)

	fetchedPost, err := db.GetPostByID(ctx, post.ID)
	require.NoError(t, err)

	assert.Equal(t, post.ID, fetchedPost.ID)
	assert.Equal(t, "Test Title", fetchedPost.Title)
}

func TestMemorySubscribeToComments(t *testing.T) {
	db := setupMemoryDB()
	ctx := context.Background()

	post, err := db.CreatePost(ctx, "Test Title", "Test Content", true)
	require.NoError(t, err)

	ch, err := db.SubscribeToComments(ctx, post.ID)
	require.NoError(t, err)

	go func() {
		time.Sleep(1 * time.Second)
		_, err = db.CreateComment(ctx, post.ID, nil, "Test Comment")
		require.NoError(t, err)
	}()

	select {
	case comment := <-ch:
		assert.Equal(t, "Test Comment", comment.Content)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for comment notification")
	}
}
