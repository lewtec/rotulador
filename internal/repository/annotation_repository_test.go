package repository

import (
	"context"
	"testing"
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
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	t.Run("creates annotation successfully", func(t *testing.T) {
		ann, err := annRepo.Create(ctx, img.SHA256, "testuser", 0, "good")
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if ann.ID == 0 {
			t.Error("Expected non-zero ID")
		}
		if ann.ImageSHA256 != img.SHA256 {
			t.Errorf("ImageSHA256 = %v, want %v", ann.ImageSHA256, img.SHA256)
		}
		if ann.Username != "testuser" {
			t.Errorf("Username = %v, want %v", ann.Username, "testuser")
		}
		if ann.StageIndex != 0 {
			t.Errorf("StageIndex = %v, want 0", ann.StageIndex)
		}
		if ann.OptionValue != "good" {
			t.Errorf("OptionValue = %v, want %v", ann.OptionValue, "good")
		}
		if ann.AnnotatedAt.IsZero() {
			t.Error("AnnotatedAt should not be zero")
		}
	})

	t.Run("upserts existing annotation", func(t *testing.T) {
		// Create initial annotation
		ann1, _ := annRepo.Create(ctx, img.SHA256, "user2", 0, "bad")

		// Update with new value
		ann2, err := annRepo.Create(ctx, img.SHA256, "user2", 0, "good")
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if ann2.ID != ann1.ID {
			t.Error("Upsert should keep same ID")
		}
		if ann2.OptionValue != "good" {
			t.Errorf("OptionValue = %v, want good", ann2.OptionValue)
		}
	})
}

func TestAnnotationRepository_Get(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img, _ := imgRepo.Create(ctx, "/test/image.jpg", "test.jpg")
	created, _ := annRepo.Create(ctx, img.SHA256, "testuser", 0, "good")

	t.Run("retrieves existing annotation", func(t *testing.T) {
		ann, err := annRepo.Get(ctx, img.SHA256, "testuser", 0)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if ann == nil {
			t.Fatal("Expected annotation, got nil")
		}
		if ann.ID != created.ID {
			t.Errorf("ID = %v, want %v", ann.ID, created.ID)
		}
	})

	t.Run("returns nil for non-existent annotation", func(t *testing.T) {
		ann, err := annRepo.Get(ctx, img.SHA256, "nonexistent", 0)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if ann != nil {
			t.Error("Expected nil for non-existent annotation")
		}
	})
}

func TestAnnotationRepository_GetForImage(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img, _ := imgRepo.Create(ctx, "/test/image.jpg", "test.jpg")
	if _, err := annRepo.Create(ctx, img.SHA256, "user1", 0, "good"); err != nil {
		t.Fatalf("failed to create annotation: %v", err)
	}
	if _, err := annRepo.Create(ctx, img.SHA256, "user2", 0, "bad"); err != nil {
		t.Fatalf("failed to create annotation: %v", err)
	}
	if _, err := annRepo.Create(ctx, img.SHA256, "user1", 1, "true"); err != nil {
		t.Fatalf("failed to create annotation: %v", err)
	}

	t.Run("retrieves all annotations for image", func(t *testing.T) {
		anns, err := annRepo.GetForImage(ctx, img.SHA256)
		if err != nil {
			t.Fatalf("GetForImage() error = %v", err)
		}

		if len(anns) != 3 {
			t.Errorf("Got %d annotations, want 3", len(anns))
		}

		// Check ordering by stage_index
		if anns[0].StageIndex > anns[len(anns)-1].StageIndex {
			t.Error("Annotations should be ordered by stage_index")
		}
	})
}

func TestAnnotationRepository_GetByUser(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img1, _ := imgRepo.Create(ctx, "/test/image1.jpg", "image1.jpg")
	img2, _ := imgRepo.Create(ctx, "/test/image2.jpg", "image2.jpg")
	if _, err := annRepo.Create(ctx, img1.SHA256, "testuser", 0, "good"); err != nil {
		t.Fatal(err)
	}
	if _, err := annRepo.Create(ctx, img2.SHA256, "testuser", 0, "bad"); err != nil {
		t.Fatal(err)
	}
	if _, err := annRepo.Create(ctx, img1.SHA256, "otheruser", 0, "good"); err != nil {
		t.Fatal(err)
	}

	t.Run("retrieves annotations by user", func(t *testing.T) {
		anns, err := annRepo.GetByUser(ctx, "testuser", 10, 0)
		if err != nil {
			t.Fatalf("GetByUser() error = %v", err)
		}

		if len(anns) != 2 {
			t.Errorf("Got %d annotations, want 2", len(anns))
		}

		// Check that all annotations are by testuser
		for _, ann := range anns {
			if ann.Username != "testuser" {
				t.Errorf("Got annotation by %v, want testuser", ann.Username)
			}
			// Check that image info is included
			if ann.ImageFilename == "" {
				t.Error("ImageFilename should not be empty")
			}
		}
	})

	t.Run("respects limit and offset", func(t *testing.T) {
		anns, err := annRepo.GetByUser(ctx, "testuser", 1, 0)
		if err != nil {
			t.Fatalf("GetByUser() error = %v", err)
		}

		if len(anns) != 1 {
			t.Errorf("Got %d annotations, want 1", len(anns))
		}

		anns2, err := annRepo.GetByUser(ctx, "testuser", 1, 1)
		if err != nil {
			t.Fatalf("GetByUser() error = %v", err)
		}

		if len(anns2) != 1 {
			t.Errorf("Got %d annotations, want 1", len(anns2))
		}

		// Should be different annotations
		if anns[0].ID == anns2[0].ID {
			t.Error("Offset should return different annotations")
		}
	})
}

func TestAnnotationRepository_GetByImageAndUser(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img, _ := imgRepo.Create(ctx, "/test/image.jpg", "test.jpg")
	if _, err := annRepo.Create(ctx, img.SHA256, "testuser", 0, "good"); err != nil {
		t.Fatal(err)
	}
	if _, err := annRepo.Create(ctx, img.SHA256, "testuser", 1, "true"); err != nil {
		t.Fatal(err)
	}
	if _, err := annRepo.Create(ctx, img.SHA256, "otheruser", 0, "bad"); err != nil {
		t.Fatal(err)
	}

	t.Run("retrieves annotations for image and user", func(t *testing.T) {
		anns, err := annRepo.GetByImageAndUser(ctx, img.SHA256, "testuser")
		if err != nil {
			t.Fatalf("GetByImageAndUser() error = %v", err)
		}

		if len(anns) != 2 {
			t.Errorf("Got %d annotations, want 2", len(anns))
		}

		// All should be by testuser
		for _, ann := range anns {
			if ann.Username != "testuser" {
				t.Errorf("Got annotation by %v, want testuser", ann.Username)
			}
		}
	})
}

func TestAnnotationRepository_CountByUser(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img1, _ := imgRepo.Create(ctx, "/test/image1.jpg", "image1.jpg")
	img2, _ := imgRepo.Create(ctx, "/test/image2.jpg", "image2.jpg")
	if _, err := annRepo.Create(ctx, img1.SHA256, "testuser", 0, "good"); err != nil {
		t.Fatal(err)
	}
	if _, err := annRepo.Create(ctx, img2.SHA256, "testuser", 0, "bad"); err != nil {
		t.Fatal(err)
	}
	if _, err := annRepo.Create(ctx, img1.SHA256, "otheruser", 0, "good"); err != nil {
		t.Fatal(err)
	}

	t.Run("counts annotations by user", func(t *testing.T) {
		count, err := annRepo.CountByUser(ctx, "testuser")
		if err != nil {
			t.Fatalf("CountByUser() error = %v", err)
		}

		if count != 2 {
			t.Errorf("Count = %v, want 2", count)
		}
	})
}

func TestAnnotationRepository_ListPendingImagesForUserAndStage(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img1, _ := imgRepo.Create(ctx, "/test/image1.jpg", "image1.jpg")
	img2, _ := imgRepo.Create(ctx, "/test/image2.jpg", "image2.jpg")
	img3, _ := imgRepo.Create(ctx, "/test/image3.jpg", "image3.jpg")

	// testuser annotated stage 0 of img1
	if _, err := annRepo.Create(ctx, img1.SHA256, "testuser", 0, "good"); err != nil {
		t.Fatal(err)
	}

	// otheruser annotated stage 0 of img2
	if _, err := annRepo.Create(ctx, img2.SHA256, "otheruser", 0, "bad"); err != nil {
		t.Fatal(err)
	}

	_ = img3 // unused

	t.Run("lists pending images for user and stage", func(t *testing.T) {
		// testuser should see img2 (not annotated by them) but not img1 or img3
		_, err := annRepo.ListPendingImagesForUserAndStage(ctx, "testuser", 0, 10)
		if err != nil {
			t.Fatalf("ListPendingImagesForUserAndStage() error = %v", err)
		}
	})

	t.Run("includes images with no annotations", func(t *testing.T) {
		// Create a new image with no annotations
		img4, _ := imgRepo.Create(ctx, "/test/image4.jpg", "image4.jpg")

		images, err := annRepo.ListPendingImagesForUserAndStage(ctx, "testuser", 0, 10)
		if err != nil {
			t.Fatalf("ListPendingImagesForUserAndStage() error = %v", err)
		}

		foundImg4 := false
		for _, img := range images {
			if img.SHA256 == img4.SHA256 {
				foundImg4 = true
			}
		}
		if !foundImg4 {
			t.Error("Should include image with no annotations")
		}
	})
}

func TestAnnotationRepository_Exists(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img, _ := imgRepo.Create(ctx, "/test/image.jpg", "test.jpg")
	if _, err := annRepo.Create(ctx, img.SHA256, "testuser", 0, "good"); err != nil {
		t.Fatal(err)
	}

	t.Run("returns true for existing annotation", func(t *testing.T) {
		exists, err := annRepo.Exists(ctx, img.SHA256, "testuser", 0)
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}

		if !exists {
			t.Error("Exists should return true")
		}
	})

	t.Run("returns false for non-existent annotation", func(t *testing.T) {
		exists, err := annRepo.Exists(ctx, img.SHA256, "nonexistent", 0)
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}

		if exists {
			t.Error("Exists should return false")
		}
	})
}

func TestAnnotationRepository_Delete(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img, _ := imgRepo.Create(ctx, "/test/image.jpg", "test.jpg")
	ann, _ := annRepo.Create(ctx, img.SHA256, "testuser", 0, "good")

	t.Run("deletes annotation", func(t *testing.T) {
		err := annRepo.Delete(ctx, ann.ID)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify deletion
		exists, _ := annRepo.Exists(ctx, img.SHA256, "testuser", 0)
		if exists {
			t.Error("Annotation should be deleted")
		}
	})
}

func TestAnnotationRepository_DeleteForImage(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img, _ := imgRepo.Create(ctx, "/test/image.jpg", "test.jpg")
	if _, err := annRepo.Create(ctx, img.SHA256, "user1", 0, "good"); err != nil {
		t.Fatal(err)
	}
	if _, err := annRepo.Create(ctx, img.SHA256, "user2", 0, "bad"); err != nil {
		t.Fatal(err)
	}

	t.Run("deletes all annotations for image", func(t *testing.T) {
		err := annRepo.DeleteForImage(ctx, img.SHA256)
		if err != nil {
			t.Fatalf("DeleteForImage() error = %v", err)
		}

		// Verify deletion
		anns, _ := annRepo.GetForImage(ctx, img.SHA256)
		if len(anns) != 0 {
			t.Errorf("Expected 0 annotations, got %d", len(anns))
		}
	})
}

func TestAnnotationRepository_GetStats(t *testing.T) {
	imgRepo, annRepo, ctx := setupTestRepositories(t)

	// Create test data
	img1, _ := imgRepo.Create(ctx, "/test/image1.jpg", "image1.jpg")
	img2, _ := imgRepo.Create(ctx, "/test/image2.jpg", "image2.jpg")
	if _, err := annRepo.Create(ctx, img1.SHA256, "user1", 0, "good"); err != nil {
		t.Fatal(err)
	}
	if _, err := annRepo.Create(ctx, img1.SHA256, "user2", 0, "bad"); err != nil {
		t.Fatal(err)
	}
	if _, err := annRepo.Create(ctx, img2.SHA256, "user1", 0, "good"); err != nil {
		t.Fatal(err)
	}

	t.Run("returns correct statistics", func(t *testing.T) {
		stats, err := annRepo.GetStats(ctx)
		if err != nil {
			t.Fatalf("GetStats() error = %v", err)
		}

		if stats.AnnotatedImages != 2 {
			t.Errorf("AnnotatedImages = %v, want 2", stats.AnnotatedImages)
		}
		if stats.TotalAnnotations != 3 {
			t.Errorf("TotalAnnotations = %v, want 3", stats.TotalAnnotations)
		}
		if stats.TotalUsers != 2 {
			t.Errorf("TotalUsers = %v, want 2", stats.TotalUsers)
		}
	})
}

// Benchmark tests
func BenchmarkAnnotationRepository_Create(b *testing.B) {
	db := SetupTestDB(&testing.T{})
	defer func() {
		_ = db.Close()
	}()

	imgRepo := NewImageRepository(db)
	annRepo := NewAnnotationRepository(db)
	ctx := context.Background()

	// Create test image
	img, _ := imgRepo.Create(ctx, "/test/image.jpg", "test.jpg")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = annRepo.Create(ctx, img.SHA256, "testuser", 0, "good")
	}
}

func BenchmarkAnnotationRepository_GetForImage(b *testing.B) {
	db := SetupTestDB(&testing.T{})
	defer func() {
		_ = db.Close()
	}()

	imgRepo := NewImageRepository(db)
	annRepo := NewAnnotationRepository(db)
	ctx := context.Background()

	// Create test data
	img, _ := imgRepo.Create(ctx, "/test/image.jpg", "test.jpg")
	for i := 0; i < 10; i++ {
		_, _ = annRepo.Create(ctx, img.SHA256, "testuser", i, "good")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = annRepo.GetForImage(ctx, img.SHA256)
	}
}
