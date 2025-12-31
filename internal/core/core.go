package core

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"void-slice/internal/fsutil"
	"void-slice/internal/index"
	"void-slice/internal/parse"
)

type Options struct {
	ExportRoot string
	Entry      string
	OutDir     string
	MaxDepth   int
}

type Result struct {
	RootsProcessed int

	VisitedNodes int
	CopiedDecl   int
	CopiedXML    int

	UnresolvedUnique int
	Unresolved       []string

	Warnings []string
}

type node struct {
	Bucket string
	Name   string
	Depth  int
}

type resolved struct {
	Bucket   string
	Resource index.Resource
}

func Run(opts Options) (Result, error) {
	var res Result

	if strings.TrimSpace(opts.ExportRoot) == "" {
		return res, fmt.Errorf("ExportRoot is required")
	}
	if strings.TrimSpace(opts.Entry) == "" {
		return res, fmt.Errorf("Entry is required")
	}
	if strings.TrimSpace(opts.OutDir) == "" {
		return res, fmt.Errorf("OutDir is required")
	}
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 10
	}

	buckets, err := listBuckets(opts.ExportRoot)
	if err != nil {
		return res, fmt.Errorf("list buckets: %w", err)
	}
	if len(buckets) == 0 {
		return res, fmt.Errorf("no bucket directories found under export root: %s", opts.ExportRoot)
	}
	sort.Strings(buckets)

	byBucket := make(map[string]map[string]index.Resource, len(buckets))
	for _, b := range buckets {
		idx, err := index.Load(filepath.Join(opts.ExportRoot, b))
		if err != nil {
			return res, fmt.Errorf("load index for %s: %w", b, err)
		}
		byBucket[b] = idx
	}

	roots, warns, err := resolveEntry(opts.ExportRoot, buckets, byBucket, opts.Entry)
	if err != nil {
		return res, err
	}
	res.Warnings = append(res.Warnings, warns...)
	res.RootsProcessed = len(roots)

	visited := make(map[string]struct{})
	unresolvedSet := make(map[string]struct{})
	copied := make(map[string]struct{})

	for _, root := range roots {
		stack := []node{{Bucket: root.Bucket, Name: root.Name, Depth: 0}}

		for len(stack) > 0 {
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			key := n.Bucket + "::" + n.Name
			if _, ok := visited[key]; ok {
				continue
			}
			visited[key] = struct{}{}

			if n.Depth > opts.MaxDepth {
				res.Warnings = append(res.Warnings,
					fmt.Sprintf("depth limit exceeded (%d) at %s/%s", opts.MaxDepth, n.Bucket, n.Name))
				continue
			}

			r, ok := byBucket[n.Bucket][n.Name]
			if !ok {
				unresolvedSet[n.Name] = struct{}{}
				continue
			}

			if _, ok := copied[key]; !ok {
				if err := copyOne(opts.OutDir, opts.ExportRoot, n.Bucket, r); err != nil {
					return res, err
				}
				copied[key] = struct{}{}
			}

			declAbs := filepath.Join(opts.ExportRoot, n.Bucket, filepath.FromSlash(r.RelDeclPath))
			data, err := os.ReadFile(declAbs)
			if err != nil {
				res.Warnings = append(res.Warnings, fmt.Sprintf("failed to read %s: %v", declAbs, err))
				continue
			}

			toks := parse.Extract(string(data))
			for _, tok := range toks {
				rr, warn, ok := resolveTokenAcrossBuckets(buckets, byBucket, tok)
				if warn != "" {
					res.Warnings = append(res.Warnings, warn)
				}
				if !ok {
					unresolvedSet[tok] = struct{}{}
					continue
				}

				stack = append(stack, node{
					Bucket: rr.Bucket,
					Name:   rr.Resource.Name,
					Depth:  n.Depth + 1,
				})
			}
		}
	}

	res.VisitedNodes = len(visited)
	res.CopiedDecl = len(copied)
	res.CopiedXML = len(copied)

	res.UnresolvedUnique = len(unresolvedSet)
	for u := range unresolvedSet {
		res.Unresolved = append(res.Unresolved, u)
	}
	sort.Strings(res.Unresolved)

	return res, nil
}

func listBuckets(exportRoot string) ([]string, error) {
	ents, err := os.ReadDir(exportRoot)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

func resolveEntry(exportRoot string, buckets []string, byBucket map[string]map[string]index.Resource, entry string) ([]node, []string, error) {
	var warns []string
	entry = strings.TrimSpace(entry)

	// Canonical name path → scan XMLs directly
	if strings.Contains(entry, "/") {
		var roots []node

		for _, b := range buckets {
			bdir := filepath.Join(exportRoot, b)
			_ = filepath.WalkDir(bdir, func(p string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() || !strings.HasSuffix(p, ".decl.xml") {
					return nil
				}
				name, err := canonicalFromXML(p)
				if err != nil {
					return nil
				}
				if name == entry {
					roots = append(roots, node{Bucket: b, Name: name, Depth: 0})
				}
				if _, ok := byBucket[b][name]; !ok {
					byBucket[b][name] = index.Resource{
						Name:        name,
						RelDeclPath: strings.TrimSuffix(filepath.Base(p), ".xml"),
						RelXMLPath:  filepath.Base(p),
					}
				}

				return nil
			})
		}

		if len(roots) == 0 {
			return nil, warns, fmt.Errorf("entry canonical name not found in any bucket index: %s", entry)
		}
		if len(roots) > 1 {
			warns = append(warns, fmt.Sprintf("entry exists in multiple buckets; processing all: %s", entry))
		}
		return roots, warns, nil
	}

	// Exported filename case
	var roots []node
	for _, b := range buckets {
		declAbs := filepath.Join(exportRoot, b, entry)
		if _, err := os.Stat(declAbs); err != nil {
			continue
		}

		xmlAbs := declAbs + ".xml"
		canon, err := canonicalFromXML(xmlAbs)
		if err != nil {
			return nil, warns, fmt.Errorf("entry file found but could not read canonical name from xml (%s): %w", xmlAbs, err)
		}

		roots = append(roots, node{Bucket: b, Name: canon, Depth: 0})
	}

	if len(roots) == 0 {
		return nil, warns, fmt.Errorf("entry exported filename not found in any bucket directory: %s", entry)
	}
	if len(roots) > 1 {
		warns = append(warns, fmt.Sprintf("entry filename exists in multiple buckets; processing all: %s", entry))
	}
	return roots, warns, nil
}

func canonicalFromXML(xmlPath string) (string, error) {
	data, err := os.ReadFile(xmlPath)
	if err != nil {
		return "", err
	}

	type resourceInfo struct {
		XMLName xml.Name `xml:"ResourceInfo"`
		Name    string   `xml:"name,attr"`
	}

	var r resourceInfo
	if err := xml.Unmarshal(data, &r); err != nil {
		return "", err
	}

	if strings.TrimSpace(r.Name) == "" {
		return "", fmt.Errorf("missing ResourceInfo@name in %s", xmlPath)
	}

	return r.Name, nil
}

func resolveTokenAcrossBuckets(buckets []string, byBucket map[string]map[string]index.Resource, token string) (resolved, string, bool) {
	found := make([]resolved, 0, 2)
	for _, b := range buckets {
		if r, ok := byBucket[b][token]; ok {
			found = append(found, resolved{Bucket: b, Resource: r})
		}
	}
	if len(found) == 0 {
		return resolved{}, "", false
	}
	if len(found) == 1 {
		return found[0], "", true
	}
	warn := fmt.Sprintf("ambiguous reference %q found in multiple buckets; choosing %s", token, found[0].Bucket)
	return found[0], warn, true
}

func copyOne(outDir, exportRoot, bucket string, r index.Resource) error {
	srcDecl := filepath.Join(exportRoot, bucket, filepath.FromSlash(r.RelDeclPath))
	srcXML := filepath.Join(exportRoot, bucket, filepath.FromSlash(r.RelXMLPath))

	dstDecl := filepath.Join(outDir, bucket, filepath.FromSlash(r.RelDeclPath))
	dstXML := filepath.Join(outDir, bucket, filepath.FromSlash(r.RelXMLPath))

	if err := fsutil.CopyFile(srcDecl, dstDecl); err != nil {
		return err
	}
	if err := fsutil.CopyFile(srcXML, dstXML); err != nil {
		return err
	}
	return nil
}
