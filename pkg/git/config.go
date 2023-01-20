package git

import (
	"bufio"
	"context"
	"fmt"
	"strings"
)

type Config struct {
	values map[string][]string
}

func ListConfig(ctx context.Context, r *Repo) (*Config, error) {
	result, err := r.ExecGit(ctx, "config", "--list")
	if err != nil {
		if result.ExitCode != 0 {
			result.PrintOutput()
		}

		return nil, err
	}

	values := make(map[string][]string)
	scanner := bufio.NewScanner(strings.NewReader(result.Stdout))
	for {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, fmt.Errorf("error parsing output: %w", err)
			}
			break
		}

		line := scanner.Text()
		tokens := strings.SplitN(line, "=", 2)
		if len(tokens) != 2 {
			return nil, fmt.Errorf("error parsing line %q (expected 2 tokens)", line)
		}

		k := tokens[0]
		v := tokens[1]
		values[k] = append(values[k], v)
	}

	return &Config{values: values}, nil
}

// Get returns the values of the key joined with commas, or "" if not found
func (c *Config) Get(k string) string {
	values := c.values[k]
	return strings.Join(values, ",")
}
