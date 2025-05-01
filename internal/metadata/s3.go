package metadata

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3api "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sirupsen/logrus"
)

type s3 struct {
	logger logrus.FieldLogger

	bucket *string
	client *s3api.Client
}

var _ Backend = (*s3)(nil)

// NewS3 creates a new AWS S3-backed metadata storage
func NewS3(logger logrus.FieldLogger, config aws.Config, bucket string) (Backend, error) {
	logger = logger.WithField("bucket", bucket)
	logger.Info("created new S3 metadata store")

	client := s3api.NewFromConfig(config)
	return &s3{logger, &bucket, client}, nil
}

func (s *s3) Load(ctx context.Context, key string, out any) error {
	logger := s.logger.WithField("key", key)
	logger.Debug("fetching object...")

	output, err := s.client.GetObject(ctx, &s3api.GetObjectInput{Bucket: s.bucket, Key: &key})
	if err != nil {
		return err
	}
	defer output.Body.Close()

	logger.WithField("size", output.ContentLength).Debug("successfully fetched object")
	return json.NewDecoder(output.Body).Decode(out)
}

func (s *s3) Save(ctx context.Context, key string, data any) error {
	logger := s.logger.WithField("key", key)
	logger.WithField("data", data).Trace("marshalling data to json")

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	logger.Debug("uploading data to object")
	_, err = s.client.PutObject(ctx, &s3api.PutObjectInput{
		Bucket:      s.bucket,
		Key:         &key,
		Body:        bytes.NewReader(body),
		ContentType: aws.String("application/json"),
	})
	return err
}
