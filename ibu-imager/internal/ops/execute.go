package ops

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Execute is an interface for executing external commands and capturing their output.
//
//go:generate mockgen -source=execute.go -package=ops -destination=mock_execute.go
type Execute interface {
	Execute(command string, args ...string) (string, error)
}

type executor struct {
	verbose bool
	log     *logrus.Logger
}

// NewExecutor creates and returns an Execute interface for executing external commands
func NewExecutor(logger *logrus.Logger, verbose bool) Execute {
	return &executor{log: logger, verbose: verbose}
}

func (e *executor) Execute(command string, args ...string) (string, error) {
	e.log.Println("Executing", command, args)
	cmd := exec.Command(command, args...)
	var stdoutBytes, stderrBytes bytes.Buffer
	cmd.Stdout = &stdoutBytes
	cmd.Stderr = &stderrBytes
	err := cmd.Run()
	return strings.TrimSpace(stdoutBytes.String()), errors.Wrap(err, stderrBytes.String())
}
