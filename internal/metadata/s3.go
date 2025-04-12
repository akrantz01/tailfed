package metadata

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3api "github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3 struct {
	bucket *string
	client *s3api.Client
}

var _ Backend = (*s3)(nil)

// NewS3 creates a new AWS S3-backed metadata storage
func NewS3(config aws.Config, bucket string) (Backend, error) {
	client := s3api.NewFromConfig(config)
	return &s3{&bucket, client}, nil
}

func (s *s3) Load(ctx context.Context, key string, out any) error {
	output, err := s.client.GetObject(ctx, &s3api.GetObjectInput{Bucket: s.bucket, Key: &key})
	if err != nil {
		return err
	}
	defer output.Body.Close()

	return json.NewDecoder(output.Body).Decode(out)
}

func (s *s3) Save(ctx context.Context, key string, data any) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = s.client.PutObject(ctx, &s3api.PutObjectInput{
		Bucket: s.bucket,
		Key:    &key,
		Body:   bytes.NewReader(body),
	})
	return err
}
