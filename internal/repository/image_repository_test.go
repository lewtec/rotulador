package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageRepository_Create(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewImageRepository(db)
	ctx := context.Background()

	t.Run("creates image successfully", func(t *testing.T) {
		img, err := repo.Create(ctx, "abcdef1234567890", "image.jpg")
		require.NoError(t, err, "Create() error")

		assert.Equal(t, "abcdef1234567890", img.SHA256, "SHA256")
		assert.Equal(t, "image.jpg", img.Filename, "Filename")
		assert.False(t, img.IngestedAt.IsZero(), "IngestedAt should not be zero")
	})
}

func TestImageRepository_GetBySHA256(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewImageRepository(db)
	ctx := context.Background()

	// Create test image
	created, err := repo.Create(ctx, "abcdef1234567890", "test.jpg")
	require.NoError(t, err, "Failed to create test image")

	t.Run("retrieves existing image", func(t *testing.T) {
		img, err := repo.GetBySHA256(ctx, created.SHA256)
		require.NoError(t, err, "GetBySHA256() error")

		require.NotNil(t, img, "Expected image, got nil")
		assert.Equal(t, created.SHA256, img.SHA256, "SHA256")
		assert.Equal(t, created.Filename, img.Filename, "Filename")
	})

	t.Run("returns nil for non-existent image", func(t *testing.T) {
		img, err := repo.GetBySHA256(ctx, "nonexistent")
		require.NoError(t, err, "GetBySHA256() error")
		assert.Nil(t, img, "Expected nil for non-existent image")
	})
}

func TestImageRepository_GetByFilename(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewImageRepository(db)
	ctx := context.Background()

	// Create test image
	created, err := repo.Create(ctx, "abcdef1234567890", "test.jpg")
	require.NoError(t, err, "Failed to create test image")

	t.Run("retrieves existing image by filename", func(t *testing.T) {
		img, err := repo.GetByFilename(ctx, "test.jpg")
		require.NoError(t, err, "GetByFilename() error")

		require.NotNil(t, img, "Expected image, got nil")
		assert.Equal(t, created.SHA256, img.SHA256, "SHA256")
	})

	t.Run("returns nil for non-existent filename", func(t *testing.T) {
		img, err := repo.GetByFilename(ctx, "nonexistent.jpg")
		require.NoError(t, err, "GetByFilename() error")
		assert.Nil(t, img, "Expected nil for non-existent filename")
	})
}

func TestImageRepository_List(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewImageRepository(db)
	ctx := context.Background()

	// Create test images
	_, err := repo.Create(ctx, "hash1", "image1.jpg")
	require.NoError(t, err, "Failed to create image1")
	_, err = repo.Create(ctx, "hash2", "image2.jpg")
	require.NoError(t, err, "Failed to create image2")

	t.Run("lists all images", func(t *testing.T) {
		images, err := repo.List(ctx)
		require.NoError(t, err, "List() error")
		assert.Len(t, images, 2, "Got %d images, want 2", len(images))
	})
}

func TestImageRepository_Count(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewImageRepository(db)
	ctx := context.Background()

	t.Run("counts all images", func(t *testing.T) {
		// Create test images
		repo.Create(ctx, "hash1", "image1.jpg")
		repo.Create(ctx, "hash2", "image2.jpg")
		repo.Create(ctx, "hash3", "image3.jpg")

		count, err := repo.Count(ctx)
		require.NoError(t, err, "Count() error")
		assert.Equal(t, int64(3), count, "Count = %v, want 3", count)
	})
}

func TestImageRepository_Delete(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewImageRepository(db)
	ctx := context.Background()

	// Create test image
	img, _ := repo.Create(ctx, "hash1", "test.jpg")

	t.Run("deletes image", func(t *testing.T) {
		err := repo.Delete(ctx, img.SHA256)
		require.NoError(t, err, "Delete() error")

		// Verify deletion
		deleted, err := repo.GetBySHA256(ctx, img.SHA256)
		require.NoError(t, err, "GetBySHA256() error")
		assert.Nil(t, deleted, "Image should be deleted")
	})
}

// Benchmark tests
func BenchmarkImageRepository_Create(b *testing.B) {
	db := SetupTestDB(&testing.T{})
	defer db.Close()

	repo := NewImageRepository(db)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.Create(ctx, "hash", "test.jpg")
		// Clean up to avoid duplicates
		db.Exec("DELETE FROM images")
	}
}

func BenchmarkImageRepository_GetBySHA256(b *testing.B) {
	db := SetupTestDB(&testing.T{})
	defer db.Close()

	repo := NewImageRepository(db)
	ctx := context.Background()

	// Create test image
	img, _ := repo.Create(ctx, "hash", "test.jpg")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.GetBySHA256(ctx, img.SHA256)
	}
}
