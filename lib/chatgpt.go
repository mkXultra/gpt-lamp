package lib

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/mkXultra/gpt-lamp/prompt"
)

type Payload struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// ChatGPTResponseType example
//
//	{
//	   "id":"chatcmpl-abc123",
//	   "object":"chat.completion",
//	   "created":1677858242,
//	   "model":"gpt-3.5-turbo-0301",
//	   "usage":{
//	      "prompt_tokens":13,
//	      "completion_tokens":7,
//	      "total_tokens":20
//	   },
//	   "choices":[
//	      {
//	         "message":{
//	            "role":"assistant",
//	            "content":"\n\nThis is a test!"
//	         },
//	         "finish_reason":"stop",
//	         "index":0
//	      }
//	   ]
//	}
type ChatGPTResponseType struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
	} `json:"choices"`
}

// ChatGPTStreamResponseType example
// {"id":"chatcmpl-7TDdDGLWPD9tXbv2do8tWJD4ihcF8","object":"chat.completion.chunk","created":1687198267,"model":"gpt-3.5-turbo-0301","choices":[{"delta":{"content":"します"},"index":0,"finish_reason":null}]}{chatcmpl-7TDdDGLWPD9tXbv2do8tWJD4ihcF8 chat.completion.chunk 1687198267 gpt-3.5-turbo-0301 [{{} 0 }]}
type ChatGPTStreamResponseType struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

const COMPLETIONS_URL = "https://api.openai.com/v1/chat/completions"

func PostMessageStream(data Payload, f func(*bufio.Reader)) (*bufio.Reader, error) {
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", COMPLETIONS_URL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	f(reader)

	// fmt.Println("reader", reader)
	// for {
	// 	line, err := reader.ReadBytes('\n')
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	fmt.Printf("Received: %s", line)
	// }

	return reader, nil
}

func PostMessage(data Payload) ([]byte, error) {
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", COMPLETIONS_URL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

func HowToFix(lang string, errCode int, errMsg string, showLang string) {
	data := Payload{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{
				Role: "system",
				// Content: "You are super programmer.\nAnalyze error messages step by step.\n suggest multiple options on how to solve the problem. The output language should be Japanese.",
				// Content: fmt.Sprintf("You are super programmer.\nAnalyze error messages step by step.\n suggest multiple options on how to solve the problem. The output language should be %s\n", showLang),
				Content: fmt.Sprintf(prompt.HOW_TO_FIX_SYSTEM_PROMPT, showLang),
			},
			{
				Role: "user",
				// Content: fmt.Sprintf("# error code\n%d\n # error message\n%s\n", errCode, errMsg),
				Content: fmt.Sprintf(prompt.HOW_TO_FIX_MESSAGE, errCode, errMsg),
			},
		},
		Temperature: 0,
		Stream:      false,
	}

	resp, err := PostMessage(data)
	if err != nil {
		log.Fatalln(err)
	}
	// resp is ChatGPTResponseType
	// convert byte to ChatGPTResponseType
	var chatGPTResponse ChatGPTResponseType
	err = json.Unmarshal(resp, &chatGPTResponse)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(chatGPTResponse.Choices[0].Message.Content)
}

func HowToFixStream(lang string, errCode int, errMsg string, showLang string) {
	data := Payload{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{
				Role: "system",
				// Content: "You are super programmer.\nAnalyze error messages step by step.\n suggest multiple options on how to solve the problem. The output language should be Japanese.",
				// Content: fmt.Sprintf("You are super programmer.\nAnalyze error messages step by step.\n suggest multiple options on how to solve the problem. The output language should be %s\n", showLang),
				Content: fmt.Sprintf(prompt.HOW_TO_FIX_SYSTEM_PROMPT, showLang),
			},
			{
				Role: "user",
				// Content: fmt.Sprintf("# error code\n%d\n # error message\n%s\n", errCode, errMsg),
				Content: fmt.Sprintf(prompt.HOW_TO_FIX_MESSAGE, errCode, errMsg),
			},
		},
		Temperature: 0,
		Stream:      true,
	}

	PostMessageStream(data, func(reader *bufio.Reader) {
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				log.Fatalln(err)
				return
			}
			jsonData := strings.Replace(string(line), "data: ", "", 1)
			jsonData = strings.TrimSpace(jsonData)

			if jsonData == "" {
				continue
			}

			var chatGPTStreamResponse ChatGPTStreamResponseType
			err = json.Unmarshal([]byte(jsonData), &chatGPTStreamResponse)
			if err != nil {
				continue
			}
			if len(chatGPTStreamResponse.Choices) == 0 && chatGPTStreamResponse.Choices[0].Delta.Content == "" {
				continue
			}
			fmt.Print(chatGPTStreamResponse.Choices[0].Delta.Content)
		}
	})
	fmt.Println("")
}
