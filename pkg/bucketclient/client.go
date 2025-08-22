package bucketclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/minio/crc64nvme"
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

func New(ctx context.Context, cfg Config) (*Client, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx,
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

// calculateCRC64NVME calculates CRC64NVME checksum from io.Reader and returns base64 encoded string
func calculateCRC64NVME(reader io.Reader) (string, error) {
	hash := crc64nvme.New()
	_, err := io.Copy(hash, reader)
	if err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}
	checksum := hash.Sum64()
	// Convert to base64 as required by AWS API
	checksumBytes := make([]byte, 8)
	for i := range 8 {
		checksumBytes[7-i] = byte(checksum >> (8 * i))
	}
	return base64.StdEncoding.EncodeToString(checksumBytes), nil
}

// calculateCRC64NVMEFromFile calculates CRC64NVME checksum from file and returns base64 encoded string
func calculateCRC64NVMEFromFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer func() {
		_ = file.Close() // Ignore close error since checksum calculation is the primary concern
	}()
	return calculateCRC64NVME(file)
}

func (c *Client) Upload(ctx context.Context, key string, reader io.Reader, contentType string) error {
	// Calculate CRC64NVME checksum while reading the data
	var checksumReader io.Reader
	var checksumB64 string

	// Use TeeReader to calculate checksum while uploading
	hash := crc64nvme.New()
	checksumReader = io.TeeReader(reader, hash)

	// Read all data to calculate checksum
	data, err := io.ReadAll(checksumReader)
	if err != nil {
		return fmt.Errorf("failed to read data for upload: %w", err)
	}

	// Calculate final checksum
	checksum := hash.Sum64()
	checksumBytes := make([]byte, 8)
	for i := range 8 {
		checksumBytes[7-i] = byte(checksum >> (8 * i))
	}
	checksumB64 = base64.StdEncoding.EncodeToString(checksumBytes)

	_, err = c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:            aws.String(c.bucket),
		Key:               aws.String(key),
		Body:              bytes.NewReader(data),
		ContentType:       aws.String(contentType),
		ChecksumCRC64NVME: aws.String(checksumB64),
	})
	if err != nil {
		return fmt.Errorf("failed to upload key '%s' to bucket '%s': %w", key, c.bucket, err)
	}
	return nil
}

// UploadFile uploads a file directly from filesystem with explicit content length
// This method is more R2-compatible as it provides seekable content with known size
func (c *Client) UploadFile(ctx context.Context, key string, filePath string, contentType string) error {
	// Calculate CRC64NVME checksum from file
	checksumB64, err := calculateCRC64NVMEFromFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Open file and get size for explicit content length
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		_ = file.Close() // Ignore close error since file upload is the primary concern
	}()

	// Get file info for content length
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Upload with CRC64NVME checksum
	_, err = c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:            aws.String(c.bucket),
		Key:               aws.String(key),
		Body:              file,
		ContentType:       aws.String(contentType),
		ContentLength:     aws.Int64(fileInfo.Size()),
		ChecksumCRC64NVME: aws.String(checksumB64),
	})
	if err != nil {
		return fmt.Errorf("failed to upload file '%s' as key '%s' to bucket '%s': %w", filePath, key, c.bucket, err)
	}
	return nil
}
