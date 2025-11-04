package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprint(os.Stdout, "$ ")
	var inp string
	fmt.Fscan(os.Stdin, &inp)
	fmt.Fprint(os.Stdout, inp+": command not found")
}
