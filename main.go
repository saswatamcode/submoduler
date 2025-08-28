package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Global flag for verbose output.
var verbose bool

// main is the entry point of the script.
// It parses command-line arguments for specific submodule commits or tags
// and then updates all submodules accordingly.
//
// Usage:
// go run main.go [-v] [path/to/submodule1=commit_hash] [path/to/submodule2=v1.2.3]
func main() {
	// Define command-line flags.
	flag.BoolVar(&verbose, "v", false, "Enable verbose output to see the commands being run.")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "%s [-v] [submodule1=ref] [submodule2=ref] ...\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "This script updates git submodules.")
		fmt.Fprintln(os.Stderr, "By default, it pulls the latest commit for each submodule's tracked branch.")
		fmt.Fprintln(os.Stderr, "You can specify a commit, tag, or branch for a submodule using the 'path=ref' format (e.g., my_sub=v1.2.3).")
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
	}
	flag.Parse()

	// A map to hold submodule paths and the specific ref (commit/tag/branch) to check out.
	// e.g., {"path/to/submodule": "a1b2c3d"} or {"path/to/submodule": "v1.2.3"}
	specificRefs := parseArgs(flag.Args())

	// Get the root directory of the Git repository.
	rootDir, err := getGitRootDir()
	if err != nil {
		fmt.Printf("Error: Not a git repository or git command not found. %v\n", err)
		os.Exit(1)
	}

	// First, ensure all submodules are initialized and cloned.
	// This handles cases where a user has cloned the repo but not run `git submodule update --init`.
	fmt.Println("Initializing and cloning any missing submodules...")
	if err := runCommand(rootDir, "git", "submodule", "update", "--init", "--recursive", "--progress"); err != nil {
		fmt.Printf("Error initializing submodules: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Initialization complete.")
	fmt.Println("---------------------------------")

	// Get a list of all submodule paths.
	submodules, err := getSubmodules(rootDir)
	if err != nil {
		fmt.Printf("Error getting submodules: %v\n", err)
		os.Exit(1)
	}

	if len(submodules) == 0 {
		fmt.Println("No submodules found.")
		return
	}

	fmt.Printf("Found %d submodules. Starting update...\n\n", len(submodules))

	// Separate submodules into two groups: those with specific refs and those to be updated to latest.
	var submodulesToUpdateRemote []string
	submodulesWithSpecificRefs := make(map[string]string)

	for _, path := range submodules {
		if ref, ok := specificRefs[path]; ok {
			submodulesWithSpecificRefs[path] = ref
		} else {
			submodulesToUpdateRemote = append(submodulesToUpdateRemote, path)
		}
	}

	// Process submodules with specific refs first by cd-ing into them and checking out the ref.
	for path, ref := range submodulesWithSpecificRefs {
		fmt.Printf("--- Processing submodule: %s -> %s ---\n", path, ref)
		submoduleDir := rootDir + "/" + path

		// Fetch all changes, including tags, from the remote.
		if err := runCommand(submoduleDir, "git", "fetch", "--all", "--tags"); err != nil {
			fmt.Printf("Error fetching in %s: %v\n", path, err)
			continue // Move to the next submodule on error.
		}

		// Checkout the specific commit, tag, or branch.
		fmt.Printf("Updating %s to specified ref: %s\n", path, ref)
		if err := runCommand(submoduleDir, "git", "checkout", ref); err != nil {
			fmt.Printf("Error checking out ref '%s' in %s: %v\n", ref, path, err)
		}
		fmt.Printf("--- Finished submodule: %s ---\n\n", path)
	}

	// Process submodules to be updated to the latest commit on their remote branch.
	if len(submodulesToUpdateRemote) > 0 {
		fmt.Println("--- Updating remaining submodules to latest ---")
		// The `git submodule update --remote` command is the correct way to update
		// submodules to the latest commit on their tracked branch. It correctly handles
		// the "detached HEAD" state where `git pull` would fail.
		args := []string{"submodule", "update", "--remote"}
		args = append(args, submodulesToUpdateRemote...)
		if err := runCommand(rootDir, "git", args...); err != nil {
			fmt.Printf("Error updating submodules to latest: %v\n", err)
		}
		fmt.Println("--- Finished updating remaining submodules ---")
	}

	fmt.Println("Submodule update process complete.")
}

// parseArgs parses the positional command-line arguments into a map of submodule paths to refs.
// Arguments are expected in the format: "submodule_path=ref" where ref is a commit, tag, or branch.
func parseArgs(args []string) map[string]string {
	targets := make(map[string]string)
	if len(args) > 0 {
		fmt.Println("Specific submodule updates:")
		for _, arg := range args {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				path := parts[0]
				ref := parts[1]
				targets[path] = ref
				fmt.Printf("  - %s -> %s\n", path, ref)
			} else {
				fmt.Printf("Warning: Ignoring invalid argument: %s\n", arg)
			}
		}
		fmt.Println("---------------------------------")
	}
	return targets
}

// getGitRootDir finds the root directory of the current Git repository.
func getGitRootDir() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getSubmodules returns a slice of strings, where each string is the path to a submodule.
func getSubmodules(rootDir string) ([]string, error) {
	var paths []string
	// Use `git submodule status` to list all submodules.
	cmd := exec.Command("git", "submodule", "status")
	cmd.Dir = rootDir
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// The output format is ` [commit] [path] ([branch])`. We just need the path.
		// The leading space is present on modified/uninitialized submodules.
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			paths = append(paths, fields[1])
		}
	}

	return paths, scanner.Err()
}

// runCommand is a helper function to execute a shell command in a specified directory
// and print its output to the console if verbose mode is enabled.
func runCommand(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir // Set the working directory for the command.

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Printf("-> Running in %s: %s %s\n", dir, name, strings.Join(args, " "))
		return cmd.Run()
	}

	// If not verbose, we still want to see errors from the command.
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Print the command's output only if there was an error.
		fmt.Print(string(output))
		return err
	}
	return nil
}
