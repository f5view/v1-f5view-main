// Black-box acceptance tests for marketplace v1.
//
// The tests do not import the student's code. They drive a built binary via
// stdin/stdout and assert on observable behavior only. Point the harness at
// the binary in one of two ways:
//
//	MARKETPLACE_BIN=/abs/path/to/marketplace  go test -v ./...
//	MARKETPLACE_SRC=/abs/path/to/project      go test -v ./...   # harness runs `go build`
package acceptance_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

var binPath string

func TestMain(m *testing.M) {
	bin := os.Getenv("MARKETPLACE_BIN")
	if bin == "" {
		src := os.Getenv("MARKETPLACE_SRC")
		if src == "" {
			fmt.Fprintln(os.Stderr, "ERROR: set MARKETPLACE_BIN=/path/to/binary or MARKETPLACE_SRC=/path/to/project")
			os.Exit(2)
		}
		tmp, err := os.MkdirTemp("", "marketplace-bin-*")
		if err != nil {
			fmt.Fprintln(os.Stderr, "mkdir temp:", err)
			os.Exit(2)
		}
		bin = filepath.Join(tmp, "marketplace")
		cmd := exec.Command("go", "build", "-o", bin, ".")
		cmd.Dir = src
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "go build failed:\n%s\n%v\n", out, err)
			os.Exit(2)
		}
	}
	if _, err := os.Stat(bin); err != nil {
		fmt.Fprintln(os.Stderr, "binary not found:", bin)
		os.Exit(2)
	}
	binPath = bin
	os.Exit(m.Run())
}

// run launches the binary, writes `stdin` (auto-appends "\nexit\n"),
// waits up to 10s, returns stdout, stderr, exit code.
func run(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	if !strings.HasSuffix(stdin, "\n") {
		stdin += "\n"
	}
	cmd.Stdin = strings.NewReader(stdin + "exit\n")
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatalf("wait: %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errb.String())
		}
		return stripPrompts(out.String()), stripPrompts(errb.String()), code
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatalf("binary timed out\nstdout:\n%s\nstderr:\n%s", out.String(), errb.String())
		return "", "", -1
	}
}

// stripPrompts removes leading "> " from each line.
func stripPrompts(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		for strings.HasPrefix(l, "> ") {
			l = l[2:]
		}
		lines[i] = l
	}
	return strings.Join(lines, "\n")
}

func mustContain(t *testing.T, where, what string) {
	t.Helper()
	if !strings.Contains(where, what) {
		t.Errorf("expected to contain %q. full text:\n---\n%s\n---", what, where)
	}
}

func mustNotContain(t *testing.T, where, what string) {
	t.Helper()
	if strings.Contains(where, what) {
		t.Errorf("expected NOT to contain %q. full text:\n---\n%s\n---", what, where)
	}
}

// ============================================================================
// Tests
// ============================================================================

func TestAcc01_AddAndList(t *testing.T) {
	stdin := strings.Join([]string{
		`add "Молоко" 50.00 10`,
		`add "Хлеб" 30.50 5`,
		`list`,
	}, "\n")
	out, _, code := run(t, stdin)
	if code != 0 {
		t.Fatalf("non-zero exit: %d", code)
	}
	mustContain(t, out, "Listing added with ID: 1")
	mustContain(t, out, "Listing added with ID: 2")

	row := regexp.MustCompile(`(?m)^\d+\. "[^"]+" — \d+\.\d{2} \(stock: \d+\)$`)
	matches := row.FindAllString(out, -1)
	if len(matches) < 2 {
		t.Errorf("expected at least 2 list rows in expected format; matched %d. stdout:\n%s", len(matches), out)
	}
}

func TestAcc02_ListEmptyIsSilent(t *testing.T) {
	out, _, _ := run(t, `list`)
	// Strip trailing Goodbye/newlines and any prompt residue.
	out = strings.ReplaceAll(out, "Goodbye!", "")
	out = strings.TrimSpace(out)
	if out != "" {
		t.Errorf("expected empty list to print nothing, got:\n%q", out)
	}
}

func TestAcc03_BuyReducesStock(t *testing.T) {
	stdin := strings.Join([]string{
		`add "Молоко" 50.00 10`,
		`buy 1 3`,
		`list`,
	}, "\n")
	out, _, _ := run(t, stdin)
	mustContain(t, out, "Bought 3 of listing 1 (remaining: 7)")
	mustContain(t, out, "(stock: 7)")
}

func TestAcc04_BuyInsufficientStock(t *testing.T) {
	stdin := strings.Join([]string{
		`add "Молоко" 50.00 2`,
		`buy 1 5`,
		`list`,
	}, "\n")
	out, errb, _ := run(t, stdin)
	mustContain(t, out+errb, "Error: insufficient stock")
	// Stock must not have changed.
	mustContain(t, out, "(stock: 2)")
}

func TestAcc05_DeleteRemovesListing(t *testing.T) {
	stdin := strings.Join([]string{
		`add "Молоко" 50.00 10`,
		`delete 1`,
		`buy 1 1`,
	}, "\n")
	out, errb, _ := run(t, stdin)
	mustContain(t, out, "Listing 1 deleted")
	mustContain(t, out+errb, "Error: listing not found")
}

func TestAcc06_IDsDoNotReset(t *testing.T) {
	stdin := strings.Join([]string{
		`add "A" 10 1`,
		`add "B" 10 1`,
		`delete 1`,
		`delete 2`,
		`add "C" 10 1`,
	}, "\n")
	out, _, _ := run(t, stdin)
	if !strings.Contains(out, "Listing added with ID: 3") {
		t.Errorf("expected new ID after deletes to be 3 (no reuse), got:\n%s", out)
	}
}

func TestAcc07_Validation(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		expect string
	}{
		{"empty title", `add "" 10 1`, "Error: title cannot be empty"},
		{"long title", fmt.Sprintf(`add "%s" 10 1`, strings.Repeat("a", 201)), "Error: title is too long"},
		{"zero price", `add "x" 0 1`, "Error: price must be positive"},
		{"negative price", `add "x" -5 1`, "Error: price must be positive"},
		{"negative stock", `add "x" 10 -1`, "Error: stock must be non-negative"},
		{"zero buy qty", "add \"x\" 10 5\nbuy 1 0", "Error: quantity must be positive"},
		{"buy missing", `buy 999 1`, "Error: listing not found"},
		{"unknown command", `flarble`, "Error: unknown command"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, errb, code := run(t, tc.input)
			if code != 0 {
				t.Fatalf("REPL crashed on bad input, exit %d. stdout:\n%s\nstderr:\n%s", code, out, errb)
			}
			mustContain(t, out+errb, tc.expect)
		})
	}
}

func TestAcc08_ErrorsGoToStderr(t *testing.T) {
	// A spec rule worth its own test: errors must go to stderr, not stdout.
	_, errb, _ := run(t, `add "" 10 1`)
	mustContain(t, errb, "Error: title cannot be empty")
}

func TestAcc09_ExitGoodbye(t *testing.T) {
	out, _, code := run(t, ``) // run() auto-appends exit
	if code != 0 {
		t.Errorf("exit should be 0, got %d", code)
	}
	mustContain(t, out, "Goodbye!")
}

// --- JSON persistence ---

func TestAcc10_JSONRoundTrip(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "data.json")

	// session 1: add two listings, then buy from one
	stdin1 := strings.Join([]string{
		`add "Молоко" 50.00 10`,
		`add "Хлеб" 30.50 5`,
		`buy 1 4`,
	}, "\n")
	_, _, code := run(t, stdin1, "--storage=json", "--file="+file)
	if code != 0 {
		t.Fatalf("session 1 exit %d", code)
	}

	// file must exist and be valid JSON
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("data file is not valid JSON. raw:\n%s", raw)
	}

	// session 2: list — must see both listings, stock of #1 = 6
	out, _, _ := run(t, `list`, "--storage=json", "--file="+file)
	mustContain(t, out, "Молоко")
	mustContain(t, out, "(stock: 6)")
	mustContain(t, out, "Хлеб")

	// session 3: new listing must get ID >= 3 (no reset on reload)
	out, _, _ = run(t, `add "Сыр" 200 1`, "--storage=json", "--file="+file)
	if !regexp.MustCompile(`Listing added with ID: [3-9]\d*`).MatchString(out) {
		t.Errorf("ID after reload should not reset; got:\n%s", out)
	}
}
