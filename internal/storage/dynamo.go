package storage

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// dynamo stores data in a AWS DynamoDB table
type dynamo struct {
	table  string
	client *dynamodb.Client
}

var _ Backend = (*dynamo)(nil)

// NewDynamo creates a new AWS DynamoDB-backed storage
func NewDynamo(config aws.Config, table string) (Backend, error) {
	return &dynamo{table, dynamodb.NewFromConfig(config)}, nil
}

// primaryKey generates a map containing the necessary attributes for the primary key
func (d *dynamo) primaryKey(id string) map[string]types.AttributeValue {
	pk, _ := attributevalue.MarshalMap(primaryKey{id})
	return pk
}

func (d *dynamo) Get(ctx context.Context, id string) (*Flow, error) {
	output, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(d.table), Key: d.primaryKey(id)})
	if err != nil {
		return nil, err
	}

	var flow Flow
	if err := attributevalue.UnmarshalMap(output.Item, &flow); err != nil {
		return nil, err
	}

	return &flow, nil
}

func (d *dynamo) Put(ctx context.Context, flow *Flow) error {
	av, err := attributevalue.MarshalMap(flow)
	if err != nil {
		return err
	}

	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(d.table), Item: av})
	return err
}

func (d *dynamo) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(d.table), Key: d.primaryKey(id)})
	return err
}

type primaryKey struct {
	ID string
}
