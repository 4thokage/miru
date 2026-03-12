package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

type ReadProgress struct {
	UserID     string    `json:"user_id"`
	MangaID    string    `json:"manga_id"`
	ChapterID  string    `json:"chapter_id"`
	PageNumber int       `json:"page_number"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ProgressRepository struct{}

func NewProgressRepository() *ProgressRepository {
	return &ProgressRepository{}
}

func (r *ProgressRepository) Upsert(ctx context.Context, progress *ReadProgress) error {
	if Pool == nil {
		return errors.New("database connection not initialized")
	}

	query := `
		INSERT INTO read_progress (user_id, manga_id, chapter_id, page_number, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id, manga_id, chapter_id)
		DO UPDATE SET 
			page_number = EXCLUDED.page_number,
			updated_at = NOW()
	`

	_, err := Pool.Exec(ctx, query,
		progress.UserID,
		progress.MangaID,
		progress.ChapterID,
		progress.PageNumber,
	)

	if err != nil {
		logger.Error("failed to upsert progress",
			slog.String("error", err.Error()),
			slog.String("user_id", progress.UserID),
			slog.String("manga_id", progress.MangaID),
			slog.String("chapter_id", progress.ChapterID),
		)
		return err
	}

	logger.Info("progress upserted",
		slog.String("user_id", progress.UserID),
		slog.String("manga_id", progress.MangaID),
		slog.String("chapter_id", progress.ChapterID),
		slog.Int("page_number", progress.PageNumber),
	)

	return nil
}

func (r *ProgressRepository) Get(ctx context.Context, userID, mangaID, chapterID string) (*ReadProgress, error) {
	if Pool == nil {
		return nil, errors.New("database connection not initialized")
	}

	query := `
		SELECT user_id, manga_id, chapter_id, page_number, updated_at
		FROM read_progress
		WHERE user_id = $1 AND manga_id = $2 AND chapter_id = $3
	`

	var progress ReadProgress
	err := Pool.QueryRow(ctx, query, userID, mangaID, chapterID).Scan(
		&progress.UserID,
		&progress.MangaID,
		&progress.ChapterID,
		&progress.PageNumber,
		&progress.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &progress, nil
}
