package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/prompts"
)

const (
	statusSuccess = 0
	statusError   = 1
)

func main() {
	os.Exit(run())
}

var llm *ollama.LLM

func getLLMClient() (*ollama.LLM, error) {
	// singleton
	if llm != nil {
		return llm, nil
	}
	var err error
	llm, err = ollama.New(ollama.WithModel("llama3"))
	if err != nil {
		return nil, fmt.Errorf("error creating LLM: %w", err)
	}
	return llm, nil
}

func run() int {
	ctx := context.Background()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		os.Exit(statusSuccess)
	}()

	llm, err := getLLMClient()
	if err != nil {
		fmt.Println("Error creating LLM:", err)
		return statusError
	}

	var queryBuf [256]byte
	for {
		fmt.Fprint(os.Stdout, "prompt here: ")
		n, err := os.Stdin.Read(queryBuf[:])
		if err != nil {
			fmt.Println("Error reading from stdin:", err)
			return statusError
		}
		if n == 0 {
			return statusSuccess
		}
		if n == 1 {
			continue
		}

		page, err := getHTML(ctx, strings.TrimSpace(string(queryBuf[:n-1])))
		if err != nil {
			fmt.Println("Error getting HTML:", err)
			return statusError
		}

		prompt := prompts.NewPromptTemplate(
			"Please summarize the following webpage content in 3-5 sentences. Focus on the main points and key information.\n\nBody: {{.body}}",
			[]string{"body"},
		)
		result, err := prompt.Format(map[string]any{
			"body": page.Body,
		})
		if err != nil {
			fmt.Println("Error formatting prompt:", err)
			return statusError
		}

		completion, err := llms.GenerateFromSinglePrompt(ctx, llm, result)
		if err != nil {
			fmt.Println("Error generating completion:", err)
			return statusError
		}

		_, err = os.Stdout.Write([]byte("----------------------------------------\n" + completion + "\n----------------------------------------\n"))
		if err != nil {
			fmt.Println("Error writing to stdout:", err)
			return statusError
		}
		_, err = os.Stdout.Write([]byte("\n"))
		if err != nil {
			fmt.Println("Error writing to stdout:", err)
			return statusError
		}
	}
}

type HTMLPage struct {
	Body string
}

func getHTML(ctx context.Context, rURL string) (*HTMLPage, error) {
	u, err := url.Parse(rURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %w", err)
	}
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	request, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	bodyContent, err := extractBody(ctx, string(body))
	if err != nil {
		return nil, fmt.Errorf("error extracting body: %w", err)
	}
	return &HTMLPage{
		Body: bodyContent,
	}, nil
}

// extractBody extracts the main content of an HTML page.
func extractBody(ctx context.Context, html string) (string, error) {
	llm, err := getLLMClient()
	if err != nil {
		return "", fmt.Errorf("error getting LLM client: %w", err)
	}
	prompt := prompts.NewPromptTemplate(
		"Please extract the main content of the following webpage.\n\nBody: {{.body}}",
		[]string{"body"},
	)
	result, err := prompt.Format(map[string]any{
		"body": html,
	})
	if err != nil {
		return "", fmt.Errorf("error formatting prompt: %w", err)
	}

	completion, err := llms.GenerateFromSinglePrompt(ctx, llm, result)
	if err != nil {
		return "", fmt.Errorf("error generating completion: %w", err)
	}
	return completion, nil
}
