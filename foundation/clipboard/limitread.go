package clipboard

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
)

// errOutputTooLarge is returned when a command's stdout exceeds the allowed size.
var errOutputTooLarge = errors.New("command output too large")

// readCommandOutput runs cmd, reading at most maxBytes from stdout. If the
// output exceeds maxBytes the process is killed and errOutputTooLarge is
// returned. The caller must not call cmd.Start or cmd.Output — this function
// manages the full command lifecycle.
func readCommandOutput(cmd *exec.Cmd, maxBytes int64) ([]byte, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}

	limited := io.LimitReader(stdout, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("read stdout: %w", err)
	}

	if int64(len(data)) > maxBytes {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf(
			"%w: output exceeds %d MB limit",
			errOutputTooLarge, maxBytes/(1024*1024),
		)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("wait: %w", err)
	}

	return data, nil
}
