package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2Config holds Cloudflare R2 connection settings loaded from env.
type R2Config struct {
	Endpoint   string
	Bucket     string
	AccessKey  string
	SecretKey  string
	PublicBase string
}

// LoadR2Config reads R2 settings. CF_KEY is the S3 Access Key ID,
// CF_SEC the S3 Secret Access Key, CF_VALUE is ignored.
func LoadR2Config() (R2Config, error) {
	cfg := R2Config{
		Endpoint:   strings.TrimSpace(os.Getenv("CF_ENDPOINT")),
		Bucket:     strings.TrimSpace(os.Getenv("CF_BUCKET")),
		AccessKey:  strings.TrimSpace(os.Getenv("CF_KEY")),
		SecretKey:  strings.TrimSpace(os.Getenv("CF_SEC")),
		PublicBase: strings.TrimRight(strings.TrimSpace(os.Getenv("CF_PUBLIC_BASE_URL")), "/"),
	}
	var missing []string
	if cfg.Endpoint == "" {
		missing = append(missing, "CF_ENDPOINT")
	}
	if cfg.Bucket == "" {
		missing = append(missing, "CF_BUCKET")
	}
	if cfg.AccessKey == "" {
		missing = append(missing, "CF_KEY")
	}
	if cfg.SecretKey == "" {
		missing = append(missing, "CF_SEC")
	}
	if cfg.PublicBase == "" {
		missing = append(missing, "CF_PUBLIC_BASE_URL")
	}
	if len(missing) > 0 {
		return R2Config{}, fmt.Errorf("R2 chưa được cấu hình (thiếu %s)", strings.Join(missing, ", "))
	}
	return cfg, nil
}

func newR2Client(cfg R2Config) *s3.Client {
	endpoint := cfg.Endpoint
	return s3.New(s3.Options{
		Region:       "auto",
		Credentials:  aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
		BaseEndpoint: &endpoint,
		UsePathStyle: true,
	})
}

// PutAvatar uploads avatar bytes to R2 under a stable per-player key so each
// upload overwrites the previous one (same behaviour as the old Cloudinary
// PublicID). Returns the versioned public URL (query ?v busts CDN cache).
func PutAvatar(ctx context.Context, playerID, contentType string, data []byte, version int64) (string, error) {
	if playerID == "" {
		return "", errors.New("playerID trống")
	}
	cfg, err := LoadR2Config()
	if err != nil {
		return "", err
	}
	key := "avatars/" + playerID
	client := newR2Client(cfg)
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &cfg.Bucket,
		Key:           &key,
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(len(data))),
		ContentType:   &contentType,
		CacheControl:  aws.String("public, max-age=86400"),
	})
	if err != nil {
		return "", fmt.Errorf("R2 upload error: %w", err)
	}
	return fmt.Sprintf("%s/%s?v=%d", cfg.PublicBase, key, version), nil
}
