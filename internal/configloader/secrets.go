package configloader

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	arns "github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

const filePrefix = "file://"

type fetcherFn func(string) (string, error)

// secretLoader retrieves values from AWS secrets manager and SSM parameter store
type secretLoader struct {
	configured     bool
	ssm            *ssm.Client
	secretsmanager *secretsmanager.Client

	errs loadErrors
}

// newSecretLoader creates a new secretLoader with the clients configured
func newSecretLoader(config *aws.Config) *secretLoader {
	l := &secretLoader{errs: make(loadErrors)}

	if config != nil {
		l.configured = true
		l.ssm = ssm.NewFromConfig(*config)
		l.secretsmanager = secretsmanager.NewFromConfig(*config)
	}

	return l
}

// Load reads a value from secrets manager or SSM parameter store if the value looks
// like an ARN and is for the correct service
func (s *secretLoader) Load(key, value string) (string, any) {
	fetcher := s.fetcherFromFile(value)
	if fetcher == nil {
		fetcher = s.fetcherFromAws(value)
	}
	if fetcher == nil {
		return key, value
	}

	if secret, err := fetcher(value); err != nil {
		s.errs[key] = err
		return key, nil
	} else {
		return key, secret
	}
}

// Err returns an error if any keys failed to load
func (s *secretLoader) Err() error {
	if len(s.errs) == 0 {
		return nil
	}

	return s.errs
}

func (s *secretLoader) fetcherFromFile(value string) fetcherFn {
	if strings.HasPrefix(value, filePrefix) {
		return s.fromFile
	}

	return nil
}

func (s *secretLoader) fromFile(value string) (string, error) {
	path := strings.TrimPrefix(value, filePrefix)
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("could not open %q: %w", path, err)
	}
	defer file.Close()

	contents, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read %q: %w", path, err)
	}

	return string(contents), nil
}

func (s *secretLoader) fetcherFromAws(value string) fetcherFn {
	if !s.configured {
		return nil
	}

	arn, err := arns.Parse(value)
	if err != nil {
		return nil
	}

	switch arn.Service {
	case "secretsmanager":
		return s.fromSecretsManager
	case "ssm":
		if !strings.HasPrefix(arn.Resource, "parameter/") {
			return nil
		}

		return s.fromParameterStore
	default:
		return nil
	}
}

func (s *secretLoader) fromParameterStore(key string) (string, error) {
	output, err := s.ssm.GetParameter(context.Background(), &ssm.GetParameterInput{
		Name:           &key,
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return "", err
	}

	return *output.Parameter.Value, nil
}

func (s *secretLoader) fromSecretsManager(key string) (string, error) {
	output, err := s.secretsmanager.GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{SecretId: &key})
	if err != nil {
		return "", err
	}

	return *output.SecretString, nil
}

type loadErrors map[string]error

var _ error = (*loadErrors)(nil)

func (l loadErrors) Error() string {
	formatted := make([]string, 0, len(l))

	for key, err := range l {
		formatted = append(formatted, fmt.Sprintf("couldn't load key %q: %s", key, err))
	}

	return strings.Join(formatted, "; ")
}
