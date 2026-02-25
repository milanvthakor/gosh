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
	Exec            string
	Args            []string
	OutputRedirFile string
}

// parseCommand parses the command given to the prompt.
func parseCommand(rawCmd string) *Command {
	var (
		tokens          []string
		prev            rune
		cur             strings.Builder
		seenSingleQuote bool
		seenDoubleQuote bool
		seenOutputRedir bool
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
			if !seenSingleQuote && i+1 < len(runes) && slices.Contains(specialChars, runes[i+1]) {
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

		case '>':
			if seenDoubleQuote || seenSingleQuote || (prev != ' ' && prev != '1') {
				cur.WriteRune(runes[i])
				continue
			}

			cur = strings.Builder{}
			seenOutputRedir = true

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
		if seenOutputRedir {
			cmd.OutputRedirFile = tokens[tokensLen-1]
			cmd.Args = tokens[1 : tokensLen-1]
		} else {
			cmd.Args = tokens[1:]
		}
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

func executeEchoCmd(cmd *Command) string {
	return strings.Join(cmd.Args, " ") + "\n"
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

func executeTypeCmd(cmd *Command) string {
	switch cmd.Args[0] {
	case "exit", "echo", "type", "pwd", "cd":
		return fmt.Sprintf("%s is a shell builtin\n", cmd.Args[0])
	default:
		exePath, err := getExecutablePath(cmd.Args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return ""
		}

		return fmt.Sprintf("%v is %v\n", cmd.Args[0], exePath)
	}
}

func executePwdCmd() string {
	curDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return ""
	}

	return curDir + "\n"
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

func runProgram(cmd *Command) string {
	_, err := getExecutablePath(cmd.Exec)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		return ""
	}

	output, err := exec.Command(cmd.Exec, cmd.Args...).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			fmt.Fprint(os.Stderr, string(ee.Stderr))
		} else {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		return ""
	}

	return string(output)
}

func evaluateCommand(rawCmd string) {
	cmd := parseCommand(rawCmd)
	if cmd == nil {
		os.Exit(0)
		return
	}

	var output string

	// Handle builtins based on parsed command name
	switch cmd.Exec {
	case "exit":
		executeExitCmd(cmd)
	case "echo":
		output = executeEchoCmd(cmd)
	case "type":
		output = executeTypeCmd(cmd)
	case "pwd":
		output = executePwdCmd()
	case "cd":
		executeCdCmd(cmd)
	default:
		output = runProgram(cmd)
	}

	if output == "" {
		return
	}

	// Handle output
	if cmd.OutputRedirFile != "" {
		if err := os.WriteFile(cmd.OutputRedirFile, []byte(output), os.ModePerm); err != nil {
			fmt.Fprintf(os.Stderr, "failed to redirect ouptut: %v\n", err)
		}
	} else {
		fmt.Print(output)
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
