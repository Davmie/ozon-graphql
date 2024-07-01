package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"log"
	"ozon-graphql/graph/model"
	"time"
)

type PostgresDB struct {
	dsn string
	db  *sqlx.DB
}

func NewPostgresDB(dsn string) (*PostgresDB, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}
	return &PostgresDB{dsn: dsn, db: db}, nil
}

func (db *PostgresDB) CreatePost(ctx context.Context, title string, content string, commentsEnabled bool) (*model.Post, error) {
	post := &model.Post{
		ID:              generateID(),
		Title:           title,
		Content:         content,
		CommentsEnabled: commentsEnabled,
		Comments:        make([]*model.Comment, 0),
	}

	query := `INSERT INTO posts (id, title, content, commentsEnabled) VALUES ($1, $2, $3, $4)`
	_, err := db.db.ExecContext(ctx, query, post.ID, post.Title, post.Content, post.CommentsEnabled)
	if err != nil {
		return nil, err
	}

	return post, nil
}

func (db *PostgresDB) CreateComment(ctx context.Context, postId string, parentId *string, content string) (*model.Comment, error) {
	comment := &model.Comment{
		ID:        generateID(),
		PostID:    postId,
		ParentID:  parentId,
		Content:   content,
		CreatedAt: getCurrentTime(),
		Replies:   make([]*model.Comment, 0),
	}

	query := `INSERT INTO comments (id, postId, parentId, content, createdAt) VALUES ($1, $2, $3, $4, $5)`
	_, err := db.db.ExecContext(ctx, query, comment.ID, comment.PostID, comment.ParentID, comment.Content, comment.CreatedAt)
	if err != nil {
		return nil, err
	}

	//notifyQuery := `NOTIFY new_comment, 'created'`
	//
	////arg := fmt.Sprintf("for %s with id %s", postId, comment.ID)
	//_, err = db.db.ExecContext(ctx, notifyQuery)
	//if err != nil {
	//	return nil, err
	//}
	//ctx = context.WithValue(ctx, "commentId", comment.ID)

	return comment, nil
}

func (db *PostgresDB) GetPosts(ctx context.Context) ([]*model.Post, error) {
	var posts []*model.Post
	query := `SELECT * FROM posts`
	err := db.db.SelectContext(ctx, &posts, query)
	if err != nil {
		return nil, err
	}

	for _, post := range posts {
		post.Comments = make([]*model.Comment, 0)
	}

	return posts, nil
}

func (db *PostgresDB) GetPostByID(ctx context.Context, id string) (*model.Post, error) {
	post := &model.Post{}
	query := `SELECT * FROM posts WHERE id = $1`
	err := db.db.GetContext(ctx, post, query, id)
	if err != nil {
		return nil, err
	}

	comments, err := db.getCommentsByPostID(ctx, id)
	if err != nil {
		return nil, err
	}

	post.Comments = comments
	return post, nil
}

// Оптимизация для проблем n+1 запросов и глубокой вложенности комментариев
func (db *PostgresDB) getCommentsByPostID(ctx context.Context, postId string) ([]*model.Comment, error) {
	var comments []*model.Comment
	query := `WITH RECURSIVE comment_tree AS (
                SELECT id, postId, parentId, content, createdAt
                FROM comments
                WHERE postId = $1 AND parentId IS NULL
                UNION ALL
                SELECT c.id, c.postId, c.parentId, c.content, c.createdAt
                FROM comments c
                INNER JOIN comment_tree ct ON ct.id = c.parentId
              )
              SELECT id, postId, parentId, content, createdAt FROM comment_tree`
	err := db.db.SelectContext(ctx, &comments, query, postId)
	if err != nil {
		return nil, err
	}

	commentMap := make(map[string]*model.Comment)
	for _, comment := range comments {
		commentMap[comment.ID] = comment
	}

	var result []*model.Comment
	for _, comment := range comments {
		if comment.ParentID == nil {
			result = append(result, comment)
		} else {
			parent, ok := commentMap[*comment.ParentID]
			if ok {
				parent.Replies = append(parent.Replies, comment)
			}
		}
	}

	return result, nil
}

// GetCommentsByPostID Функция с поддержкой пагинации
func (db *PostgresDB) GetCommentsByPostID(ctx context.Context, postId string, limit, offset int) ([]*model.Comment, error) {
	comments := []*model.Comment{}
	query := `SELECT id, postId, parentId, content, createdAt 
              FROM comments 
              WHERE postId = $1 
              ORDER BY createdAt 
              LIMIT $2 OFFSET $3`
	err := db.db.SelectContext(ctx, &comments, query, postId, limit, offset)
	if err != nil {
		return nil, err
	}

	return comments, nil
}

func (db *PostgresDB) SubscribeToComments(ctx context.Context, postId string) (<-chan *model.Comment, error) {
	ch := make(chan *model.Comment)

	go func() {
		defer close(ch)

		listener := pq.NewListener(db.dsn, 10*time.Second, time.Minute, nil)
		err := listener.Listen("new_comment")
		if err != nil {
			log.Printf("failed to listen on new_comment channel: %v", err)
			return
		}

		fmt.Printf("listening on new_comment channel\n")

		for notif := range listener.Notify {
			if notif == nil {
				continue
			}

			var commentID, commentPostID string
			fmt.Sscanf(notif.Extra, "%s,%s", &commentID, &commentPostID)

			if commentPostID == postId {
				var comment *model.Comment
				query := `SELECT * FROM comments WHERE id = $1`
				err = db.db.GetContext(ctx, &comment, query, commentID)
				if err != nil {
					log.Printf("failed to get comment: %v", err)
					continue
				}

				ch <- comment
			}
		}
	}()

	return ch, nil
}

func generateID() string {
	return uuid.New().String()
}

func getCurrentTime() string {
	return time.Now().Format(time.RFC3339)
}
