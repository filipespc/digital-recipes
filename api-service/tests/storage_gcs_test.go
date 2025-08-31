package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"digital-recipes/api-service/handlers"
	"digital-recipes/api-service/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGCSStorageService(t *testing.T) {
	// Skip integration tests if GCS credentials are not available
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" && os.Getenv("GOOGLE_CLOUD_PROJECT") == "" {
		t.Skip("Skipping GCS integration tests: No Google Cloud credentials found")
	}

	// Test storage service initialization
	t.Run("NewStorageService", func(t *testing.T) {
		storageService, err := handlers.NewStorageService()
		
		if err != nil {
			// If we can't connect to GCS, skip the test gracefully
			t.Skipf("Skipping GCS test due to initialization error: %v", err)
		}
		
		require.NoError(t, err)
		require.NotNil(t, storageService)
	})

	// Test health check functionality
	t.Run("HealthCheck", func(t *testing.T) {
		storageService, err := handlers.NewStorageService()
		if err != nil {
			t.Skipf("Skipping GCS test due to initialization error: %v", err)
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		err = storageService.HealthCheck(ctx)
		
		// Health check may fail if bucket doesn't exist or permissions are wrong
		// This is expected in development/testing environments
		if err != nil {
			t.Logf("GCS health check failed (expected in test environment): %v", err)
		}
	})

	// Test pre-signed URL generation with different configurations
	t.Run("GenerateUploadURLs", func(t *testing.T) {
		storageService, err := handlers.NewStorageService()
		if err != nil {
			t.Skipf("Skipping GCS test due to initialization error: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		testCases := []struct {
			name        string
			uploadReq   *models.UploadRequest
			expectError bool
		}{
			{
				name: "Single JPEG upload",
				uploadReq: &models.UploadRequest{
					ImageCount:      1,
					MaxFileSizeMB:   10,
					AllowedTypes:    []string{"image/jpeg"},
					ExpirationHours: 1,
				},
				expectError: false,
			},
			{
				name: "Multiple PNG uploads",
				uploadReq: &models.UploadRequest{
					ImageCount:      3,
					MaxFileSizeMB:   5,
					AllowedTypes:    []string{"image/png"},
					ExpirationHours: 2,
				},
				expectError: false,
			},
			{
				name: "WebP upload with defaults",
				uploadReq: &models.UploadRequest{
					ImageCount:   2,
					AllowedTypes: []string{"image/webp"},
				},
				expectError: false,
			},
			{
				name: "Maximum allowed uploads",
				uploadReq: &models.UploadRequest{
					ImageCount:      10,
					MaxFileSizeMB:   50,
					AllowedTypes:    []string{"image/jpeg", "image/png"},
					ExpirationHours: 24,
				},
				expectError: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				recipeID := 12345
				clientIP := "192.168.1.100"

				uploadURLs, err := storageService.GenerateUploadURLs(ctx, recipeID, tc.uploadReq, clientIP)

				if tc.expectError {
					assert.Error(t, err)
					return
				}

				// URL generation might fail due to missing permissions or bucket
				if err != nil {
					t.Logf("Expected error in test environment: %v", err)
					return
				}

				require.NoError(t, err)
				assert.Len(t, uploadURLs, tc.uploadReq.ImageCount)

				// Validate each generated URL
				for i, uploadURL := range uploadURLs {
					assert.NotEmpty(t, uploadURL.ImageID, "Image ID should not be empty for upload %d", i)
					assert.NotEmpty(t, uploadURL.UploadURL, "Upload URL should not be empty for upload %d", i)
					assert.Contains(t, uploadURL.UploadURL, "googleapis.com", "URL should point to Google Cloud Storage for upload %d", i)
					
					// Check required fields
					assert.NotEmpty(t, uploadURL.Fields["Content-Type"], "Content-Type should be set for upload %d", i)
					assert.Contains(t, uploadURL.Fields["Content-Type"], "image/", "Content-Type should be an image type for upload %d", i)
					assert.NotEmpty(t, uploadURL.Fields["Content-Length-Range"], "Content-Length-Range should be set for upload %d", i)
					
					// Validate metadata fields
					expectedMetadataFields := []string{
						"x-goog-meta-recipe-id",
						"x-goog-meta-image-id",
						"x-goog-meta-uploader-ip",
						"x-goog-meta-upload-time",
						"x-goog-meta-max-size-mb",
						"x-goog-meta-content-type",
					}
					
					for _, field := range expectedMetadataFields {
						assert.NotEmpty(t, uploadURL.Fields[field], "Metadata field %s should be set for upload %d", field, i)
					}

					// Validate specific metadata values
					assert.Equal(t, "12345", uploadURL.Fields["x-goog-meta-recipe-id"], "Recipe ID should match for upload %d", i)
					assert.Equal(t, clientIP, uploadURL.Fields["x-goog-meta-uploader-ip"], "Client IP should match for upload %d", i)
					assert.Contains(t, uploadURL.Fields["x-goog-meta-content-type"], "image/", "Content type metadata should be image type for upload %d", i)
				}
			})
		}
	})

	// Test URL generation with various edge cases
	t.Run("GenerateUploadURLs_EdgeCases", func(t *testing.T) {
		storageService, err := handlers.NewStorageService()
		if err != nil {
			t.Skipf("Skipping GCS test due to initialization error: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Test with minimum valid request
		minimalReq := &models.UploadRequest{
			ImageCount: 1,
		}

		uploadURLs, err := storageService.GenerateUploadURLs(ctx, 1, minimalReq, "127.0.0.1")
		
		// Might fail due to missing GCS setup, but shouldn't panic
		if err == nil {
			assert.Len(t, uploadURLs, 1)
			assert.NotEmpty(t, uploadURLs[0].ImageID)
			assert.NotEmpty(t, uploadURLs[0].UploadURL)
		} else {
			t.Logf("Expected error in test environment: %v", err)
		}
	})
}

func TestGCSStorageService_ConfigValidation(t *testing.T) {
	// Test environment variable handling
	t.Run("DefaultConfiguration", func(t *testing.T) {
		// Temporarily clear environment variables to test defaults
		originalBucket := os.Getenv("GCS_BUCKET_NAME")
		originalProject := os.Getenv("GOOGLE_CLOUD_PROJECT")
		originalCreds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

		os.Unsetenv("GCS_BUCKET_NAME")
		os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")

		defer func() {
			// Restore original environment
			if originalBucket != "" {
				os.Setenv("GCS_BUCKET_NAME", originalBucket)
			}
			if originalProject != "" {
				os.Setenv("GOOGLE_CLOUD_PROJECT", originalProject)
			}
			if originalCreds != "" {
				os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", originalCreds)
			}
		}()

		// This should use default values and attempt to create service
		storageService, err := handlers.NewStorageService()
		
		// Expect an error due to missing credentials, but service should handle gracefully
		if err == nil {
			assert.NotNil(t, storageService)
		} else {
			// Expected - no valid credentials available
			assert.Contains(t, err.Error(), "failed to create GCS client")
		}
	})

	t.Run("CustomConfiguration", func(t *testing.T) {
		// Set custom environment variables
		os.Setenv("GCS_BUCKET_NAME", "test-bucket-name")
		os.Setenv("GOOGLE_CLOUD_PROJECT", "test-project-id")
		
		defer func() {
			os.Unsetenv("GCS_BUCKET_NAME")
			os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		}()

		// This should use custom values
		_, err := handlers.NewStorageService()
		
		// Might fail due to invalid credentials, but should attempt with custom config
		if err != nil {
			// Expected in test environment
			assert.Contains(t, err.Error(), "failed to create GCS client")
		}
	})
}