-- Migration: Create read_progress table
-- Version: 001

CREATE TABLE IF NOT EXISTS read_progress (
    user_id UUID NOT NULL,
    manga_id UUID NOT NULL,
    chapter_id UUID NOT NULL,
    page_number INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, manga_id, chapter_id)
);

CREATE INDEX IF NOT EXISTS idx_read_progress_user_id ON read_progress(user_id);
CREATE INDEX IF NOT EXISTS idx_read_progress_manga_id ON read_progress(manga_id);
