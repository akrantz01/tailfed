//go:build windows

package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

func (r *refresh) Run(*cobra.Command, []string) error {
	return errors.New("refresh currently unsupported on windows")
}
