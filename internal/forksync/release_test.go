package forksync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// upstreamRepo is the only repository whose tag may publish the npm
// packages or warm the egoist/mygo module proxy.
const upstreamRepo = "github.repository == 'egoist/mygo'"

func TestReleaseWorkflow(t *testing.T) {
	root := moduleRoot(t)
	body, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := checkReleaseWorkflow(string(body)); err != nil {
		t.Fatal(err)
	}
	notes := filepath.Join(root, ".github", "fork-release-notes.md")
	b, err := os.ReadFile(notes)
	if err != nil {
		t.Fatal(err)
	}
	for _, phrase := range []string{
		"github.repository",
		"does not publish mygo-runtime",
		"GUI e2e",
		"was not run",
		"struct NativeB",
		"was not skipped",
	} {
		if !strings.Contains(string(b), phrase) {
			t.Errorf("release notes omit %q", phrase)
		}
	}
}

func TestReleaseWorkflowRejectsUngatedPublish(t *testing.T) {
	const ungated = `
jobs:
  publish:
    steps:
      - name: Check the version
        run: bun scripts/version.ts --check "$GITHUB_REF_NAME"
      - name: Publish the npm packages
        run: bun scripts/publish.ts --provenance
      - name: Create the GitHub release
        run: |
          if [[ -f .github/fork-release-notes.md ]]; then
            flags+=(--notes-file .github/fork-release-notes.md)
          else
            flags+=(--generate-notes)
          fi
          gh release create "$GITHUB_REF_NAME" --verify-tag --title "MyGo" "${flags[@]}"
`
	if err := checkReleaseWorkflow(ungated); err == nil {
		t.Fatal("an ungated npm publish was accepted")
	}
}

func TestVersionsAgree(t *testing.T) {
	root := moduleRoot(t)
	want := goString(t, filepath.Join(root, "mygo.go"), `const Version = "([^"]*)"`)
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(want) {
		t.Fatalf("mygo.Version %q is not a release semver", want)
	}
	if got := goString(t, filepath.Join(root, "cmd/mygo/main.go"), `const version = "([^"]*)"`); got != want {
		t.Fatalf("cmd/mygo version is %s, mygo.Version is %s", got, want)
	}
	plugins := stringList(t, filepath.Join(root, "scripts", "version.ts"), `export const pluginPackages = \[([^\]]*)\]`)
	platforms := stringList(t, filepath.Join(root, "packages", "cli", "index.js"), `export const platforms = \[([^\]]*)\]`)
	dirs := append([]string{"packages/runtime"}, plugins...)
	dirs = append(dirs, "packages/cli")
	for _, platform := range platforms {
		dirs = append(dirs, filepath.Join("packages", "cli", "npm", platform))
	}
	lock, err := os.ReadFile(filepath.Join(root, "bun.lock"))
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range dirs {
		name := checkPackage(t, filepath.Join(root, dir, "package.json"), want)
		re := regexp.MustCompile(`"name": "` + regexp.QuoteMeta(name) + `",\s*"version": "([^"]*)"`)
		m := re.FindStringSubmatch(string(lock))
		if m == nil {
			t.Errorf("bun.lock has no workspace version of %s", name)
			continue
		}
		if m[1] != want {
			t.Errorf("bun.lock workspace %s is %s, not %s", name, m[1], want)
		}
	}
	for _, platform := range platforms {
		name := "@egoist/mygo-cli-" + platform
		re := regexp.MustCompile(`"` + regexp.QuoteMeta(name) + `": "([^"]*)"`)
		m := re.FindStringSubmatch(string(lock))
		if m == nil {
			t.Errorf("bun.lock has no version of %s", name)
			continue
		}
		if m[1] != want {
			t.Errorf("bun.lock %s is %s, not %s", name, m[1], want)
		}
	}
}

// checkReleaseWorkflow reports how a release workflow would publish from
// a fork. The version check and the GitHub release stay unconditional.
// Publishing, the npm token and the upstream module proxy do not.
func checkReleaseWorkflow(yml string) error {
	var (
		sawCheck   bool
		sawPublish bool
		sawRelease bool
	)
	for _, step := range workflowSteps(yml) {
		text := strings.Join(step.lines, "\n")
		gated := strings.Contains(step.condition, upstreamRepo)
		switch {
		case strings.Contains(text, "scripts/publish.ts") || strings.Contains(text, "npm publish") || strings.Contains(text, "NPM_TOKEN") || strings.Contains(text, "proxy.golang.org/github.com/egoist/mygo") || strings.Contains(text, "npm install -g npm"):
			sawPublish = sawPublish || strings.Contains(text, "scripts/publish.ts")
			if !gated {
				return errString(step.name + " reaches npm or the egoist/mygo proxy without " + upstreamRepo)
			}
		case strings.Contains(text, "version.ts --check"):
			sawCheck = true
			if gated {
				return errString("the version check does not run on a fork")
			}
		case strings.Contains(text, "gh release create"):
			sawRelease = true
			if gated {
				return errString("the GitHub release does not run on a fork")
			}
			if !strings.Contains(text, "fork-release-notes.md") || !strings.Contains(text, "--generate-notes") {
				return errString("the GitHub release dropped the fork notes file or upstream's generated notes")
			}
		}
	}
	if !sawCheck || !sawPublish || !sawRelease {
		return errString("the workflow is missing the version check, npm publish, or GitHub release")
	}
	return nil
}

type step struct {
	name      string
	condition string
	lines     []string
}

func workflowSteps(yml string) []step {
	var steps []step
	var cur *step
	for _, line := range strings.Split(yml, "\n") {
		if strings.HasPrefix(line, "      - ") {
			steps = append(steps, step{name: strings.TrimSpace(strings.TrimPrefix(line, "      - "))})
			cur = &steps[len(steps)-1]
		}
		if cur == nil {
			continue
		}
		cur.lines = append(cur.lines, line)
		if rest, ok := strings.CutPrefix(line, "        if:"); ok {
			cur.condition = strings.TrimSpace(rest)
		}
		if rest, ok := strings.CutPrefix(line, "        name:"); ok {
			cur.name = strings.TrimSpace(rest)
		}
	}
	return steps
}

func checkPackage(t *testing.T, path, want string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var pkg struct {
		Name                 string
		Version              string
		OptionalDependencies map[string]string
		PeerDependencies     map[string]string
		DevDependencies      map[string]string
	}
	if err := json.Unmarshal(body, &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.Name == "" {
		t.Fatalf("%s has no name", path)
	}
	if pkg.Version != want {
		t.Errorf("%s version is %s, not %s", path, pkg.Version, want)
	}
	for name, v := range pkg.OptionalDependencies {
		if v != want {
			t.Errorf("%s optional %s is %s, not %s", path, name, v, want)
		}
	}
	for _, deps := range []map[string]string{pkg.PeerDependencies, pkg.DevDependencies} {
		if v, ok := deps["mygo-runtime"]; ok && strings.TrimPrefix(v, "^") != want {
			t.Errorf("%s mygo-runtime is %s, not %s", path, v, want)
		}
	}
	return pkg.Name
}

func goString(t *testing.T, path, pattern string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(pattern).FindSubmatch(body)
	if m == nil {
		t.Fatalf("%s has no %s", path, pattern)
	}
	return string(m[1])
}

func stringList(t *testing.T, path, pattern string) []string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(pattern).FindSubmatch(body)
	if m == nil {
		t.Fatalf("%s has no %s", path, pattern)
	}
	var list []string
	for _, part := range strings.Split(string(m[1]), ",") {
		part = strings.Trim(strings.TrimSpace(part), `"`)
		if part != "" {
			list = append(list, part)
		}
	}
	if len(list) == 0 {
		t.Fatalf("%s matched an empty list", path)
	}
	return list
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

type errString string

func (e errString) Error() string { return string(e) }
