package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Compare current implementation with the branch spec",
	RunE:  runReview,
}

func runReview(cmd *cobra.Command, args []string) error {
	if err := requireGitRepo(); err != nil {
		return err
	}

	currentBranch, err := getCurrentBranch()
	if err != nil {
		return err
	}

	specContent := findSpecForBranch(currentBranch)
	if specContent == "" {
		color.Red("✗ No spec file found for branch '%s'", currentBranch)
		color.Yellow("  Create one with: do spec <name>")
		color.Yellow("  Or place a .md file in specs/ matching the branch name")
		return fmt.Errorf("spec not found")
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Analyzing implementation against spec..."
	s.Start()

	fullDiff, _ := runGitOutput("diff", "origin/main...HEAD")
	changedFiles, _ := runGitOutput("diff", "origin/main...HEAD", "--name-only")

	s.Stop()

	prompt := fmt.Sprintf(`You are a code reviewer. Compare the implementation against the spec below.

For each acceptance criterion in the spec, evaluate:
- ✅ Implemented and working
- ❌ Not implemented
- ⚠️ Partially implemented

Also identify:
- Unimplemented functional requirements
- Possible gaps or concerns

Spec:
%s

Changed files:
%s

Diff:
%s

Provide a clear, structured review.`, specContent, changedFiles, fullDiff)

	claudeCmd := exec.Command("claude", "--print", prompt)
	claudeCmd.Stdout = os.Stdout
	claudeCmd.Stderr = os.Stderr

	s2 := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s2.Suffix = " Claude is reviewing..."
	s2.Start()

	err = claudeCmd.Run()
	s2.Stop()

	if err != nil {
		return fmt.Errorf("claude review failed: %w", err)
	}

	return nil
}
