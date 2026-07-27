package executor

import (
	"errors"
	"testing"
)

func TestInterpretWaitErrorReportsUnexpectedFailure(t *testing.T) {
	waitErr := errors.New("wait failed")
	exitCode, err := interpretWaitError(waitErr, false)
	if exitCode != nil {
		t.Fatalf("exit code = %d, want nil", *exitCode)
	}
	if !errors.Is(err, waitErr) {
		t.Fatalf("error = %v, want wrapped %v", err, waitErr)
	}
}

func TestInterpretWaitErrorAllowsContextInterruption(t *testing.T) {
	exitCode, err := interpretWaitError(errors.New("context canceled"), true)
	if err != nil || exitCode != nil {
		t.Fatalf("exit code/error = %v/%v, want nil/nil", exitCode, err)
	}
}
