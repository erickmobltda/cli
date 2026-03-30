package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [spec-file]",
	Short: "Start development from a spec file",
	Args:  cobra.ExactArgs(1),
	RunE:  runStart,
}

func runStart(cmd *cobra.Command, args []string) error {
	if err := requireGitRepo(); err != nil {
		return err
	}

	if os.Getenv("TMUX") == "" {
		return fmt.Errorf("not inside a tmux session. Please run this command from within tmux")
	}

	specPath := args[0]
	content, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("failed to read spec file: %w", err)
	}

	title, err := extractSpecTitle(string(content))
	if err != nil {
		return fmt.Errorf("failed to extract title from spec: %w", err)
	}

	branchName := "feature/" + sanitizeSlug(title)
	color.Cyan("→ Branch: %s", branchName)
	color.Cyan("→ Spec: %s", specPath)

	// Determine worktree path (reuse branchFromBranch=origin/main default)
	worktreePath, err := getWorktreePath(branchName, "")
	if err != nil {
		return err
	}

	// Create worktree with default base (origin/main)
	branchFromBranch = ""
	branchFromHere = false
	if err := createWorktree(branchName, worktreePath); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	color.Green("✓ Worktree created at %s", worktreePath)

	target, err := openTmux(branchName, worktreePath, "window", "")
	if err != nil {
		return fmt.Errorf("failed to open tmux: %w", err)
	}

	if err := sendClaudeToTmux(target, string(content)); err != nil {
		color.Yellow("⚠ Failed to start Claude Code with spec context: %v", err)
	} else {
		color.Green("✓ Claude Code started with spec context")
	}

	return nil
}

func extractSpecTitle(content string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# "), nil
		}
	}
	return "", fmt.Errorf("no title found (expected '# Title' on first heading line)")
}
