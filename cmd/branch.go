package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	branchTmuxMode   string
	branchCustomPath string
	branchSession    string
	branchFromBranch string
	branchFromHere   bool
)

var branchCmd = &cobra.Command{
	Use:   "branch [name]",
	Short: "Create a git worktree and open Claude Code in tmux",
	Args:  cobra.ExactArgs(1),
	RunE:  runBranch,
}

func init() {
	branchCmd.Flags().StringVarP(&branchTmuxMode, "tmux", "t", "window", "tmux mode: window or session")
	branchCmd.Flags().StringVarP(&branchCustomPath, "path", "p", "", "custom path for the worktree")
	branchCmd.Flags().StringVarP(&branchSession, "session", "s", "", "tmux session name (for session mode)")
	branchCmd.Flags().StringVarP(&branchFromBranch, "from", "f", "", "create new branch from this branch instead of origin/main")
	branchCmd.Flags().BoolVarP(&branchFromHere, "here", "H", false, "create new branch from current HEAD")
}

func runBranch(cmd *cobra.Command, args []string) error {
	if err := requireGitRepo(); err != nil {
		return err
	}

	branchName := args[0]

	// Check if inside tmux
	if os.Getenv("TMUX") == "" {
		return fmt.Errorf("not inside a tmux session. Please run this command from within tmux")
	}

	// Determine worktree path
	worktreePath, err := getWorktreePath(branchName, branchCustomPath)
	if err != nil {
		return err
	}

	// Create worktree
	if err := createWorktree(branchName, worktreePath); err != nil {
		return err
	}

	color.Green("✓ Worktree created at %s", worktreePath)

	// Open tmux window/session
	target, err := openTmux(branchName, worktreePath, branchTmuxMode, branchSession)
	if err != nil {
		return fmt.Errorf("failed to open tmux: %w", err)
	}

	// Send claude command to tmux
	if err := sendClaudeToTmux(target, ""); err != nil {
		color.Yellow("⚠ Failed to start Claude Code: %v", err)
	} else {
		color.Green("✓ Claude Code started in tmux %s '%s'", branchTmuxMode, branchName)
	}

	return nil
}

func getWorktreePath(branchName, customPath string) (string, error) {
	if customPath != "" {
		return customPath, nil
	}

	root, err := getRepoRoot()
	if err != nil {
		return "", err
	}

	repoName, err := getRepoName()
	if err != nil {
		return "", err
	}

	// Sanitize branch name for directory: replace / and other special chars with -
	safeBranch := strings.ReplaceAll(branchName, "/", "-")
	safeBranch = strings.ReplaceAll(safeBranch, "\\", "-")

	parent := filepath.Dir(root)
	return filepath.Join(parent, repoName+"-"+safeBranch), nil
}

func createWorktree(branchName, worktreePath string) error {
	if branchExistsLocally(branchName) {
		// Branch exists locally - use it directly
		color.Yellow("→ Branch '%s' exists locally, using it", branchName)
		return exec.Command("git", "worktree", "add", worktreePath, branchName).Run()
	}

	if branchExistsRemotely(branchName) {
		// Branch exists on remote - create from remote
		color.Yellow("→ Branch '%s' exists on remote, checking out from origin", branchName)
		return exec.Command("git", "worktree", "add", "--track", "-b", branchName, worktreePath, "origin/"+branchName).Run()
	}

	// New branch - determine base
	base, err := getWorktreeBase()
	if err != nil {
		return err
	}

	color.Yellow("→ Creating new branch '%s' from %s", branchName, base)
	return exec.Command("git", "worktree", "add", "-b", branchName, worktreePath, base).Run()
}

func getWorktreeBase() (string, error) {
	if branchFromHere {
		return "HEAD", nil
	}

	if branchFromBranch != "" {
		return branchFromBranch, nil
	}

	// Default: fetch and use origin/main
	fmt.Println("Fetching origin/main...")
	if err := exec.Command("git", "fetch", "origin", "main").Run(); err != nil {
		// Try master
		if err2 := exec.Command("git", "fetch", "origin", "master").Run(); err2 != nil {
			color.Yellow("⚠ Could not fetch from origin, using local HEAD as base")
			return "HEAD", nil
		}
		return "origin/master", nil
	}
	return "origin/main", nil
}

func openTmux(branchName, worktreePath, mode, session string) (string, error) {
	// Sanitize window/session name (tmux doesn't like slashes)
	windowName := strings.ReplaceAll(branchName, "/", "-")

	if mode == "session" {
		sessionName := session
		if sessionName == "" {
			sessionName = windowName
		}
		err := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", worktreePath).Run()
		if err != nil {
			return "", fmt.Errorf("failed to create tmux session '%s': %w", sessionName, err)
		}
		return sessionName, nil
	}

	// Default: window mode
	err := exec.Command("tmux", "new-window", "-n", windowName, "-c", worktreePath).Run()
	if err != nil {
		return "", fmt.Errorf("failed to create tmux window '%s': %w", windowName, err)
	}
	return windowName, nil
}

func sendClaudeToTmux(target, context string) error {
	claudeCmd := "claude"
	if context != "" {
		// Save context to temp file approach
		tmpFile, err := os.CreateTemp("", "do-spec-*.md")
		if err != nil {
			return err
		}
		if _, err := tmpFile.WriteString(context); err != nil {
			tmpFile.Close()
			return err
		}
		tmpFile.Close()
		claudeCmd = fmt.Sprintf("claude '%s'", tmpFile.Name())
	}

	return exec.Command("tmux", "send-keys", "-t", target, claudeCmd, "Enter").Run()
}
