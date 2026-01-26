package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Instruction struct {
	Command string
	Args    []string
}

// parseInstruction parses the instruction given to the prompt.
func parseInstruction(rawInstruction string) *Instruction {
	tokens := strings.Split(rawInstruction, " ")
	tokensLen := len(tokens)
	if tokensLen < 1 {
		return nil
	}

	inst := &Instruction{
		Command: tokens[0],
	}
	if tokensLen > 1 {
		inst.Args = handleQuotes(strings.Join(tokens[1:], " "))
	}

	return inst
}

func handleQuotes(arg string) []string {
	var (
		args      []string
		prev      rune
		cur       strings.Builder
		seenQuote bool
	)

	runes := []rune(arg)
	for i := 0; i < len(runes); {
		switch runes[i] {
		case '\'':
			seenQuote = !seenQuote

		case ' ':
			if seenQuote {
				cur.WriteRune(runes[i])
			} else if prev != ' ' && cur.Len() > 0 {
				args = append(args, cur.String())
				cur = strings.Builder{}
			}

		default:
			cur.WriteRune(runes[i])
		}

		prev = runes[i]
		i++
	}

	if cur.Len() > 0 {
		args = append(args, cur.String())
	}

	return args
}

func executeExitCmd(inst *Instruction) {
	if len(inst.Args) <= 0 {
		os.Exit(0)
		return
	}

	// Parse the exit code
	exitCode, err := strconv.Atoi(inst.Args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading exit code: ", err)
		exitCode = 1
	}
	os.Exit(exitCode)
}

func executeEchoCmd(inst *Instruction) {
	fmt.Println(strings.Join(inst.Args, " "))
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

func executeTypeCmd(inst *Instruction) {
	switch inst.Args[0] {
	case "exit", "echo", "type", "pwd", "cd":
		fmt.Printf("%s is a shell builtin\n", inst.Args[0])
	default:
		exePath, err := getExecutablePath(inst.Args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}

		fmt.Printf("%v is %v\n", inst.Args, exePath)
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

func executeCdCmd(inst *Instruction) {
	absPath, err := filepath.Abs(inst.Args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// Handle tilde (home directory)
	if inst.Args[0] == "~" {
		absPath = os.Getenv("HOME")
	}

	if err := os.Chdir(absPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "cd: %v: No such file or directory\n", strings.Join(inst.Args, " "))
		} else {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
	}
}

func runProgram(inst *Instruction) bool {
	_, err := getExecutablePath(inst.Command)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		return false
	}

	cmd := exec.Command(inst.Command, inst.Args...)
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return false
	}

	fmt.Print(string(output))
	return true
}

func evaluateCommand(command string) {
	inst := parseInstruction(command)
	if inst == nil {
		os.Exit(0)
		return
	}

	// Handle the "exit" builtin
	if strings.HasPrefix(command, "exit") {
		executeExitCmd(inst)
	} else if strings.HasPrefix(command, "echo") {
		executeEchoCmd(inst)
	} else if strings.HasPrefix(command, "type") {
		executeTypeCmd(inst)
	} else if command == "pwd" {
		executePwdCmd()
	} else if strings.HasPrefix(command, "cd") {
		executeCdCmd(inst)
	} else if !runProgram(inst) {
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
