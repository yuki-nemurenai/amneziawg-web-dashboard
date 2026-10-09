// Package system connects the service layer to the host: it runs the awg, ip
// and iptables commands and asks public services where the server is.
package system

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Exec runs host commands with os/exec. The zero value is ready to use.
type Exec struct{}

// Run runs the command and returns its standard output. When the command
// fails, the error includes its arguments and standard error, which is where
// awg and ip explain the failure.
func (Exec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		command := strings.Join(append([]string{name}, args...), " ")
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return stdout.Bytes(), fmt.Errorf("%s: %w: %s", command, err, msg)
		}
		return stdout.Bytes(), fmt.Errorf("%s: %w", command, err)
	}
	return stdout.Bytes(), nil
}

// LookPath returns the path of the named executable.
func (Exec) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}
