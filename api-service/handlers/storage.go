package handlers

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	"digital-recipes/api-service/models"
)

// StorageService handles file storage operations
type StorageService struct {
	s3Client   *s3.Client
	bucketName string
	region     string
}

// NewStorageService creates a new storage service
func NewStorageService() (*StorageService, error) {
	bucketName := os.Getenv("S3_BUCKET_NAME")
	if bucketName == "" {
		bucketName = "digital-recipes-images" // Default for development
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1" // Default region
	}

	// Load AWS configuration
	var cfg aws.Config
	var err error

	// Check if we have explicit credentials (for development/testing)
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	if accessKey != "" && secretKey != "" {
		// Use explicit credentials
		cfg, err = config.LoadDefaultConfig(context.Background(),
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				accessKey, secretKey, os.Getenv("AWS_SESSION_TOKEN"),
			)),
		)
	} else {
		// Use default credential chain (IAM roles, etc.)
		cfg, err = config.LoadDefaultConfig(context.Background(),
			config.WithRegion(region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	return &StorageService{
		s3Client:   s3Client,
		bucketName: bucketName,
		region:     region,
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
		imageID := fmt.Sprintf("recipe-%d-%d-%s", recipeID, timestamp, uuid.New().String())
		
		// Determine file extension based on allowed types (default to jpg)
		extension := "jpg"
		contentType := "image/jpeg"
		if len(allowedTypes) > 0 {
			contentType = allowedTypes[0]
			switch contentType {
			case "image/png":
				extension = "png"
			case "image/webp":
				extension = "webp"
			}
		}
		
		// Create object key with proper prefix and extension
		objectKey := fmt.Sprintf("recipes/%d/images/%s.%s", recipeID, imageID, extension)

		// Create pre-signed PUT request with enhanced security
		presignClient := s3.NewPresignClient(s.s3Client)
		
		request, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket:        aws.String(s.bucketName),
			Key:           aws.String(objectKey),
			ContentType:   aws.String(contentType),
			ContentLength: aws.Int64(maxFileSizeBytes),
			Metadata: map[string]string{
				"recipe-id":     fmt.Sprintf("%d", recipeID),
				"image-id":      imageID,
				"uploader-ip":   clientIP,
				"upload-time":   fmt.Sprintf("%d", time.Now().Unix()),
				"max-size-mb":   fmt.Sprintf("%d", uploadReq.GetMaxFileSizeMB()),
				"content-type":  contentType,
			},
		}, func(opts *s3.PresignOptions) {
			opts.Expires = expirationDuration
		})

		if err != nil {
			log.Printf("Failed to create pre-signed URL for image %d: %v", i, err)
			return nil, fmt.Errorf("failed to create upload URL: %w", err)
		}

		uploadURL := models.ImageUploadURL{
			ImageID:   imageID,
			UploadURL: request.URL,
			Fields:    make(map[string]string),
		}

		// Add required headers and constraints as fields
		uploadURL.Fields["Content-Type"] = contentType
		uploadURL.Fields["Content-Length-Range"] = fmt.Sprintf("1,%d", maxFileSizeBytes)
		uploadURL.Fields["Cache-Control"] = "no-cache"

		uploadURLs = append(uploadURLs, uploadURL)
	}

	return uploadURLs, nil
}

// HealthCheck verifies S3 connectivity
func (s *StorageService) HealthCheck(ctx context.Context) error {
	// Simple operation to test connectivity
	_, err := s.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucketName),
	})
	
	if err != nil {
		log.Printf("S3 health check failed: %v", err)
		return fmt.Errorf("S3 connectivity check failed: %w", err)
	}

	return nil
}