package apk

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// We only want to load the AWS configuration and create a client once.
var loadS3Client = sync.OnceValues(func() (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("config.LoadDefaultConfig failed: %w", err)
	}
	return s3.NewFromConfig(cfg), nil
})

// fetchS3 fetches an object from S3.
func fetchS3(ctx context.Context, u *url.URL) (io.ReadCloser, error) {
	client, err := loadS3Client()
	if err != nil {
		return nil, err
	}
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(u.Host),
		Key:    aws.String(strings.TrimPrefix(u.Path, "/")),
	})
	if err != nil {
		return nil, fmt.Errorf("(*s3.Client).GetObject failed: %w", err)
	}
	return out.Body, nil
}
