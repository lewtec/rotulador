package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRepositories(t *testing.T) (*ImageRepository, *AnnotationRepository, context.Context) {
	t.Helper()
	db := SetupTestDB(t)
	t.Cleanup(func() { CleanupTestDB(t, db) })

	return NewImageRepository(db), NewAnnotationRepository(db), context.Background()
}

func TestAnnotationRepository_Create(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test image
	img, err := imgRepo.Create(ctx, "/test/image.jpg", "test.jpg")
	require.NoError(t, err, "Failed to create test image")

	t.Run("creates annotation successfully", func(t *testing.T) {
		ann, err := annRepo.Create(ctx, img.SHA256, "testuser", 0, "good")
		require.NoError(t, err, "Create() error")

		assert.NotZero(t, ann.ID, "Expected non-zero ID")
		assert.Equal(t, img.SHA256, ann.ImageSHA256, "ImageSHA256")
		assert.Equal(t, "testuser", ann.Username, "Username")
		assert.Equal(t, 0, ann.StageIndex, "StageIndex")
		assert.Equal(t, "good", ann.OptionValue, "OptionValue")
		assert.False(t, ann.AnnotatedAt.IsZero(), "AnnotatedAt should not be zero")
	})

	t.Run("upserts existing annotation", func(t *testing.T) {
		// Create initial annotation
		ann1, _ := annRepo.Create(ctx, img.SHA256, "user2", 0, "bad")

		// Update with new value
		ann2, err := annRepo.Create(ctx, img.SHA256, "user2", 0, "good")
		require.NoError(t, err, "Create() error")

		assert.Equal(t, ann1.ID, ann2.ID, "Upsert should keep same ID")
		assert.Equal(t, "good", ann2.OptionValue, "OptionValue")
	})
}

func TestAnnotationRepository_Get(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img, _ := imgRepo.Create(ctx, "/test/image.jpg", "test.jpg")
	created, _ := annRepo.Create(ctx, img.SHA256, "testuser", 0, "good")

	t.Run("retrieves existing annotation", func(t *testing.T) {
		ann, err := annRepo.Get(ctx, img.SHA256, "testuser", 0)
		require.NoError(t, err, "Get() error")

		require.NotNil(t, ann, "Expected annotation, got nil")
		assert.Equal(t, created.ID, ann.ID, "ID")
	})

	t.Run("returns nil for non-existent annotation", func(t *testing.T) {
		ann, err := annRepo.Get(ctx, img.SHA256, "nonexistent", 0)
		require.NoError(t, err, "Get() error")
		assert.Nil(t, ann, "Expected nil for non-existent annotation")
	})
}

func TestAnnotationRepository_GetForImage(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img, _ := imgRepo.Create(ctx, "/test/image.jpg", "test.jpg")
	annRepo.Create(ctx, img.SHA256, "user1", 0, "good")
	annRepo.Create(ctx, img.SHA256, "user2", 0, "bad")
	annRepo.Create(ctx, img.SHA256, "user1", 1, "true")

	t.Run("retrieves all annotations for image", func(t *testing.T) {
		anns, err := annRepo.GetForImage(ctx, img.SHA256)
		require.NoError(t, err, "GetForImage() error")

		require.Len(t, anns, 3, "Got %d annotations, want 3", len(anns))

		// Check ordering by stage_index
		assert.LessOrEqual(t, anns[0].StageIndex, anns[len(anns)-1].StageIndex, "Annotations should be ordered by stage_index")
	})
}

func TestAnnotationRepository_GetByUser(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img1, _ := imgRepo.Create(ctx, "/test/image1.jpg", "image1.jpg")
	img2, _ := imgRepo.Create(ctx, "/test/image2.jpg", "image2.jpg")
	annRepo.Create(ctx, img1.SHA256, "testuser", 0, "good")
	annRepo.Create(ctx, img2.SHA256, "testuser", 0, "bad")
	annRepo.Create(ctx, img1.SHA256, "otheruser", 0, "good")

	t.Run("retrieves annotations by user", func(t *testing.T) {
		anns, err := annRepo.GetByUser(ctx, "testuser", 10, 0)
		require.NoError(t, err, "GetByUser() error")

		require.Len(t, anns, 2, "Got %d annotations, want 2", len(anns))

		// Check that all annotations are by testuser
		for _, ann := range anns {
			assert.Equal(t, "testuser", ann.Username, "Got annotation by %v, want testuser", ann.Username)
			// Check that image info is included
			assert.NotEmpty(t, ann.ImageFilename, "ImageFilename should not be empty")
		}
	})

	t.Run("respects limit and offset", func(t *testing.T) {
		anns, err := annRepo.GetByUser(ctx, "testuser", 1, 0)
		require.NoError(t, err, "GetByUser() error")
		require.Len(t, anns, 1, "Got %d annotations, want 1", len(anns))

		anns2, err := annRepo.GetByUser(ctx, "testuser", 1, 1)
		require.NoError(t, err, "GetByUser() error")
		require.Len(t, anns2, 1, "Got %d annotations, want 1", len(anns2))

		// Should be different annotations
		assert.NotEqual(t, anns[0].ID, anns2[0].ID, "Offset should return different annotations")
	})
}

func TestAnnotationRepository_GetByImageAndUser(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img, _ := imgRepo.Create(ctx, "/test/image.jpg", "test.jpg")
	annRepo.Create(ctx, img.SHA256, "testuser", 0, "good")
	annRepo.Create(ctx, img.SHA256, "testuser", 1, "true")
	annRepo.Create(ctx, img.SHA256, "otheruser", 0, "bad")

	t.Run("retrieves annotations for image and user", func(t *testing.T) {
		anns, err := annRepo.GetByImageAndUser(ctx, img.SHA256, "testuser")
		require.NoError(t, err, "GetByImageAndUser() error")

		require.Len(t, anns, 2, "Got %d annotations, want 2", len(anns))

		// All should be by testuser
		for _, ann := range anns {
			assert.Equal(t, "testuser", ann.Username, "Got annotation by %v, want testuser", ann.Username)
		}
	})
}

func TestAnnotationRepository_CountByUser(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img1, _ := imgRepo.Create(ctx, "/test/image1.jpg", "image1.jpg")
	img2, _ := imgRepo.Create(ctx, "/test/image2.jpg", "image2.jpg")
	annRepo.Create(ctx, img1.SHA256, "testuser", 0, "good")
	annRepo.Create(ctx, img2.SHA256, "testuser", 0, "bad")
	annRepo.Create(ctx, img1.SHA256, "otheruser", 0, "good")

	t.Run("counts annotations by user", func(t *testing.T) {
		count, err := annRepo.CountByUser(ctx, "testuser")
		require.NoError(t, err, "CountByUser() error")
		assert.Equal(t, int64(2), count, "Count = %v, want 2", count)
	})
}

func TestAnnotationRepository_ListPendingImagesForUserAndStage(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img1, _ := imgRepo.Create(ctx, "/test/image1.jpg", "image1.jpg")
	img2, _ := imgRepo.Create(ctx, "/test/image2.jpg", "image2.jpg")
	img3, _ := imgRepo.Create(ctx, "/test/image3.jpg", "image3.jpg")

	// testuser annotated stage 0 of img1
	annRepo.Create(ctx, img1.SHA256, "testuser", 0, "good")

	// otheruser annotated stage 0 of img2
	annRepo.Create(ctx, img2.SHA256, "otheruser", 0, "bad")

	_ = img3 // unused

	t.Run("lists pending images for user and stage", func(t *testing.T) {
		// testuser should see img2 (not annotated by them) but not img1 or img3
		_, err := annRepo.ListPendingImagesForUserAndStage(ctx, "testuser", 0, 10)
		require.NoError(t, err, "ListPendingImagesForUserAndStage() error")
	})

	t.Run("includes images with no annotations", func(t *testing.T) {
		// Create a new image with no annotations
		img4, _ := imgRepo.Create(ctx, "/test/image4.jpg", "image4.jpg")

		images, err := annRepo.ListPendingImagesForUserAndStage(ctx, "testuser", 0, 10)
		require.NoError(t, err, "ListPendingImagesForUserAndStage() error")

		foundImg4 := false
		for _, img := range images {
			if img.SHA256 == img4.SHA256 {
				foundImg4 = true
			}
		}
		assert.True(t, foundImg4, "Should include image with no annotations")
	})
}

func TestAnnotationRepository_Exists(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img, _ := imgRepo.Create(ctx, "/test/image.jpg", "test.jpg")
	annRepo.Create(ctx, img.SHA256, "testuser", 0, "good")

	t.Run("returns true for existing annotation", func(t *testing.T) {
		exists, err := annRepo.Exists(ctx, img.SHA256, "testuser", 0)
		require.NoError(t, err, "Exists() error")
		assert.True(t, exists, "Exists should return true")
	})

	t.Run("returns false for non-existent annotation", func(t *testing.T) {
		exists, err := annRepo.Exists(ctx, img.SHA256, "nonexistent", 0)
		require.NoError(t, err, "Exists() error")
		assert.False(t, exists, "Exists should return false")
	})
}

func TestAnnotationRepository_Delete(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img, _ := imgRepo.Create(ctx, "/test/image.jpg", "test.jpg")
	ann, _ := annRepo.Create(ctx, img.SHA256, "testuser", 0, "good")

	t.Run("deletes annotation", func(t *testing.T) {
		err := annRepo.Delete(ctx, ann.ID)
		require.NoError(t, err, "Delete() error")

		// Verify deletion
		exists, _ := annRepo.Exists(ctx, img.SHA256, "testuser", 0)
		assert.False(t, exists, "Annotation should be deleted")
	})
}

func TestAnnotationRepository_DeleteForImage(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img, _ := imgRepo.Create(ctx, "/test/image.jpg", "test.jpg")
	annRepo.Create(ctx, img.SHA256, "user1", 0, "good")
	annRepo.Create(ctx, img.SHA256, "user2", 0, "bad")

	t.Run("deletes all annotations for image", func(t *testing.T) {
		err := annRepo.DeleteForImage(ctx, img.SHA256)
		require.NoError(t, err, "DeleteForImage() error")

		// Verify deletion
		anns, _ := annRepo.GetForImage(ctx, img.SHA256)
		assert.Empty(t, anns, "Expected 0 annotations")
	})
}

func TestAnnotationRepository_GetStats(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img1, _ := imgRepo.Create(ctx, "/test/image1.jpg", "image1.jpg")
	img2, _ := imgRepo.Create(ctx, "/test/image2.jpg", "image2.jpg")
	annRepo.Create(ctx, img1.SHA256, "user1", 0, "good")
	annRepo.Create(ctx, img1.SHA256, "user2", 0, "bad")
	annRepo.Create(ctx, img2.SHA256, "user1", 0, "good")

	t.Run("returns correct statistics", func(t *testing.T) {
		stats, err := annRepo.GetStats(ctx)
		require.NoError(t, err, "GetStats() error")

		assert.Equal(t, int64(2), stats.AnnotatedImages, "AnnotatedImages")
		assert.Equal(t, int64(3), stats.TotalAnnotations, "TotalAnnotations")
		assert.Equal(t, int64(2), stats.TotalUsers, "TotalUsers")
	})
}

// Benchmark tests
func BenchmarkAnnotationRepository_Create(b *testing.B) {
	db := SetupTestDB(&testing.T{})
	defer db.Close()

	imgRepo := NewImageRepository(db)
	annRepo := NewAnnotationRepository(db)
	ctx := context.Background()

	// Create test image
	img, _ := imgRepo.Create(ctx, "/test/image.jpg", "test.jpg")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		annRepo.Create(ctx, img.SHA256, "testuser", 0, "good")
	}
}

func BenchmarkAnnotationRepository_GetForImage(b *testing.B) {
	db := SetupTestDB(&testing.T{})
	defer db.Close()

	imgRepo := NewImageRepository(db)
	annRepo := NewAnnotationRepository(db)
	ctx := context.Background()

	// Create test data
	img, _ := imgRepo.Create(ctx, "/test/image.jpg", "test.jpg")
	for i := 0; i < 10; i++ {
		annRepo.Create(ctx, img.SHA256, "testuser", i, "good")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		annRepo.GetForImage(ctx, img.SHA256)
	}
}
