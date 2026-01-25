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
	// RawArg is actual argument passed to the prompt without any parsing for quotes
	RawArg string
	Args   string
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
		inst.RawArg = strings.Join(tokens[1:], " ")
	}

	inst.Args = handleSingleQuote(inst.RawArg)
	return inst
}

func handleSingleQuote(arg string) string {
	var (
		newArg         strings.Builder
		prev           rune
		hasSingleQuote bool
	)
	for _, r := range arg {
		if r == '\'' {
			hasSingleQuote = !hasSingleQuote
		} else if !hasSingleQuote && r == ' ' && prev == ' ' { // Ignore more than one space if not inside single quotes
			continue
		} else {
			prev = r
			newArg.WriteRune(r)
		}
	}

	return newArg.String()
}

func executeExitCmd(inst *Instruction) {
	if len(inst.Args) <= 0 {
		os.Exit(0)
		return
	}

	// Parse the exit code
	exitCode, err := strconv.Atoi(inst.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading exit code: ", err)
		exitCode = 1
	}
	os.Exit(exitCode)
}

func executeEchoCmd(inst *Instruction) {
	fmt.Println(inst.Args)
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
	switch inst.Args {
	case "exit", "echo", "type", "pwd", "cd":
		fmt.Printf("%s is a shell builtin\n", inst.Args)
	default:
		exePath, err := getExecutablePath(inst.Args)
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
	absPath, err := filepath.Abs(inst.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// Handle tilde (home directory)
	if inst.Args == "~" {
		absPath = os.Getenv("HOME")
	}

	if err := os.Chdir(absPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "cd: %v: No such file or directory\n", inst.Args)
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

	// Handle single quotes separated string
	args := strings.Split(inst.RawArg, "' '")
	for i := range args {
		args[i] = strings.Trim(args[i], "'")
	}

	cmd := exec.Command(inst.Command, args...)
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
