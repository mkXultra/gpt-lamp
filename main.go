package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/c-bata/go-prompt"
	"github.com/mkXultra/gpt-lamp/lib"
)

const version = "0.0.1"

var currentDir string
var chatGptSwitch bool

// example config file
// {
// 	apiKey: "OPENAI_API_KEY",
// }

type Config struct {
	ApiKey           string `json:"apiKey"`
	DefaultGptSwitch bool   `json:"defaultGptSwitch"`
	GptModel         string `json:"gptModel"`
}

func initShell() {

	var err error
	currentDir, err = os.Getwd()
	if err != nil {
		log.Fatalf("error getting current directory: %v", err)
	}
	chatGptSwitch = false
	apikey := os.Getenv("OPENAI_API_KEY")
	var config Config
	if apikey == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}

		confDirPath := filepath.Join(homeDir, ".gpt-lamp")
		// Check if the directory exists, if not, create it
		if _, err := os.Stat(confDirPath); os.IsNotExist(err) {
			err := os.MkdirAll(confDirPath, 0755)
			if err != nil {
				panic(err)
			}
		}

		// conf.jsonへのパスを作成
		confPath := filepath.Join(homeDir, ".gpt-lamp", "conf.json")

		// conf.jsonを読み込む
		confData, err := ioutil.ReadFile(confPath)
		if err != nil {
			// Check if the file exists
			_, err := os.Stat(confPath)
			if os.IsNotExist(err) {
				// If the file does not exist, create it with a default configuration
				defaultConfig := Config{
					ApiKey:           "Your default API Key",
					DefaultGptSwitch: true,
					GptModel:         "gpt-3.5-turbo",
				}
				configBytes, err := json.MarshalIndent(defaultConfig, "", "  ")
				if err != nil {
					panic(err)
				}

				// Write the default configuration to the file
				err = ioutil.WriteFile(confPath, configBytes, 0644)
				if err != nil {
					panic(err)
				}
				fmt.Println("Default configuration file created.")
				return
			} else if err != nil {
				// If there was an error other than "file does not exist", panic
				panic(err)
			}
		}

		// JSONを構造体にデコード
		if err := json.Unmarshal(confData, &config); err != nil {
			panic(err)
		}
		os.Setenv("OPENAI_API_KEY", config.ApiKey)
		os.Setenv("GPT_MODEL", config.GptModel)
		if config.DefaultGptSwitch {
			chatGptSwitch = true
		}
	}
	// fmt.Println("apikey is set to:", os.Getenv("OPENAI_API_KEY"))
	fmt.Println("gpt model:", os.Getenv("GPT_MODEL"))
	fmt.Println("gptswitch", chatGptSwitch)

}

type Queue struct {
	lines []string
	max   int
}

func NewQueue(max int) *Queue {
	return &Queue{
		lines: make([]string, 0, max),
		max:   max,
	}
}

func (q *Queue) Add(line string) {
	q.lines = append(q.lines, line)
	if len(q.lines) > q.max {
		q.lines = q.lines[1:]
	}
}

func (q *Queue) Get() []string {
	return q.lines
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
		// var stderr bytes.Buffer
		// var stdout bytes.Buffer
		cmd := exec.Command("bash", "-c", in)
		// cmd.Stderr = &stderr
		stderr, _ := cmd.StderrPipe()
		// cmd.Stdout = &stdout
		stdout, _ := cmd.StdoutPipe()
		cmd.Dir = currentDir
		cmd.Start()
		// err := cmd.Run()
		outQueue := NewQueue(30) // stdoutの最新の10行を保持
		errQueue := NewQueue(10) // stderrの最新の10行を保持

		// stdoutの処理
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				fmt.Println(line)  // 行を表示
				outQueue.Add(line) // 行をキューに追加
			}
			if err := scanner.Err(); err != nil {
				fmt.Fprintln(os.Stderr, "reading standard output:", err)
			}
		}()

		// stderrの処理
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()
				fmt.Fprintln(os.Stderr, line) // エラーメッセージを表示
				errQueue.Add(line)            // エラーメッセージをキューに追加
			}
			if err := scanner.Err(); err != nil {
				fmt.Fprintln(os.Stderr, "reading standard error:", err)
			}
		}()

		err := cmd.Wait()
		if err != nil {
			errMsgLines := errQueue.Get()
			errMsgs := strings.Join(errMsgLines, "\n")
			outMsgLines := outQueue.Get()
			outMsgs := strings.Join(outMsgLines, "\n")

			fmt.Println("---------error------------")
			fmt.Println(errMsgs)
			fmt.Println("---------stdout------------")
			fmt.Println(outMsgs)
			if exiterr, ok := err.(*exec.ExitError); ok {
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					if chatGptSwitch {
						fmt.Println("Error thinking gpt")
						// lib.HowToFix("go", status.ExitStatus(), stderr.String(), "JP")
						lib.HowToFixStream("go", in, status.ExitStatus(), errMsgs, outMsgs, "JP")
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
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Println("\t-version\n\t\tprint version information")
		fmt.Println("\tinit\n\t\tinitialize the gpt-lamp")
		flag.PrintDefaults()
	}
	var cmdversion = flag.Bool("version", false, "print version information")
	fmt.Println("gpt-lamp version", version)
	flag.Parse()
	if *cmdversion {
		fmt.Sprintf("Version %s", version)
		os.Exit(0)
	}
	if len(flag.Args()) > 0 {
		switch flag.Arg(0) {
		case "init":
			fmt.Println("Initialization...")
			initShell()
			os.Exit(0)
		}
	}

	initShell()

	p := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix(currentDir+"> "),
		prompt.OptionTitle("gpt-lamp"),
	)
	p.Run()
}
