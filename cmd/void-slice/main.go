package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"void-slice/internal/core"
)

func main() {
	var (
		nonInteractive = flag.Bool("non-interactive", false, "Run without prompts")
		exportRoot     = flag.String("export-root", "", "Path to Export/ directory (contains game1/game2/...)")
		entry          = flag.String("entry", "", "Entry: canonical resource name (contains '/') OR exported .decl filename")
		outDir         = flag.String("out", "", "Destination output directory")
		maxDepth       = flag.Int("max-depth", 10, "Max traversal depth (hard cap)")
	)
	flag.Parse()

	reader := bufio.NewReader(os.Stdin)

	if !*nonInteractive {
		if *exportRoot == "" {
			guess := "./Export"
			if _, err := os.Stat(guess); err == nil {
				*exportRoot = guess
			} else {
				*exportRoot = prompt(reader, "Export root", "")
			}
		}
		if *entry == "" {
			*entry = prompt(reader, "Entry (canonical name with '/' OR exported filename)", "")
		}
		if *outDir == "" {
			def := defaultOutDir()
			v := prompt(reader, "Destination folder", def)
			if strings.TrimSpace(v) == "" {
				*outDir = def
			} else {
				*outDir = v
			}
		}
	} else {
		missing := []string{}
		if *exportRoot == "" {
			missing = append(missing, "--export-root")
		}
		if *entry == "" {
			missing = append(missing, "--entry")
		}
		if *outDir == "" {
			missing = append(missing, "--out")
		}
		if len(missing) > 0 {
			fmt.Fprintf(os.Stderr, "missing required flags in --non-interactive mode: %s\n", strings.Join(missing, ", "))
			os.Exit(2)
		}
	}

	absOut, err := filepath.Abs(*outDir)
	if err != nil {
		fatal("failed to resolve output dir", err)
	}

	// Safety check: destination exists and is non-empty
	if info, err := os.Stat(absOut); err == nil && info.IsDir() {
		entries, _ := os.ReadDir(absOut)
		if len(entries) > 0 {
			if *nonInteractive {
				fatal("destination folder exists and is not empty", nil)
			}
			resp := prompt(reader, "Destination exists and is not empty. Continue? (y/N)", "n")
			if strings.ToLower(strings.TrimSpace(resp)) != "y" {
				fmt.Println("aborted")
				return
			}
		}
	}

	if err := os.MkdirAll(absOut, 0o755); err != nil {
		fatal("failed to create output directory", err)
	}

	result, err := core.Run(core.Options{
		ExportRoot: *exportRoot,
		Entry:      *entry,
		OutDir:     absOut,
		MaxDepth:   *maxDepth,
	})
	if err != nil {
		fatal("error", err)
	}

	fmt.Println("OK")
	fmt.Printf("  roots processed:   %d\n", result.RootsProcessed)
	fmt.Printf("  visited nodes:     %d\n", result.VisitedNodes)
	fmt.Printf("  unresolved unique: %d\n", result.UnresolvedUnique)
	fmt.Printf("  output:            %s\n", absOut)

	if len(result.Warnings) > 0 {
		fmt.Printf("\nWarnings (%d):\n", len(result.Warnings))
		for i, w := range result.Warnings {
			if i >= 20 {
				fmt.Printf("  ... (%d more)\n", len(result.Warnings)-20)
				break
			}
			fmt.Printf("  - %s\n", w)
		}
	}

	if result.UnresolvedUnique > 0 {
		fmt.Printf("\nUnresolved (%d) (showing up to 50):\n", result.UnresolvedUnique)
		for i, u := range result.Unresolved {
			if i >= 50 {
				fmt.Printf("  ... (%d more)\n", result.UnresolvedUnique-50)
				break
			}
			fmt.Printf("  - %s\n", u)
		}
	}
}

func prompt(r *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func defaultOutDir() string {
	ts := time.Now().Format("20060102-150405")
	return "./void-slice-output-" + ts
}

func fatal(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	} else {
		fmt.Fprintln(os.Stderr, msg)
	}
	os.Exit(1)
}
