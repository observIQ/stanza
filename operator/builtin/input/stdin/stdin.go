package stdin

import (
	"context"
	"fmt"
	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"os"
	"sync"

	"bufio"
	"go.uber.org/zap"
)

func init() {
	operator.Register("stdin", func() operator.Builder { return NewStdinInputConfig("") })
}

// NewStdinInputConfig creates a new stdin input config with default values
func NewStdinInputConfig(operatorID string) *StdinInputConfig {
	return &StdinInputConfig{
		InputConfig: helper.NewInputConfig(operatorID, "stdin"),
	}
}

// StdinInputConfig is the configuration of a stdin input operator.
type StdinInputConfig struct {
	helper.InputConfig `yaml:",inline"`
}

// Build will build a stdin input operator.
func (c *StdinInputConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	stdinInput := &StdinInput{
		InputOperator: inputOperator,
		stdin:         os.Stdin,
	}
	return []operator.Operator{stdinInput}, nil
}

// StdinInput is an operator that reads input from stdin
type StdinInput struct {
	helper.InputOperator
	wg     sync.WaitGroup
	cancel context.CancelFunc
	stdin  *os.File
}

// Start will start generating log entries.
func (g *StdinInput) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	g.cancel = cancel

	stat, err := g.stdin.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat stdin: %s", err)
	}

	if stat.Mode()&os.ModeNamedPipe == 0 {
		g.Warn("No data is being written to stdin")
		return nil
	}

	scanner := bufio.NewScanner(g.stdin)

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if ok := scanner.Scan(); !ok {
				if err := scanner.Err(); err != nil {
					g.Errorf("Scanning failed", zap.Error(err))
				}
				g.Infow("Stdin has been closed")
				return
			}

			e := entry.New()
			e.Record = scanner.Text()
			g.Write(ctx, e)
		}
	}()

	return nil
}

// Stop will stop generating logs.
func (g *StdinInput) Stop() error {
	g.cancel()
	g.wg.Wait()
	return nil
}
