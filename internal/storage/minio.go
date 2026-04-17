package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"wzap/internal/config"
	"wzap/internal/logger"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Minio struct {
	Client *minio.Client
	Bucket string
}

func New(cfg *config.Config) (*Minio, error) {
	client, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.MinioBucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, cfg.MinioBucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		logger.Info().Str("component", "s3").Str("bucket", cfg.MinioBucket).Msg("Created new MinIO bucket")
	}

	logger.Info().Str("component", "s3").Msg("Successfully connected to MinIO")

	return &Minio{
		Client: client,
		Bucket: cfg.MinioBucket,
	}, nil
}

func (m *Minio) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	return m.UploadWithMeta(ctx, key, reader, size, contentType, nil)
}

// UploadWithMeta faz upload com metadata customizado (user metadata).
// Útil para persistir o filename original, que depois é recuperado via StatMeta
// e embarcado em `Content-Disposition` nos downloads.
func (m *Minio) UploadWithMeta(ctx context.Context, key string, reader io.Reader, size int64, contentType string, userMeta map[string]string) error {
	_, err := m.Client.PutObject(ctx, m.Bucket, key, reader, size, minio.PutObjectOptions{
		ContentType:  contentType,
		UserMetadata: userMeta,
	})
	return err
}

func (m *Minio) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := m.Client.GetObject(ctx, m.Bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// Stat returns the stored Content-Type and Size for the given key.
// Returns an error if the object does not exist.
func (m *Minio) Stat(ctx context.Context, key string) (contentType string, size int64, err error) {
	info, err := m.Client.StatObject(ctx, m.Bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return "", 0, err
	}
	return info.ContentType, info.Size, nil
}

// StatMeta returns content-type, size and user metadata for the given key.
// User metadata keys are returned lowercased and WITHOUT the "X-Amz-Meta-" prefix.
func (m *Minio) StatMeta(ctx context.Context, key string) (contentType string, size int64, userMeta map[string]string, err error) {
	info, err := m.Client.StatObject(ctx, m.Bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return "", 0, nil, err
	}
	return info.ContentType, info.Size, info.UserMetadata, nil
}

func (m *Minio) PresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	presignedURL, err := m.Client.PresignedGetObject(ctx, m.Bucket, key, expiry, reqParams)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}

func (m *Minio) Health(ctx context.Context) error {
	_, err := m.Client.BucketExists(ctx, m.Bucket)
	return err
}
