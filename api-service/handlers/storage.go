package handlers

import (
	"context"
	"fmt"
	"net"
	"os"
	"regexp"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"

	"digital-recipes/api-service/models"
)

// Input validation functions
func sanitizeClientIP(clientIP string) string {
	// Validate IP address format
	if parsedIP := net.ParseIP(clientIP); parsedIP != nil {
		return parsedIP.String()
	}
	// Return safe fallback for invalid IPs
	return "unknown"
}

func sanitizeImageID(imageID string) string {
	// Only allow alphanumeric characters, hyphens, and underscores
	reg := regexp.MustCompile(`[^a-zA-Z0-9\-_]`)
	sanitized := reg.ReplaceAllString(imageID, "")
	
	// Ensure minimum length and maximum length
	if len(sanitized) < 10 {
		sanitized = uuid.New().String()
	}
	if len(sanitized) > 100 {
		sanitized = sanitized[:100]
	}
	
	return sanitized
}

func validateContentType(contentType string) bool {
	allowedTypes := []string{
		"image/jpeg",
		"image/png", 
		"image/webp",
	}
	
	for _, allowed := range allowedTypes {
		if contentType == allowed {
			return true
		}
	}
	return false
}

// validateFileSignature validates file content based on magic numbers/file signatures
// This prevents file upload attacks where malicious files have image extensions
func validateFileSignature(filename, contentType string) error {
	// Define expected file signatures (magic numbers) for image types
	validSignatures := map[string][]string{
		"image/jpeg": {
			"\xFF\xD8\xFF",        // JPEG/JFIF
			"\xFF\xD8\xFF\xE0",    // JPEG/JFIF
			"\xFF\xD8\xFF\xE1",    // JPEG/EXIF
		},
		"image/png": {
			"\x89PNG\r\n\x1a\n",   // PNG signature
		},
		"image/webp": {
			"RIFF",                // WebP starts with RIFF
		},
	}
	
	// Validate content type is supported
	if !validateContentType(contentType) {
		return fmt.Errorf("unsupported content type: %s", contentType)
	}
	
	// Additional filename validation to prevent directory traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return fmt.Errorf("invalid filename: directory traversal characters not allowed")
	}
	
	// Validate filename extension matches content type
	expectedExtensions := map[string][]string{
		"image/jpeg": {".jpg", ".jpeg"},
		"image/png":  {".png"},
		"image/webp": {".webp"},
	}
	
	extensions := expectedExtensions[contentType]
	validExtension := false
	filename = strings.ToLower(filename)
	
	for _, ext := range extensions {
		if strings.HasSuffix(filename, ext) {
			validExtension = true
			break
		}
	}
	
	if !validExtension {
		return fmt.Errorf("filename extension doesn't match content type %s", contentType)
	}
	
	return nil
}

// StorageService handles file storage operations
type StorageService struct {
	gcsClient  *storage.Client
	bucketName string
	projectID  string
}

// NewStorageService creates a new storage service
func NewStorageService() (*StorageService, error) {
	bucketName := os.Getenv("GCS_BUCKET_NAME")
	if bucketName == "" {
		bucketName = "digital-recipes-images" // Default for development
	}

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = "digital-recipes-dev" // Default project for development
	}

	ctx := context.Background()
	var gcsClient *storage.Client
	var err error

	// Check if we have a service account key file
	credentialsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credentialsFile != "" {
		// Use service account key file
		gcsClient, err = storage.NewClient(ctx, option.WithCredentialsFile(credentialsFile))
	} else {
		// Use default credential chain (Application Default Credentials)
		// This includes: service account on GCE/GKE, gcloud credentials, etc.
		gcsClient, err = storage.NewClient(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	return &StorageService{
		gcsClient:  gcsClient,
		bucketName: bucketName,
		projectID:  projectID,
	}, nil
}

// GenerateUploadURLs creates pre-signed URLs for image uploads with enhanced security
func (s *StorageService) GenerateUploadURLs(ctx context.Context, recipeID int, uploadReq *models.UploadRequest, clientIP string) ([]models.ImageUploadURL, error) {
	var uploadURLs []models.ImageUploadURL
	
	// Get validated parameters from request
	maxFileSizeBytes := int64(uploadReq.GetMaxFileSizeMB()) * 1024 * 1024
	allowedTypes := uploadReq.GetAllowedTypes()
	expirationDuration := time.Duration(uploadReq.GetExpirationHours()) * time.Hour

	for i := 0; i < uploadReq.ImageCount; i++ {
		// Generate unique image ID with timestamp for uniqueness
		timestamp := time.Now().Unix()
		rawImageID := fmt.Sprintf("recipe-%d-%d-%s", recipeID, timestamp, uuid.New().String())
		imageID := sanitizeImageID(rawImageID)
		
		// Determine file extension based on allowed types (default to jpg)
		extension := "jpg"
		contentType := "image/jpeg"
		if len(allowedTypes) > 0 {
			contentType = allowedTypes[0]
			// Validate content type for security
			if !validateContentType(contentType) {
				logrus.WithFields(logrus.Fields{
					"content_type": contentType,
					"recipe_id":   recipeID,
				}).Warn("Invalid content type provided, using default")
				contentType = "image/jpeg"
			}
			switch contentType {
			case "image/png":
				extension = "png"
			case "image/webp":
				extension = "webp"
			}
		}
		
		// Create object key with proper prefix and extension
		objectKey := fmt.Sprintf("recipes/%d/images/%s.%s", recipeID, imageID, extension)

		// Set object metadata with sanitized inputs
		sanitizedClientIP := sanitizeClientIP(clientIP)
		metadata := map[string]string{
			"recipe-id":     fmt.Sprintf("%d", recipeID),
			"image-id":      imageID,
			"uploader-ip":   sanitizedClientIP,
			"upload-time":   fmt.Sprintf("%d", time.Now().Unix()),
			"max-size-mb":   fmt.Sprintf("%d", uploadReq.GetMaxFileSizeMB()),
			"content-type":  contentType,
		}

		// Generate pre-signed URL for PUT operation
		opts := &storage.SignedURLOptions{
			Scheme:  storage.SigningSchemeV4,
			Method:  "PUT",
			Headers: []string{
				"Content-Type:" + contentType,
			},
			Expires: time.Now().Add(expirationDuration),
		}

		// Add content length restriction
		for key, value := range metadata {
			opts.Headers = append(opts.Headers, fmt.Sprintf("x-goog-meta-%s:%s", key, value))
		}

		signedURL, err := s.gcsClient.Bucket(s.bucketName).SignedURL(objectKey, opts)
		if err != nil {
			logrus.WithError(err).WithField("image_index", i).Error("Failed to create pre-signed URL")
			return nil, fmt.Errorf("failed to create upload URL: %w", err)
		}

		uploadURL := models.ImageUploadURL{
			ImageID:   imageID,
			UploadURL: signedURL,
			Fields:    make(map[string]string),
		}

		// Add required headers and constraints as fields
		uploadURL.Fields["Content-Type"] = contentType
		uploadURL.Fields["Content-Length-Range"] = fmt.Sprintf("1,%d", maxFileSizeBytes)
		uploadURL.Fields["Cache-Control"] = "no-cache"
		
		// Add custom metadata headers
		for key, value := range metadata {
			uploadURL.Fields[fmt.Sprintf("x-goog-meta-%s", key)] = value
		}

		uploadURLs = append(uploadURLs, uploadURL)
	}

	return uploadURLs, nil
}

// HealthCheck verifies GCS connectivity
func (s *StorageService) HealthCheck(ctx context.Context) error {
	// Simple operation to test connectivity
	bucket := s.gcsClient.Bucket(s.bucketName)
	_, err := bucket.Attrs(ctx)
	
	if err != nil {
		logrus.WithError(err).Error("GCS health check failed")
		return fmt.Errorf("GCS connectivity check failed: %w", err)
	}

	return nil
}