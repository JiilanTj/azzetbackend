package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"codeberg.org/azzet/azzetbe/internal/config"
)

type R2Client struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucketName    string
}

func NewR2Client(cfg *config.Config) (*R2Client, error) {
	if cfg.R2AccountID == "" || cfg.R2AccessKeyID == "" || cfg.R2SecretAccessKey == "" {
		return nil, fmt.Errorf("R2 configuration incomplete")
	}

	endpoint := cfg.R2Endpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.R2AccountID)
	}

	s3Client := s3.New(s3.Options{
		Region:      "auto",
		BaseEndpoint: &endpoint,
		Credentials: credentials.NewStaticCredentialsProvider(
			cfg.R2AccessKeyID,
			cfg.R2SecretAccessKey,
			"",
		),
	})

	presignClient := s3.NewPresignClient(s3Client)

	return &R2Client{
		client:        s3Client,
		presignClient: presignClient,
		bucketName:    cfg.R2BucketName,
	}, nil
}

func (r *R2Client) GeneratePresignedPutURL(ctx context.Context, key, contentType string, expiry time.Duration) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      &r.bucketName,
		Key:         &key,
		ContentType: &contentType,
	}

	resp, err := r.presignClient.PresignPutObject(ctx, input, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned PUT URL: %w", err)
	}

	return resp.URL, nil
}

func (r *R2Client) GeneratePresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: &r.bucketName,
		Key:    &key,
	}

	resp, err := r.presignClient.PresignGetObject(ctx, input, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned GET URL: %w", err)
	}

	return resp.URL, nil
}

func (r *R2Client) DeleteObject(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: &r.bucketName,
		Key:    &key,
	}

	_, err := r.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

func (r *R2Client) ObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &r.bucketName,
		Key:    &key,
	})
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object: %w", err)
	}
	return true, nil
}

func ClaimDocumentKey(claimID, documentID, filename string) string {
	return fmt.Sprintf("claims/%s/%s/%s", claimID, documentID, filename)
}

func WorkspaceDocumentKey(workspaceID, documentID, filename string) string {
	return fmt.Sprintf("documents/%s/%s/%s", workspaceID, documentID, filename)
}
