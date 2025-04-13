package launcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

type stepFunction struct {
	machine *string
	client  *sfn.Client
}

var _ Backend = (*stepFunction)(nil)

// NewStepFunction creates a launcher backed via an AWS Step Function workflow
func NewStepFunction(config aws.Config, machine string) (Backend, error) {
	client := sfn.NewFromConfig(config)

	output, err := client.DescribeStateMachine(context.Background(), &sfn.DescribeStateMachineInput{StateMachineArn: &machine})
	if err != nil {
		return nil, err
	}
	if output.Status != types.StateMachineStatusActive {
		return nil, errors.New("state machine is not active")
	}

	return &stepFunction{output.StateMachineArn, client}, nil
}

func (sf *stepFunction) Launch(id string, addresses []netip.AddrPort) error {
	encoded, err := json.Marshal(&Request{id, addresses})
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	_, err = sf.client.StartExecution(context.Background(), &sfn.StartExecutionInput{
		StateMachineArn: sf.machine,
		Name:            &id,
		Input:           aws.String(string(encoded)),
	})
	return err
}
