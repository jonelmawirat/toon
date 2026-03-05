package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jonelmawirat/toon"
)

type fixture struct {
	Version     string     `json:"version"`
	Category    string     `json:"category"`
	Description string     `json:"description"`
	Tests       []testCase `json:"tests"`
}

type testCase struct {
	Name        string          `json:"name"`
	Input       json.RawMessage `json:"input"`
	Expected    json.RawMessage `json:"expected"`
	ShouldError bool            `json:"shouldError"`
	Options     testOptions     `json:"options"`
	SpecSection string          `json:"specSection"`
}

type testOptions struct {
	Delimiter    *string `json:"delimiter"`
	Indent       *int    `json:"indent"`
	Strict       *bool   `json:"strict"`
	KeyFolding   *string `json:"keyFolding"`
	FlattenDepth *int    `json:"flattenDepth"`
	ExpandPaths  *string `json:"expandPaths"`
}

type sectionStats struct {
	Total int
	Pass  int
	Fail  int
}

type failure struct {
	File     string
	Category string
	Section  string
	TestName string
	Reason   string
}

func main() {
	var fixturesDir string
	var download bool
	var workDir string
	var tarURL string

	flag.StringVar(&fixturesDir, "fixtures-dir", "", "path to spec tests/fixtures directory")
	flag.BoolVar(&download, "download", true, "download upstream spec fixtures when -fixtures-dir is empty")
	flag.StringVar(&workDir, "work-dir", filepath.Join(os.TempDir(), "toon-spec-main"), "working directory used for downloaded upstream spec archive")
	flag.StringVar(&tarURL, "tar-url", "https://codeload.github.com/toon-format/spec/tar.gz/refs/heads/main", "upstream spec tarball URL")
	flag.Parse()

	if fixturesDir == "" {
		if !download {
			fmt.Fprintln(os.Stderr, "ERROR: set -fixtures-dir or enable -download")
			os.Exit(2)
		}
		if err := downloadAndExtractSpecTar(tarURL, workDir); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: download fixtures: %v\n", err)
			os.Exit(1)
		}
		fixturesDir = filepath.Join(workDir, "tests", "fixtures")
	}

	if err := run(fixturesDir); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run(fixturesDir string) error {
	fixtureFiles, err := filepath.Glob(filepath.Join(fixturesDir, "*", "*.json"))
	if err != nil {
		return fmt.Errorf("glob fixtures: %w", err)
	}
	sort.Strings(fixtureFiles)
	if len(fixtureFiles) == 0 {
		return fmt.Errorf("no fixture files found under %s", fixturesDir)
	}

	section := map[string]*sectionStats{}
	category := map[string]*sectionStats{}
	byFile := map[string]*sectionStats{}
	fails := make([]failure, 0)

	for _, fpath := range fixtureFiles {
		data, err := os.ReadFile(fpath)
		if err != nil {
			fails = append(fails, failure{File: fpath, Category: "load", Section: "internal", TestName: "<fixture-read>", Reason: err.Error()})
			continue
		}

		var fx fixture
		if err := json.Unmarshal(data, &fx); err != nil {
			fails = append(fails, failure{File: fpath, Category: "load", Section: "internal", TestName: "<fixture-parse>", Reason: err.Error()})
			continue
		}

		for _, tc := range fx.Tests {
			sec := normalizeSection(tc.SpecSection)
			inc(section, sec)
			inc(category, fx.Category)
			inc(byFile, filepath.Base(fpath))

			if err := runCase(fx.Category, tc); err != nil {
				section[sec].Fail++
				category[fx.Category].Fail++
				byFile[filepath.Base(fpath)].Fail++
				fails = append(fails, failure{
					File:     fpath,
					Category: fx.Category,
					Section:  sec,
					TestName: tc.Name,
					Reason:   err.Error(),
				})
				continue
			}

			section[sec].Pass++
			category[fx.Category].Pass++
			byFile[filepath.Base(fpath)].Pass++
		}
	}

	fmt.Println("=== OVERALL ===")
	total, pass, fail := totals(section)
	fmt.Printf("total=%d pass=%d fail=%d\n", total, pass, fail)
	fmt.Println()

	fmt.Println("=== BY CATEGORY ===")
	for _, k := range sortedKeys(category) {
		st := category[k]
		fmt.Printf("%s\ttotal=%d\tpass=%d\tfail=%d\n", k, st.Total, st.Pass, st.Fail)
	}
	fmt.Println()

	fmt.Println("=== BY SPEC SECTION ===")
	for _, k := range sortedSectionKeys(section) {
		st := section[k]
		fmt.Printf("%s\ttotal=%d\tpass=%d\tfail=%d\n", k, st.Total, st.Pass, st.Fail)
	}
	fmt.Println()

	fmt.Println("=== BY FIXTURE FILE ===")
	for _, k := range sortedKeys(byFile) {
		st := byFile[k]
		fmt.Printf("%s\ttotal=%d\tpass=%d\tfail=%d\n", k, st.Total, st.Pass, st.Fail)
	}
	fmt.Println()

	fmt.Println("=== FAILURES ===")
	if len(fails) == 0 {
		fmt.Println("none")
		return nil
	}
	for i, f := range fails {
		fmt.Printf("%d) %s | %s | %s | %s\n", i+1, filepath.Base(f.File), f.Section, f.Category, f.TestName)
		fmt.Printf("   reason: %s\n", f.Reason)
	}
	return fmt.Errorf("conformance failures: %d", len(fails))
}

func runCase(cat string, tc testCase) error {
	switch cat {
	case "encode":
		return runEncode(tc)
	case "decode":
		return runDecode(tc)
	default:
		return fmt.Errorf("unknown fixture category %q", cat)
	}
}

func runEncode(tc testCase) error {
	in, err := parseJSONRawToToonValue(tc.Input)
	if err != nil {
		return fmt.Errorf("input parse: %w", err)
	}
	opts, err := encoderOpts(tc.Options)
	if err != nil {
		return fmt.Errorf("options: %w", err)
	}

	out, err := toon.Marshal(in, opts...)
	if tc.ShouldError {
		if err == nil {
			return fmt.Errorf("expected encode error, got success: %q", string(out))
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("unexpected encode error: %w", err)
	}

	expected, err := parseJSONString(tc.Expected)
	if err != nil {
		return fmt.Errorf("expected parse: %w", err)
	}
	if string(out) != expected {
		return fmt.Errorf("output mismatch\n  got:      %q\n  expected: %q", string(out), expected)
	}
	return nil
}

func runDecode(tc testCase) error {
	input, err := parseJSONString(tc.Input)
	if err != nil {
		return fmt.Errorf("input parse: %w", err)
	}
	opts, err := decoderOpts(tc.Options)
	if err != nil {
		return fmt.Errorf("options: %w", err)
	}

	got, err := toon.Unmarshal([]byte(input), opts...)
	if tc.ShouldError {
		if err == nil {
			return fmt.Errorf("expected decode error, got success: %#v", got)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("unexpected decode error: %w", err)
	}

	want, err := parseJSONRawToToonValue(tc.Expected)
	if err != nil {
		return fmt.Errorf("expected parse: %w", err)
	}
	if !equalValue(got, want) {
		return fmt.Errorf("value mismatch\n  got:  %#v\n  want: %#v", got, want)
	}
	return nil
}

func encoderOpts(o testOptions) ([]toon.EncoderOption, error) {
	opts := make([]toon.EncoderOption, 0, 6)

	delim := ","
	if o.Delimiter != nil {
		delim = *o.Delimiter
	}
	d, err := parseDelimiter(delim)
	if err != nil {
		return nil, err
	}
	opts = append(opts, toon.WithArrayDelimiter(d), toon.WithDocDelimiter(d))

	if o.Indent != nil {
		opts = append(opts, toon.WithEncoderIndent(*o.Indent))
	}

	folding := "off"
	if o.KeyFolding != nil {
		folding = *o.KeyFolding
	}
	switch folding {
	case "off":
		opts = append(opts, toon.WithKeyFolding(toon.KeyFoldingOff))
	case "safe":
		opts = append(opts, toon.WithKeyFolding(toon.KeyFoldingSafe))
	default:
		return nil, fmt.Errorf("invalid keyFolding %q", folding)
	}

	if o.FlattenDepth != nil {
		opts = append(opts, toon.WithFlattenDepth(*o.FlattenDepth))
	}

	return opts, nil
}

func decoderOpts(o testOptions) ([]toon.DecoderOption, error) {
	opts := make([]toon.DecoderOption, 0, 4)

	strict := true
	if o.Strict != nil {
		strict = *o.Strict
	}
	opts = append(opts, toon.WithStrict(strict))

	if o.Indent != nil {
		opts = append(opts, toon.WithDecoderIndent(*o.Indent))
	}

	expand := "off"
	if o.ExpandPaths != nil {
		expand = *o.ExpandPaths
	}
	switch expand {
	case "off":
		opts = append(opts, toon.WithExpandPaths(toon.ExpandPathsOff))
	case "safe":
		opts = append(opts, toon.WithExpandPaths(toon.ExpandPathsSafe))
	default:
		return nil, fmt.Errorf("invalid expandPaths %q", expand)
	}

	return opts, nil
}

func parseDelimiter(s string) (toon.Delimiter, error) {
	switch s {
	case ",":
		return toon.Comma, nil
	case "\t":
		return toon.Tab, nil
	case "|":
		return toon.Pipe, nil
	default:
		return 0, fmt.Errorf("invalid delimiter %q", s)
	}
}

func parseJSONString(raw json.RawMessage) (string, error) {
	var s string
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&s); err != nil {
		return "", err
	}
	return s, nil
}

func parseJSONRawToToonValue(raw json.RawMessage) (toon.Value, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	return readJSONValue(dec)
}

func readJSONValue(dec *json.Decoder) (toon.Value, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}

	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			obj := toon.Object{Members: make([]toon.Member, 0, 8)}
			for dec.More() {
				ktok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := ktok.(string)
				if !ok {
					return nil, fmt.Errorf("non-string key token %T", ktok)
				}
				v, err := readJSONValue(dec)
				if err != nil {
					return nil, err
				}
				obj.Members = append(obj.Members, toon.Member{Key: key, Value: v})
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return obj, nil

		case '[':
			arr := make(toon.Array, 0, 8)
			for dec.More() {
				v, err := readJSONValue(dec)
				if err != nil {
					return nil, err
				}
				arr = append(arr, v)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return arr, nil
		default:
			return nil, fmt.Errorf("unexpected delimiter %q", t)
		}
	case string:
		return t, nil
	case bool:
		return t, nil
	case nil:
		return nil, nil
	case json.Number:
		return toon.Number(t.String()), nil
	default:
		return nil, fmt.Errorf("unexpected token type %T", t)
	}
}

func equalValue(a, b toon.Value) bool {
	a = unboxValue(a)
	b = unboxValue(b)

	if av, ok := objectFromValue(a); ok {
		bv, ok := objectFromValue(b)
		if !ok || len(av.Members) != len(bv.Members) {
			return false
		}
		for i := range av.Members {
			if av.Members[i].Key != bv.Members[i].Key {
				return false
			}
			if !equalValue(av.Members[i].Value, bv.Members[i].Value) {
				return false
			}
		}
		return true
	}
	if av, ok := arrayFromValue(a); ok {
		bv, ok := arrayFromValue(b)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !equalValue(av[i], bv[i]) {
				return false
			}
		}
		return true
	}

	switch av := a.(type) {
	case toon.Number:
		switch bv := b.(type) {
		case toon.Number:
			return string(av) == string(bv)
		case string:
			return string(av) == bv
		default:
			return false
		}
	case string:
		if bn, ok := b.(toon.Number); ok {
			return av == string(bn)
		}
		bv, ok := b.(string)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case nil:
		return b == nil
	default:
		return fmt.Sprintf("%#v", a) == fmt.Sprintf("%#v", b)
	}
}

func unboxValue(v toon.Value) toon.Value {
	return toon.Unbox(v)
}

func objectFromValue(v toon.Value) (toon.Object, bool) {
	return toon.AsObject(v)
}

func arrayFromValue(v toon.Value) (toon.Array, bool) {
	return toon.AsArray(v)
}

func normalizeSection(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "§")
	if s == "" {
		return "unspecified"
	}
	return s
}

func inc(m map[string]*sectionStats, k string) {
	st, ok := m[k]
	if !ok {
		st = &sectionStats{}
		m[k] = st
	}
	st.Total++
}

func totals(m map[string]*sectionStats) (int, int, int) {
	t, p, f := 0, 0, 0
	for _, st := range m {
		t += st.Total
		p += st.Pass
		f += st.Fail
	}
	return t, p, f
}

func sortedKeys[T any](m map[string]*T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedSectionKeys(m map[string]*sectionStats) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a := sectionSortKey(keys[i])
		b := sectionSortKey(keys[j])
		if a != b {
			return a < b
		}
		return keys[i] < keys[j]
	})
	return keys
}

func sectionSortKey(s string) string {
	if s == "unspecified" {
		return "zzzz"
	}
	parts := strings.Split(s, ".")
	for i, p := range parts {
		parts[i] = fmt.Sprintf("%04s", p)
	}
	return strings.Join(parts, ".")
}

func downloadAndExtractSpecTar(tarURL, workDir string) error {
	if err := os.RemoveAll(workDir); err != nil {
		return fmt.Errorf("clean work dir: %w", err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return fmt.Errorf("create work dir: %w", err)
	}

	resp, err := http.Get(tarURL)
	if err != nil {
		return fmt.Errorf("download tarball: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download tarball: unexpected HTTP status %s", resp.Status)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("open gzip stream: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}
		if h == nil || h.Name == "" {
			continue
		}

		name := filepath.Clean(h.Name)
		parts := strings.Split(name, string(filepath.Separator))
		if len(parts) < 2 {
			continue
		}
		rel := filepath.Join(parts[1:]...)
		if rel == "." || strings.HasPrefix(rel, "..") {
			continue
		}
		dst := filepath.Join(workDir, rel)
		if !strings.HasPrefix(dst, workDir+string(filepath.Separator)) && dst != workDir {
			return fmt.Errorf("unsafe archive path %q", h.Name)
		}

		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return fmt.Errorf("mkdir %s: %w", dst, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return fmt.Errorf("mkdir parent for %s: %w", dst, err)
			}
			mode := os.FileMode(0o644)
			if h.Mode != 0 {
				mode = os.FileMode(h.Mode)
			}
			f, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
			if err != nil {
				return fmt.Errorf("create %s: %w", dst, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				return fmt.Errorf("write %s: %w", dst, err)
			}
			if err := f.Close(); err != nil {
				return fmt.Errorf("close %s: %w", dst, err)
			}
		}
	}

	fixturesDir := filepath.Join(workDir, "tests", "fixtures")
	if _, err := os.Stat(fixturesDir); err != nil {
		return fmt.Errorf("fixtures dir missing after extraction: %s", fixturesDir)
	}
	return nil
}
