package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

var specialChars = []rune{'"', '\\'}

type Command struct {
	Exec string
	Args []string
}

// parseCommand parses the command given to the prompt.
func parseCommand(rawCmd string) *Command {
	var (
		tokens          []string
		prev            rune
		cur             strings.Builder
		seenSingleQuote bool
		seenDoubleQuote bool
	)

	// Handle special characters, single, and double quotes
	runes := []rune(rawCmd)
	for i := 0; i < len(runes); {
		switch runes[i] {
		case '\'':
			if seenDoubleQuote {
				cur.WriteRune(runes[i])
			} else {
				seenSingleQuote = !seenSingleQuote
			}

		case '"':
			if seenSingleQuote {
				cur.WriteRune(runes[i])
			} else {
				seenDoubleQuote = !seenDoubleQuote
			}

		case '\\':
			if !seenSingleQuote && i+1 < len(runes) && slices.Contains(specialChars, runes[i]) {
				i++
			}

			cur.WriteRune(runes[i])

		case ' ':
			seenQuote := seenDoubleQuote || seenSingleQuote
			if seenQuote {
				cur.WriteRune(runes[i])
			} else if prev != ' ' && cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur = strings.Builder{}
			}

		default:
			cur.WriteRune(runes[i])
		}

		prev = runes[i]
		i++
	}

	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}

	tokensLen := len(tokens)
	// Parsing failed, invalid command
	if len(tokens) < 1 {
		return nil
	}

	cmd := &Command{
		Exec: tokens[0],
	}
	if tokensLen > 1 {
		cmd.Args = tokens[1:]
	}

	return cmd
}

func executeExitCmd(cmd *Command) {
	if len(cmd.Args) <= 0 {
		os.Exit(0)
		return
	}

	// Parse the exit code
	exitCode, err := strconv.Atoi(cmd.Args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading exit code: ", err)
		exitCode = 1
	}
	os.Exit(exitCode)
}

func executeEchoCmd(cmd *Command) {
	fmt.Println(strings.Join(cmd.Args, " "))
}

func getExecutablePath(file string) (string, error) {
	// Look for executable files with "command" name
	// Get the path
	path, ok := os.LookupEnv("PATH")
	if !ok {
		fmt.Fprintf(os.Stderr, "'PATH' env is not set\n")
		os.Exit(1)
		return "", nil
	}

	// Get directory paths
	dirs := strings.SplitSeq(path, string(os.PathListSeparator))
	for dir := range dirs {
		// Read the directory
		entries, err := os.ReadDir(dir)
		if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to read directory: %v", err)
		}

		// Loop over directory items
		for _, entry := range entries {
			if entry.IsDir() { // Skip if directory, we need file
				continue
			}

			info, err := entry.Info()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to get file info: %v\n", err)
				continue
			}

			// Check if the file owner has executable permission on it
			// and is the file that we are looking for
			if entry.Name() == file && (info.Mode().Perm()&0100) != 0 {
				return fmt.Sprintf("%v/%v", dir, file), nil
			}
		}
	}

	return "", fmt.Errorf("%s: not found", file)
}

func executeTypeCmd(cmd *Command) {
	switch cmd.Args[0] {
	case "exit", "echo", "type", "pwd", "cd":
		fmt.Printf("%s is a shell builtin\n", cmd.Args[0])
	default:
		exePath, err := getExecutablePath(cmd.Args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}

		fmt.Printf("%v is %v\n", cmd.Args[0], exePath)
	}
}

func executePwdCmd() {
	curDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	fmt.Println(curDir)
}

func executeCdCmd(cmd *Command) {
	absPath, err := filepath.Abs(cmd.Args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// Handle tilde (home directory)
	if cmd.Args[0] == "~" {
		absPath = os.Getenv("HOME")
	}

	if err := os.Chdir(absPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "cd: %v: No such file or directory\n", strings.Join(cmd.Args, " "))
		} else {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
	}
}

func runProgram(cmd *Command) bool {
	_, err := getExecutablePath(cmd.Exec)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		return false
	}

	output, err := exec.Command(cmd.Exec, cmd.Args...).Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return false
	}

	fmt.Print(string(output))
	return true
}

func evaluateCommand(command string) {
	cmd := parseCommand(command)
	if cmd == nil {
		os.Exit(0)
		return
	}

	// Handle the "exit" builtin
	if strings.HasPrefix(command, "exit") {
		executeExitCmd(cmd)
	} else if strings.HasPrefix(command, "echo") {
		executeEchoCmd(cmd)
	} else if strings.HasPrefix(command, "type") {
		executeTypeCmd(cmd)
	} else if command == "pwd" {
		executePwdCmd()
	} else if strings.HasPrefix(command, "cd") {
		executeCdCmd(cmd)
	} else if !runProgram(cmd) {
		fmt.Println(command + ": command not found")
	}
}

func main() {
	for {
		fmt.Fprint(os.Stdout, "$ ")

		// Wait for user input
		command, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input: ", err)
			os.Exit(1)
		}

		evaluateCommand(command[:len(command)-1])
	}
}
