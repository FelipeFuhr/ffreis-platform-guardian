package main

import (
	"os"

	"github.com/ffreis/platform-guardian/cmd"
)

var execute = cmd.Execute
var exitFunc = os.Exit

func main() {
	code := execute()
	if code != 0 {
		exitFunc(code)
	}
}
