package main

import (
	"bufio"
	"fmt"
	"os"
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

func executeTypeCmd(command string) {
	tokens := strings.Split(command, " ")[1:]
	argCmd := strings.Join(tokens, " ")
	switch argCmd {
	case "exit", "echo", "type":
		fmt.Printf("%s is a shell builtin\n", argCmd)
	default:
		fmt.Printf("%s: not found\n", argCmd)
	}
}

func evaluateCommand(command string) {
	// Handle the "exit" builtin
	if strings.HasPrefix(command, "exit") {
		executeExitCmd(command)
	} else if strings.HasPrefix(command, "echo") {
		executeEchoCmd(command)
	} else if strings.HasPrefix(command, "type") {
		executeTypeCmd(command)
	} else {
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
