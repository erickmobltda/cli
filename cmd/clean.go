package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove worktrees for merged branches",
	RunE:  runClean,
}

func runClean(cmd *cobra.Command, args []string) error {
	if err := requireGitRepo(); err != nil {
		return err
	}

	worktrees, err := listWorktrees()
	if err != nil {
		return err
	}

	if len(worktrees) <= 1 {
		color.Yellow("No extra worktrees found.")
		return nil
	}

	// Get merged branches
	mergedOut, err := exec.Command("git", "branch", "--merged", "origin/main").Output()
	if err != nil {
		// Try master
		mergedOut, err = exec.Command("git", "branch", "--merged", "origin/master").Output()
		if err != nil {
			return fmt.Errorf("failed to get merged branches: %w", err)
		}
	}

	mergedBranches := map[string]bool{}
	for _, line := range strings.Split(string(mergedOut), "\n") {
		b := strings.TrimSpace(strings.TrimPrefix(line, "* "))
		if b != "" {
			mergedBranches[b] = true
		}
	}

	// Filter worktrees to remove (skip first/main worktree)
	var toRemove []worktreeInfo
	for i, wt := range worktrees {
		if i == 0 {
			continue // skip main worktree
		}
		if wt.branch == "main" || wt.branch == "master" {
			continue
		}
		if mergedBranches[wt.branch] {
			toRemove = append(toRemove, wt)
		}
	}

	if len(toRemove) == 0 {
		color.Green("No merged worktrees to clean up.")
		return nil
	}

	fmt.Println("The following worktrees will be removed:")
	for _, wt := range toRemove {
		color.Yellow("  %s → %s", wt.branch, wt.path)
	}

	fmt.Print("\nProceed? [y/N] ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))

	if answer != "y" && answer != "yes" {
		color.Yellow("Aborted.")
		return nil
	}

	removed := 0
	for _, wt := range toRemove {
		if err := exec.Command("git", "worktree", "remove", wt.path).Run(); err != nil {
			color.Red("✗ Failed to remove worktree %s: %v", wt.path, err)
			continue
		}
		if err := exec.Command("git", "branch", "-d", wt.branch).Run(); err != nil {
			color.Yellow("⚠ Worktree removed but failed to delete branch %s: %v", wt.branch, err)
		} else {
			color.Green("✓ Removed %s (%s)", wt.branch, wt.path)
		}
		removed++
	}

	color.Green("\n✓ Cleaned up %d worktree(s)", removed)
	return nil
}
