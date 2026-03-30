package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active git worktrees with status",
	RunE:  runList,
}

type worktreeInfo struct {
	path      string
	branch    string
	head      string
	isDirty   bool
	isCurrent bool
}

func runList(cmd *cobra.Command, args []string) error {
	if err := requireGitRepo(); err != nil {
		return err
	}

	worktrees, err := listWorktrees()
	if err != nil {
		return err
	}

	currentBranch, _ := getCurrentBranch()

	for _, wt := range worktrees {
		printWorktree(wt, currentBranch)
	}

	return nil
}

func listWorktrees() ([]worktreeInfo, error) {
	out, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []worktreeInfo
	var current worktreeInfo

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "worktree "):
			if current.path != "" {
				worktrees = append(worktrees, current)
			}
			current = worktreeInfo{path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "HEAD "):
			current.head = strings.TrimPrefix(line, "HEAD ")
			if len(current.head) > 7 {
				current.head = current.head[:7]
			}
		case strings.HasPrefix(line, "branch "):
			branchRef := strings.TrimPrefix(line, "branch ")
			current.branch = strings.TrimPrefix(branchRef, "refs/heads/")
		case line == "":
			if current.path != "" {
				worktrees = append(worktrees, current)
				current = worktreeInfo{}
			}
		}
	}

	if current.path != "" {
		worktrees = append(worktrees, current)
	}

	// Get dirty status for each worktree
	for i := range worktrees {
		statusOut, err := exec.Command("git", "-C", worktrees[i].path, "status", "--porcelain").Output()
		if err == nil && strings.TrimSpace(string(statusOut)) != "" {
			worktrees[i].isDirty = true
		}

		// Get last commit message
		commitMsg, err := exec.Command("git", "-C", worktrees[i].path, "log", "-1", "--format=%s").Output()
		if err == nil {
			msg := strings.TrimSpace(string(commitMsg))
			if len(msg) > 50 {
				msg = msg[:50] + "..."
			}
			worktrees[i].head = worktrees[i].head + " " + msg
		}
	}

	return worktrees, nil
}

func printWorktree(wt worktreeInfo, currentBranch string) {
	isCurrent := wt.branch == currentBranch

	branchColor := color.New(color.FgCyan)
	if isCurrent {
		branchColor = color.New(color.FgGreen, color.Bold)
	}

	dirtyIndicator := ""
	if wt.isDirty {
		dirtyIndicator = color.YellowString(" *")
	}

	currentIndicator := ""
	if isCurrent {
		currentIndicator = color.GreenString(" ◀ current")
	}

	fmt.Printf("%s%s%s\n", branchColor.Sprint(wt.branch), dirtyIndicator, currentIndicator)
	fmt.Printf("  path:   %s\n", wt.path)
	fmt.Printf("  commit: %s\n", wt.head)
	fmt.Println()
}
