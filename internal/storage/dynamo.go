package storage

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sirupsen/logrus"
)

// dynamo stores data in a AWS DynamoDB table
type dynamo struct {
	logger logrus.FieldLogger

	table  string
	client *dynamodb.Client
}

var _ Backend = (*dynamo)(nil)

// NewDynamo creates a new AWS DynamoDB-backed storage
func NewDynamo(logger logrus.FieldLogger, config aws.Config, table string) (Backend, error) {
	logger = logger.WithField("table", table)
	logger.Info("created new DynamoDB storage")
	return &dynamo{logger, table, dynamodb.NewFromConfig(config)}, nil
}

// primaryKey generates a map containing the necessary attributes for the primary key
func (d *dynamo) primaryKey(id string) map[string]types.AttributeValue {
	pk, _ := attributevalue.MarshalMap(primaryKey{id})
	return pk
}

func (d *dynamo) Get(ctx context.Context, id string) (*Flow, error) {
	logger := d.logger.WithField("id", id)
	logger.Debug("fetching item from table")

	output, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(d.table), Key: d.primaryKey(id)})
	if err != nil {
		return nil, err
	}

	logger.Debug("deserializing attribute values")
	var flow Flow
	if err := attributevalue.UnmarshalMap(output.Item, &flow); err != nil {
		return nil, err
	}

	return &flow, nil
}

func (d *dynamo) Put(ctx context.Context, flow *Flow) error {
	logger := d.logger.WithField("id", flow.ID)
	logger.Debug("serializing flow to attribute values")

	av, err := attributevalue.MarshalMap(flow)
	if err != nil {
		return err
	}

	logger.Debug("writing item to table")
	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(d.table), Item: av})
	return err
}

func (d *dynamo) Delete(ctx context.Context, id string) error {
	d.logger.WithField("id", id).Debug("deleting item (if exists)")
	_, err := d.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(d.table), Key: d.primaryKey(id)})
	return err
}

type primaryKey struct {
	ID string
}
