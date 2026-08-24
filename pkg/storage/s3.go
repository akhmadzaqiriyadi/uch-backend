package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	"gozaq/config"
)

// S3ClientAPI allows mocking S3 Client operations for testing
type S3ClientAPI interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

// S3Storage implements Storage for AWS S3, Cloudflare R2, MinIO, and DigitalOcean Spaces
type S3Storage struct {
	client       S3ClientAPI
	bucket       string
	region       string
	publicURL    string
	maxSize      int64
	allowedMimes map[string]bool
}

func NewS3Storage(cfg *config.Config) (*S3Storage, error) {
	if cfg.Storage.S3Bucket == "" {
		return nil, errors.New("S3_BUCKET is required when using S3 storage driver")
	}

	var optFns []func(*awsConfig.LoadOptions) error
	optFns = append(optFns, awsConfig.WithRegion(cfg.Storage.S3Region))

	if cfg.Storage.S3AccessKey != "" && cfg.Storage.S3SecretKey != "" {
		optFns = append(optFns, awsConfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.Storage.S3AccessKey, cfg.Storage.S3SecretKey, ""),
		))
	}

	awsCfg, err := awsConfig.LoadDefaultConfig(context.Background(), optFns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS S3 configuration: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Storage.S3Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Storage.S3Endpoint)
			o.UsePathStyle = true // Required for MinIO and self-hosted S3
		}
	})

	allowedMimes := map[string]bool{
		"image/jpeg":       true,
		"image/png":        true,
		"image/webp":       true,
		"image/gif":        true,
		"application/pdf":  true,
		"application/json": true,
		"text/plain":       true,
	}

	return &S3Storage{
		client:       client,
		bucket:       cfg.Storage.S3Bucket,
		region:       cfg.Storage.S3Region,
		publicURL:    strings.TrimRight(cfg.Storage.S3PublicURL, "/"),
		maxSize:      cfg.Storage.MaxFileSize,
		allowedMimes: allowedMimes,
	}, nil
}

// NewS3StorageWithClient allows injecting a mock or pre-configured S3 client
func NewS3StorageWithClient(client S3ClientAPI, bucket, region, publicURL string, maxSize int64) *S3Storage {
	return &S3Storage{
		client:    client,
		bucket:    bucket,
		region:    region,
		publicURL: strings.TrimRight(publicURL, "/"),
		maxSize:   maxSize,
		allowedMimes: map[string]bool{
			"image/jpeg":       true,
			"image/png":        true,
			"image/webp":       true,
			"image/gif":        true,
			"application/pdf":  true,
			"application/json": true,
			"text/plain":       true,
		},
	}
}

func (s *S3Storage) SaveFile(header *multipart.FileHeader) (*FileInfo, error) {
	if header.Size > s.maxSize {
		return nil, ErrFileTooLarge
	}

	file, err := header.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Sniff MIME type from first 512 bytes
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read file header: %w", err)
	}
	mimeType := http.DetectContentType(buffer[:n])

	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}

	if !s.allowedMimes[mimeType] {
		return nil, fmt.Errorf("%w: %s", ErrInvalidFileType, mimeType)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to reset file pointer: %w", err)
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".bin"
	}
	uniqueKey := fmt.Sprintf("%s%s", uuid.New().String(), strings.ToLower(ext))

	ctx := context.Background()
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(uniqueKey),
		Body:          file,
		ContentType:   aws.String(mimeType),
		ContentLength: aws.Int64(header.Size),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload object to S3: %w", err)
	}

	return &FileInfo{
		OriginalName: header.Filename,
		FileName:     uniqueKey,
		FilePath:     fmt.Sprintf("s3://%s/%s", s.bucket, uniqueKey),
		URL:          s.GetURL(uniqueKey),
		Size:         header.Size,
		MimeType:     mimeType,
	}, nil
}

func (s *S3Storage) DeleteFile(filename string) error {
	cleanName := filepath.Base(filename)
	ctx := context.Background()

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(cleanName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object from S3: %w", err)
	}
	return nil
}

func (s *S3Storage) GetURL(filename string) string {
	cleanName := filepath.Base(filename)
	if s.publicURL != "" {
		return fmt.Sprintf("%s/%s", s.publicURL, cleanName)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, cleanName)
}
