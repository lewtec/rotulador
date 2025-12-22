package repository

import (
	"context"
	"testing"
)

func TestImageRepository_Create(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewImageRepository(db)
	ctx := context.Background()

	t.Run("creates image successfully", func(t *testing.T) {
		img, err := repo.Create(ctx, "abcdef1234567890", "image.jpg")
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if img.SHA256 != "abcdef1234567890" {
			t.Errorf("SHA256 = %v, want %v", img.SHA256, "abcdef1234567890")
		}
		if img.Filename != "image.jpg" {
			t.Errorf("Filename = %v, want %v", img.Filename, "image.jpg")
		}
		if img.IngestedAt.IsZero() {
			t.Error("IngestedAt should not be zero")
		}
	})
}

func TestImageRepository_GetBySHA256(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewImageRepository(db)
	ctx := context.Background()

	// Create test image
	created, err := repo.Create(ctx, "abcdef1234567890", "test.jpg")
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	t.Run("retrieves existing image", func(t *testing.T) {
		img, err := repo.GetBySHA256(ctx, created.SHA256)
		if err != nil {
			t.Fatalf("GetBySHA256() error = %v", err)
		}

		if img == nil {
			t.Fatal("Expected image, got nil")
		}
		if img.SHA256 != created.SHA256 {
			t.Errorf("SHA256 = %v, want %v", img.SHA256, created.SHA256)
		}
		if img.Filename != created.Filename {
			t.Errorf("Filename = %v, want %v", img.Filename, created.Filename)
		}
	})

	t.Run("returns nil for non-existent image", func(t *testing.T) {
		img, err := repo.GetBySHA256(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("GetBySHA256() error = %v", err)
		}
		if img != nil {
			t.Error("Expected nil for non-existent image")
		}
	})
}

func TestImageRepository_GetByFilename(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewImageRepository(db)
	ctx := context.Background()

	// Create test image
	created, err := repo.Create(ctx, "abcdef1234567890", "test.jpg")
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	t.Run("retrieves existing image by filename", func(t *testing.T) {
		img, err := repo.GetByFilename(ctx, "test.jpg")
		if err != nil {
			t.Fatalf("GetByFilename() error = %v", err)
		}

		if img == nil {
			t.Fatal("Expected image, got nil")
		}
		if img.SHA256 != created.SHA256 {
			t.Errorf("SHA256 = %v, want %v", img.SHA256, created.SHA256)
		}
	})

	t.Run("returns nil for non-existent filename", func(t *testing.T) {
		img, err := repo.GetByFilename(ctx, "nonexistent.jpg")
		if err != nil {
			t.Fatalf("GetByFilename() error = %v", err)
		}
		if img != nil {
			t.Error("Expected nil for non-existent filename")
		}
	})
}

func TestImageRepository_List(t *testing.T) {
	db := SetupTestDB(t)
	defer CleanupTestDB(t, db)

	repo := NewImageRepository(db)
	ctx := context.Background()

	// Create test images
	_, err := repo.Create(ctx, "hash1", "image1.jpg")
	if err != nil {
		t.Fatalf("Failed to create image1: %v", err)
	}
	_, err = repo.Create(ctx, "hash2", "image2.jpg")
	if err != nil {
		t.Fatalf("Failed to create image2: %v", err)
	}

	t.Run("lists all images", func(t *testing.T) {
		images, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(images) != 2 {
			t.Errorf("Got %d images, want 2", len(images))
		}
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
		if err != nil {
			t.Fatalf("Count() error = %v", err)
		}

		if count != 3 {
			t.Errorf("Count = %v, want 3", count)
		}
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
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify deletion
		deleted, err := repo.GetBySHA256(ctx, img.SHA256)
		if err != nil {
			t.Fatalf("GetBySHA256() error = %v", err)
		}
		if deleted != nil {
			t.Error("Image should be deleted")
		}
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
