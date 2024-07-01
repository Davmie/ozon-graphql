package memory

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"ozon-graphql/graph/model"
	"sync"
	"time"
)

type InMemoryDB struct {
	posts         map[string]*model.Post
	comments      map[string][]*model.Comment
	subscriptions map[string][]chan *model.Comment
	mu            sync.RWMutex
}

func NewInMemoryDB() *InMemoryDB {
	return &InMemoryDB{
		posts:         make(map[string]*model.Post),
		comments:      make(map[string][]*model.Comment),
		subscriptions: make(map[string][]chan *model.Comment),
	}
}

func (db *InMemoryDB) CreatePost(ctx context.Context, title string, content string, commentsEnabled bool) (*model.Post, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	post := &model.Post{
		ID:              generateID(),
		Title:           title,
		Content:         content,
		CommentsEnabled: commentsEnabled,
		Comments:        []*model.Comment{},
	}
	db.posts[post.ID] = post
	return post, nil
}

func (db *InMemoryDB) CreateComment(ctx context.Context, postId string, parentId *string, content string) (*model.Comment, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	comment := &model.Comment{
		ID:        generateID(),
		PostID:    postId,
		ParentID:  parentId,
		Content:   content,
		CreatedAt: getCurrentTime(),
		Replies:   []*model.Comment{},
	}

	db.comments[postId] = append(db.comments[postId], comment)

	for _, ch := range db.subscriptions[postId] {
		ch <- comment
	}

	return comment, nil
}

func (db *InMemoryDB) GetPosts(ctx context.Context) ([]*model.Post, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	posts := make([]*model.Post, 0, len(db.posts))
	for _, post := range db.posts {
		posts = append(posts, post)
	}

	return posts, nil
}

func (db *InMemoryDB) GetPostByID(ctx context.Context, id string) (*model.Post, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	post, ok := db.posts[id]
	if !ok {
		return nil, fmt.Errorf("post not found")
	}

	post.Comments = db.comments[id]
	return post, nil
}

func (db *InMemoryDB) GetCommentsByPostID(ctx context.Context, postId string, limit, offset int) ([]*model.Comment, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	comments, ok := db.comments[postId]
	if !ok {
		return nil, fmt.Errorf("post not found")
	}

	start := offset
	end := offset + limit

	if start > len(comments) {
		return []*model.Comment{}, nil
	}

	if end > len(comments) {
		end = len(comments)
	}

	return comments[start:end], nil
}

func (db *InMemoryDB) SubscribeToComments(ctx context.Context, postId string) (<-chan *model.Comment, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	ch := make(chan *model.Comment)
	db.subscriptions[postId] = append(db.subscriptions[postId], ch)
	return ch, nil
}

func generateID() string {
	return uuid.New().String()
}

func getCurrentTime() string {
	return time.Now().Format(time.RFC3339)
}
