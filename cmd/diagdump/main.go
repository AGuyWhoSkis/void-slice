package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"void-slice/internal/lint"
)

func main() {
	root := "testdata/golden"
	linter := lint.New()
	counts := map[string]map[string]int{}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		lower := strings.ToLower(path)
		if !strings.HasSuffix(lower, ".decl.xml") {
			ext := filepath.Ext(path)
			if ext != ".decl" && ext != ".entitydef" && ext != ".entities" && ext != ".cfg" {
				return nil
			}
		}
		src, _ := os.ReadFile(path)
		diags, _ := linter.Lint(path, src)
		rel, _ := filepath.Rel(root, path)
		for _, dg := range diags {
			if counts[dg.Code] == nil {
				counts[dg.Code] = map[string]int{}
			}
			counts[dg.Code][rel]++
		}
		return nil
	})
	codes := []string{}
	for c := range counts {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	for _, c := range codes {
		fmt.Printf("=== %s ===\n", c)
		files := []string{}
		for f := range counts[c] {
			files = append(files, f)
		}
		sort.Slice(files, func(i, j int) bool { return counts[c][files[i]] > counts[c][files[j]] })
		total := 0
		for _, f := range files {
			fmt.Printf("  %d  %s\n", counts[c][f], f)
			total += counts[c][f]
		}
		fmt.Printf("  total: %d\n", total)
	}
}
