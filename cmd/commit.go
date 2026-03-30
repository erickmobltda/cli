package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit [message]",
	Short: "Commit staged changes, using Claude to generate message if not provided",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runCommit,
}

func runCommit(cmd *cobra.Command, args []string) error {
	if err := requireGitRepo(); err != nil {
		return err
	}

	// Check for staged changes
	out, err := exec.Command("git", "diff", "--cached", "--name-only").Output()
	if err != nil {
		return fmt.Errorf("failed to check staged changes: %w", err)
	}
	if strings.TrimSpace(string(out)) == "" {
		return fmt.Errorf("no staged changes found. Use 'git add' to stage files before committing")
	}

	var message string

	if len(args) > 0 {
		message = args[0]
	} else {
		// Generate message with Claude
		var genErr error
		message, genErr = generateCommitMessage()
		if genErr != nil {
			return genErr
		}
	}

	color.Cyan("→ Committing with message: %s", message)

	if err := exec.Command("git", "commit", "-m", message).Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	color.Green("✓ Committed successfully")
	return nil
}

func generateCommitMessage() (string, error) {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Generating commit message with Claude..."
	s.Start()
	defer s.Stop()

	status, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %w", err)
	}

	diff, err := exec.Command("git", "diff", "--cached", "--name-status").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git diff: %w", err)
	}

	prompt := fmt.Sprintf(`Generate a git commit message in Conventional Commits format for the following changes.
Return ONLY the commit message, nothing else. No explanation, no code blocks, just the message.

Git status:
%s

Changed files:
%s`, string(status), string(diff))

	claudeCmd := exec.Command("claude", "--print", "--allowedTools", "Bash(git:*)", prompt)
	claudeCmd.Stderr = os.Stderr
	result, err := claudeCmd.Output()
	if err != nil {
		return "", fmt.Errorf("claude failed to generate commit message: %w", err)
	}

	message := strings.TrimSpace(string(result))
	if message == "" {
		return "", fmt.Errorf("claude returned empty commit message")
	}

	return message, nil
}
