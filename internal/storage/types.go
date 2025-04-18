package storage

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Backend provides a mechanism for storing state between Lambda handlers
type Backend interface {
	// Get retrieves a flow by its ID
	Get(ctx context.Context, id string) (*Flow, error)
	// Put writes a flow
	Put(ctx context.Context, flow *Flow) error
	// Delete permanently deletes a flow
	Delete(ctx context.Context, id string) error
}

// Flow represents all the data associated with a single token issuance process
type Flow struct {
	ID        string
	Status    Status
	ExpiresAt UnixTime

	Secret      []byte
	Node        string
	PublicKey   string
	DNSName     string
	MachineName string
	Hostname    string
	Tailnet     string
	OS          string
	Tags        []string
	Authorized  bool
	External    bool
}

// Status represents the current status of the flow
type Status string

var (
	StatusPending Status = "pending"
	StatusFailed  Status = "failed"
	StatusSuccess Status = "success"
)

func (s *Status) UnmarshalText(raw string) error {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "pending":
		*s = StatusPending
	case "failed":
		*s = StatusFailed
	case "success":
		*s = StatusSuccess
	default:
		return errors.New("unknown status")
	}
	return nil
}

func (s *Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(*s))
}

func (s *Status) UnmarshalJSON(encoded []byte) error {
	var raw string
	if err := json.Unmarshal(encoded, &raw); err != nil {
		return err
	}

	return s.UnmarshalText(raw)
}

func (s *Status) MarshalDynamoDBAttributeValue() (types.AttributeValue, error) {
	return attributevalue.Marshal(string(*s))
}

func (s *Status) UnmarshalDynamoDBAttributeValue(value types.AttributeValue) error {
	var raw string
	if err := attributevalue.Unmarshal(value, &raw); err != nil {
		return err
	}

	return s.UnmarshalText(raw)
}

// UnixTime wraps a [time.Time] to serialize it as a unix timestamp with seconds resolution
type UnixTime time.Time

func (u *UnixTime) UnmarshalText(raw string) error {
	epoch, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return err
	}

	*u = UnixTime(time.Unix(epoch, 0))
	return nil
}

func (u *UnixTime) MarshalJSON() ([]byte, error) {
	epoch := time.Time(*u).Unix()
	return json.Marshal(strconv.FormatInt(epoch, 10))
}

func (u *UnixTime) UnmarshalJSON(raw []byte) error {
	var decoded string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}

	return u.UnmarshalText(decoded)
}

func (u *UnixTime) MarshalDynamoDBAttributeValue() (types.AttributeValue, error) {
	epoch := time.Time(*u).Unix()
	return attributevalue.Marshal(epoch)
}

func (u *UnixTime) UnmarshalDynamoDBAttributeValue(value types.AttributeValue) error {
	var decoded string
	if err := attributevalue.Unmarshal(value, &decoded); err != nil {
		return err
	}

	return u.UnmarshalText(decoded)
}
