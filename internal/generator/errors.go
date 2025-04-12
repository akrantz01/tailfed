package generator

import (
	"context"
	"strings"
	"sync"

	"github.com/akrantz01/tailfed/internal/types"
)

type jobHandler func(context.Context, types.GenerateRequest) error

func wrapWriteJob(ctx context.Context, req types.GenerateRequest, wg *sync.WaitGroup, handler jobHandler) <-chan error {
	errCh := make(chan error, 1)
	wg.Add(1)

	go func() {
		if err := handler(ctx, req); err != nil {
			errCh <- err
		}

		close(errCh)
		wg.Done()
	}()

	return errCh
}

type errorStack []error

func combineErrors(errChs ...<-chan error) error {
	var stack errorStack

	for _, errCh := range errChs {
		if err := <-errCh; err != nil {
			stack = append(stack, err)
		}
	}

	if len(stack) == 0 {
		return nil
	}

	return stack
}

func (e errorStack) Error() string {
	var builder strings.Builder
	for i, err := range e {
		if i != 0 {
			builder.WriteRune(',')
			builder.WriteRune(' ')
		}

		builder.WriteString(err.Error())
	}

	return builder.String()
}
