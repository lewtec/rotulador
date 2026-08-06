-- Images table stores information about images to be annotated
-- Uses SHA256 hash as primary key for content-based addressing
CREATE TABLE images (
  sha256 TEXT PRIMARY KEY,
  filename TEXT NOT NULL,
  ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Annotations table stores all annotations
-- Uses username directly from YAML config (no FK to users table)
CREATE TABLE annotations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  image_sha256 TEXT NOT NULL,
  username TEXT NOT NULL,
  stage_index INTEGER NOT NULL,
  option_value TEXT NOT NULL,
  annotated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(image_sha256, username, stage_index),
  FOREIGN KEY(image_sha256) REFERENCES images(sha256) ON DELETE CASCADE
);

CREATE INDEX idx_annotations_image_sha256 ON annotations(image_sha256);
CREATE INDEX idx_annotations_username ON annotations(username);
CREATE INDEX idx_annotations_stage ON annotations(stage_index);
