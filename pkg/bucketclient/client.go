package bucketclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

type Config struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

type Client struct {
	s3Client *s3.Client
	bucket   string
}

func New(cfg Config) (*Client, error) {
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = true // Required for S3-compatible services like Cloudflare R2
	})

	return &Client{
		s3Client: s3Client,
		bucket:   cfg.Bucket,
	}, nil
}

func (c *Client) Upload(ctx context.Context, key string, reader io.Reader, contentType string) error {
	// For R2 compatibility, avoid checksum algorithms that can cause signature issues
	// Let R2 handle integrity checking on its end
	_, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
		// Remove ChecksumAlgorithm - can cause signature calculation issues with R2
		// R2 will validate integrity using its own mechanisms
	})
	if err != nil {
		// Enhanced error context for debugging R2 issues
		var ae smithy.APIError
		if errors.As(err, &ae) {
			return fmt.Errorf("failed to upload to bucket [%s]: %s - %w", ae.ErrorCode(), ae.ErrorMessage(), err)
		}
		return fmt.Errorf("failed to upload to bucket: %w", err)
	}
	return nil
}

// UploadFile uploads a file directly from filesystem with explicit content length
// This method is more R2-compatible as it provides seekable content with known size
func (c *Client) UploadFile(ctx context.Context, key string, filePath string, contentType string) error {
	// Open file and get size for explicit content length
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info for content length
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Use explicit content length for better R2 compatibility
	_, err = c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(key),
		Body:          file,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(fileInfo.Size()),
		// No checksum algorithm - let R2 handle integrity
	})
	if err != nil {
		// Enhanced error context for debugging R2 issues
		var ae smithy.APIError
		if errors.As(err, &ae) {
			return fmt.Errorf("failed to upload file to bucket [%s]: %s - %w", ae.ErrorCode(), ae.ErrorMessage(), err)
		}
		return fmt.Errorf("failed to upload file to bucket: %w", err)
	}
	return nil
}

