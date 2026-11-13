// Package flowtest prova os scripts do fluxo git num repositório de verdade. Roda no `make ci`, dentro do container,
// que tem bash e git.
package flowtest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const changelog = `# Registro de mudanças

## [Não lançado]

- Novidade ainda não lançada.

## [0.1.0] — 2026-01-01

- O que já saiu.
`

type repo struct {
	t   *testing.T
	dir string
	env []string
}

// newRepo cria um repositório com o CHANGELOG à mão, o VERSION 0.1.0 num commit de baseline em main e o develop.
func newRepo(t *testing.T, mode string) repo {
	t.Helper()
	r := repo{t: t, dir: t.TempDir()}
	r.env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t",
		"GIT_COMMITTER_EMAIL=t@t", "GIT_CONFIG_GLOBAL=/dev/null")
	if err := os.CopyFS(filepath.Join(r.dir, "scripts"), os.DirFS("../../scripts")); err != nil {
		t.Fatal(err)
	}
	r.write(".workflow.conf", "GATE_CMD=true\nFORMAT_ON_COMMIT=0\nCHANGELOG_MODE="+mode+"\n")
	r.write("CHANGELOG.md", changelog)
	r.write("VERSION", "0.1.0\n")
	r.git("init", "-q", "-b", "main")
	r.git("add", ".")
	r.git("commit", "-q", "-m", "chore(release): v0.1.0 (baseline)")
	r.git("checkout", "-q", "-b", "develop")
	return r
}

func (r repo) write(name, content string) {
	r.t.Helper()
	if err := os.WriteFile(filepath.Join(r.dir, name), []byte(content), 0o644); err != nil { //nolint:gosec // Arquivo de teste.
		r.t.Fatal(err)
	}
}

func (r repo) read(name string) string {
	r.t.Helper()
	content, err := os.ReadFile(filepath.Join(r.dir, name))
	if err != nil {
		r.t.Fatal(err)
	}
	return string(content)
}

// run roda o comando no repositório e devolve a saída e se terminou com sucesso.
func (r repo) run(name string, args ...string) (string, bool) {
	r.t.Helper()
	command := exec.Command(name, args...)
	command.Dir, command.Env = r.dir, r.env
	out, err := command.CombinedOutput()
	return string(out), err == nil
}

func (r repo) git(args ...string) string {
	r.t.Helper()
	out, ok := r.run("git", args...)
	if !ok {
		r.t.Fatalf("git %s: %s", strings.Join(args, " "), out)
	}
	return out
}

// checkCommit põe o arquivo no stage e roda o check-version como o gancho commit-msg roda.
func (r repo) checkCommit(file, content, subject string) (string, bool) {
	r.t.Helper()
	r.write(file, content)
	r.git("add", file)
	r.write(".git/COMMIT_EDITMSG", subject+"\n")
	return r.run("scripts/check-version", ".git/COMMIT_EDITMSG")
}

func TestCuratedChangelogTakesEditsOnlyInTheUnreleasedSection(t *testing.T) {
	cases := []struct {
		name    string
		mode    string
		file    string
		content string
		subject string
		allowed bool
	}{
		{"nota nova no não lançado", "curated", "CHANGELOG.md",
			strings.Replace(changelog, "- Novidade", "- Outra.\n- Novidade", 1), "fix: something", true},
		{"versão já lançada reescrita", "curated", "CHANGELOG.md",
			strings.Replace(changelog, "O que já saiu.", "Outra história.", 1), "fix: something", false},
		{"título do não lançado apagado", "curated", "CHANGELOG.md",
			strings.Replace(changelog, "## [Não lançado]\n", "", 1), "fix: something", false},
		{"cabeçalho mudado", "curated", "CHANGELOG.md",
			strings.Replace(changelog, "# Registro de mudanças", "# Mudanças", 1), "fix: something", false},
		{"VERSION fora da release", "curated", "VERSION", "0.2.0\n", "fix: something", false},
		{"commit de release", "curated", "CHANGELOG.md",
			strings.Replace(changelog, "O que já saiu.", "Outra história.", 1), "chore(release): v0.2.0", true},
		// Controle: no modo gerado, nem a seção do não lançado se edita à mão.
		{"modo gerado", "generated", "CHANGELOG.md",
			strings.Replace(changelog, "- Novidade", "- Outra.\n- Novidade", 1), "fix: something", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := newRepo(t, c.mode)
			out, ok := r.checkCommit(c.file, c.content, c.subject)
			if ok != c.allowed {
				t.Errorf("check-version aceitou = %v, queria %v; saída: %s", ok, c.allowed, out)
			}
		})
	}
}

func TestReleasePromotesTheCuratedSection(t *testing.T) {
	r := newRepo(t, "curated")
	r.write("feature.txt", "x\n")
	r.git("add", "feature.txt")
	r.git("commit", "-q", "-m", "feat: something new")

	if out, ok := r.run("scripts/release"); !ok {
		t.Fatalf("release falhou: %s", out)
	}

	want := strings.Replace(changelog, "## [Não lançado]\n",
		"## [Não lançado]\n\n## [0.2.0] — "+time.Now().Format(time.DateOnly)+"\n", 1)
	if got := r.read("CHANGELOG.md"); got != want {
		t.Errorf("CHANGELOG depois da release:\n%s\nqueria:\n%s", got, want)
	}
	if got := strings.TrimSpace(r.read("VERSION")); got != "0.2.0" {
		t.Errorf("VERSION = %s, queria 0.2.0", got)
	}
	if got := strings.TrimSpace(r.git("log", "-1", "--format=%s")); got != "chore(release): v0.2.0" {
		t.Errorf("último commit = %q", got)
	}
	r.git("rev-parse", "--verify", "v0.2.0")
}

func TestReleaseRefusesAnEmptyCuratedSection(t *testing.T) {
	r := newRepo(t, "curated")
	r.write("CHANGELOG.md", strings.Replace(changelog, "- Novidade ainda não lançada.\n", "", 1))
	r.write("feature.txt", "x\n")
	r.git("add", ".")
	r.git("commit", "-q", "-m", "feat: something new")

	out, ok := r.run("scripts/release")
	if ok || !strings.Contains(out, "vazia") {
		t.Errorf("release com a seção vazia passou ou não disse por quê: %s", out)
	}
	if got := strings.TrimSpace(r.read("VERSION")); got != "0.1.0" {
		t.Errorf("VERSION mudou para %s numa release recusada", got)
	}
}
