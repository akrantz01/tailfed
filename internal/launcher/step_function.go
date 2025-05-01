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
	"github.com/sirupsen/logrus"
)

type stepFunction struct {
	logger logrus.FieldLogger

	machine *string
	client  *sfn.Client
}

var _ Backend = (*stepFunction)(nil)

// NewStepFunction creates a launcher backed via an AWS Step Function workflow
func NewStepFunction(logger logrus.FieldLogger, config aws.Config, machine string) (Backend, error) {
	client := sfn.NewFromConfig(config)

	logger.WithField("machine", machine).Debug("attempting to find state machine...")
	output, err := client.DescribeStateMachine(context.Background(), &sfn.DescribeStateMachineInput{StateMachineArn: &machine})
	if err != nil {
		return nil, err
	}
	if output.Status != types.StateMachineStatusActive {
		return nil, errors.New("state machine is not active")
	}

	logger.WithField("arn", output.StateMachineArn).Info("created new step functions-backed launcher")
	return &stepFunction{logger, output.StateMachineArn, client}, nil
}

func (sf *stepFunction) Launch(id string, addresses []netip.AddrPort) error {
	sf.logger.WithField("id", id).Debug("launching state machine...")

	encoded, err := json.Marshal(&Request{id, addresses})
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	resp, err := sf.client.StartExecution(context.Background(), &sfn.StartExecutionInput{
		StateMachineArn: sf.machine,
		Name:            &id,
		Input:           aws.String(string(encoded)),
	})
	if err != nil {
		return err
	}

	sf.logger.WithField("execution", resp.ExecutionArn).Debug("state machine instance launched")
	return nil
}
