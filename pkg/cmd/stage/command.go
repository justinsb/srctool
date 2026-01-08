package stage

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

func AddCommand(ctx context.Context, parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "stage",
		Short: "git stage powertool",
	}
	var opt Options
	opt.InitDefaults()

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return Run(cmd.Context(), opt, args)
	}
	cmd.Flags().BoolVar(&opt.Preview, "preview", opt.Preview, "Preview the hunks that would be staged without actually staging them")
	cmd.Flags().StringVar(&opt.Pattern, "pattern", opt.Pattern, "Regex pattern to match hunks to stage")
	parent.AddCommand(cmd)
}

type Options struct {
	Pattern string
	Preview bool
}

func (o *Options) InitDefaults() {

}

func Run(ctx context.Context, opt Options, args []string) error {
	if opt.Pattern == "" {
		return fmt.Errorf("pattern is required")
	}

	re, err := regexp.Compile(opt.Pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern %q: %w", opt.Pattern, err)
	}

	// 1. Get the diff with 0 context lines
	cmd := exec.Command("git", "diff", "-U0")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running git diff: %w", err)
	}

	// 2. Parse the diff into hunks
	hunks := parseDiff(bytes.NewReader(stdout.Bytes()))

	var buffer bytes.Buffer
	matchedCount := 0

	for _, hunk := range hunks {
		// Only match against the content of the hunk (lines starting with +)
		if re.MatchString(hunk.content) {
			buffer.WriteString(hunk.header)
			buffer.WriteString(hunk.content)
			matchedCount++
		}
	}

	if matchedCount == 0 {
		fmt.Fprintf(os.Stderr, "No hunks matched the pattern.")
		return nil
	}

	if opt.Preview {
		fmt.Fprintf(os.Stderr, "Previewing %d hunk(s) matching '%s':\n", matchedCount, opt.Pattern)
		fmt.Fprintf(os.Stdout, buffer.String())
		return nil
	}
	// 3. Apply the filtered hunks to the index (--cached)
	applyCmd := exec.Command("git", "apply", "--cached", "--unidiff-zero", "-")
	applyCmd.Stdin = &buffer
	applyCmd.Stderr = os.Stderr

	if err := applyCmd.Run(); err != nil {
		return fmt.Errorf("error applying patch: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully staged %d hunk(s) matching '%s'\n", matchedCount, opt.Pattern)
	return nil
}

type hunk struct {
	header  string // The diff/index/---/+++/@@ lines
	content string // The actual + and - lines
}

func parseDiff(r io.Reader) []hunk {
	var hunks []hunk
	scanner := bufio.NewScanner(r)

	var currentHunk *hunk
	var headerLines []string

	for scanner.Scan() {
		line := scanner.Text()

		// New file header or hunk header starts
		if strings.HasPrefix(line, "diff --git") {
			headerLines = []string{line}
		} else if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "index") {
			headerLines = append(headerLines, line)
		} else if strings.HasPrefix(line, "@@") {
			if currentHunk != nil {
				hunks = append(hunks, *currentHunk)
			}
			currentHunk = &hunk{}
			currentHunk.header = strings.Join(headerLines, "\n") + "\n"
			currentHunk.content = line + "\n"
		} else {
			// It's a content line (+ or -)
			if currentHunk != nil {
				currentHunk.content += line + "\n"
			}
		}
	}
	if currentHunk != nil {
		hunks = append(hunks, *currentHunk)
	}
	return hunks
}
