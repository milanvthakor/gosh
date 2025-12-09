package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func executeExitCmd(command string) {
	// Get the optional exit code
	tokens := strings.Split(command, " ")
	if len(tokens) <= 1 {
		os.Exit(0)
	}
	// Parse the exit code
	exitCode, err := strconv.Atoi(strings.Split(command, " ")[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading exit code: ", err)
		exitCode = 1
	}
	os.Exit(exitCode)
}

func executeEchoCmd(command string) {
	tokens := strings.Split(command, " ")
	if len(tokens) < 1 {
		os.Exit(0)
	}

	fmt.Println(strings.Join(tokens[1:], " "))
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

func executeTypeCmd(command string) {
	tokens := strings.Split(command, " ")[1:]
	argCmd := strings.Join(tokens, " ")
	switch argCmd {
	case "exit", "echo", "type", "pwd", "cd":
		fmt.Printf("%s is a shell builtin\n", argCmd)
	default:
		exePath, err := getExecutablePath(argCmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}

		fmt.Printf("%v is %v\n", argCmd, exePath)
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

func executeCdCmd(command string) {
	newDir := strings.Split(command, " ")[1]
	if err := os.Chdir(newDir); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "cd: %v: No such file or directory\n", newDir)
		} else {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		return
	}

	if err := os.Chdir(newDir); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
}

func runProgram(command string) bool {
	tokens := strings.Split(command, " ")
	argCmd := tokens[0]

	_, err := getExecutablePath(argCmd)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		return false
	}

	cmd := exec.Command(tokens[0], tokens[1:]...)
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return false
	}

	fmt.Print(string(output))
	return true
}

func evaluateCommand(command string) {
	// Handle the "exit" builtin
	if strings.HasPrefix(command, "exit") {
		executeExitCmd(command)
	} else if strings.HasPrefix(command, "echo") {
		executeEchoCmd(command)
	} else if strings.HasPrefix(command, "type") {
		executeTypeCmd(command)
	} else if command == "pwd" {
		executePwdCmd()
	} else if strings.HasPrefix(command, "cd") {
		executeCdCmd(command)
	} else if !runProgram(command) {
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
