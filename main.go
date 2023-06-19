package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/c-bata/go-prompt"
	"github.com/mkXultra/gpt-lamp/lib"
)

var currentDir string
var chatGptSwitch bool

func initShell() {

	var err error
	currentDir, err = os.Getwd()
	if err != nil {
		log.Fatalf("error getting current directory: %v", err)
	}
	chatGptSwitch = false
}

func executor(in string) {
	if strings.HasPrefix(in, "exit") {
		fmt.Println("Bye!")
		os.Exit(0)
	}
	if strings.HasPrefix(in, "on") {
		chatGptSwitch = true
		return
	} else if strings.HasPrefix(in, "off") {
		chatGptSwitch = false
		return
	}

	if strings.HasPrefix(in, "cd ") {
		dir := strings.TrimPrefix(in, "cd ")
		err := os.Chdir(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERROR", err.Error())
			return
		}
		currentDir, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERROR", err.Error())
			return
		}
	} else {
		var stderr bytes.Buffer
		cmd := exec.Command("bash", "-c", in)
		cmd.Stderr = &stderr
		cmd.Stdout = os.Stdout
		cmd.Dir = currentDir
		err := cmd.Run()
		if err != nil {
			fmt.Println(stderr.String())
			if exiterr, ok := err.(*exec.ExitError); ok {
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					if chatGptSwitch {
						fmt.Println("Error thinking gpt")
						// lib.HowToFix("go", status.ExitStatus(), stderr.String(), "JP")
						lib.HowToFixStream("go", status.ExitStatus(), stderr.String(), "JP")
					}
				}
			} else {
				fmt.Fprintln(os.Stderr, "Cmd failed:", err)
			}
			return
		}
	}
}

func completer(in prompt.Document) []prompt.Suggest {
	suggestions := []prompt.Suggest{
		{Text: "on", Description: "Turn on the chatgpt analyzer"},
		{Text: "off", Description: "Turn off the chatgpt analyzer"},
	}
	return prompt.FilterHasPrefix(suggestions, in.GetWordBeforeCursor(), true)
}

func main() {
	initShell()

	p := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix(currentDir+"> "),
		prompt.OptionTitle("gpt-lamp"),
	)
	p.Run()
}
