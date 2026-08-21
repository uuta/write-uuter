package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/uuta/write-uuter/internal/app"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "__agent" {
		if err := app.RunAgent(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) < 2 || os.Args[1] != "run" {
		fmt.Fprintln(os.Stderr, "usage: write-uuter run --brief <path> --run-dir <new-directory>")
		os.Exit(2)
	}

	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	brief := flags.String("brief", "", "path to the Markdown brief")
	runDir := flags.String("run-dir", "", "new directory for run artifacts")
	codex := flags.String("codex", "codex", "Codex CLI executable path")
	tmux := flags.String("tmux", "tmux", "tmux executable path")
	timeout := flags.Duration("timeout", 10*time.Minute, "timeout for each agent contract")
	promptsDir := flags.String("prompts-dir", "", "directory containing version-controlled role prompts")
	if err := flags.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(2)
	}
	promptsDirSet := false
	flags.Visit(func(parsed *flag.Flag) {
		if parsed.Name == "prompts-dir" {
			promptsDirSet = true
		}
	})

	err := app.Run(app.Config{
		BriefPath:       *brief,
		RunDir:          *runDir,
		CodexExecutable: *codex,
		TmuxExecutable:  *tmux,
		AgentTimeout:    *timeout,
		PromptsDir:      *promptsDir,
		PromptsDirSet:   promptsDirSet,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
