package pass

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"unicode/utf8"
)

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

var passLineRe = regexp.MustCompile(`^([^A-Za-z]*)([A-Za-z][A-Za-z0-9./_-]*)$`)

type Resource struct {
	Namespace string
	Name      string
}

func ParseResource(s string) (Resource, error) {
	parts := strings.Split(s, "/")
	switch len(parts) {
	case 1:
		return Resource{Namespace: parts[0]}, nil
	case 2:
		return Resource{Namespace: parts[0], Name: parts[1]}, nil
	default:
		return Resource{}, fmt.Errorf("resource must be in the format namespace/name")
	}
}

func Root() string {
	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}
	return "k8s/" + host
}

type Store struct {
	RunPass func(args []string, stdin io.Reader) (string, string, error)
}

func New() *Store {
	return &Store{
		RunPass: func(args []string, stdin io.Reader) (string, string, error) {
			cmd := exec.Command("pass", args...)
			if stdin != nil {
				cmd.Stdin = stdin
			}
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			return stdout.String(), stderr.String(), err
		},
	}
}

func ParsePassLines(prefix string, lines []string, index int) (int, []string, error) {
	var entries []string
	indent := -1
	last := ""

	cleanLine := func(line string) string {
		return ansiRe.ReplaceAllString(line, "")
	}

	for index < len(lines) {
		line := strings.TrimRight(lines[index], "\r")
		if line == "" {
			index++
			continue
		}

		cleaned := cleanLine(line)
		match := passLineRe.FindStringSubmatch(cleaned)
		if match == nil {
			return 0, nil, fmt.Errorf("%s: not parsed by line regex", line)
		}

		lineIndent := utf8.RuneCountInString(match[1])
		lineName := match[2]

		if indent < 0 {
			indent = lineIndent
			last = lineName
			index++
		} else if lineIndent == indent {
			if last != "" {
				entries = append(entries, subFolder(prefix, last))
			}
			last = lineName
			index++
		} else if lineIndent < indent {
			break
		} else {
			newIdx, newEntries, parseErr := ParsePassLines(subFolder(prefix, last), lines, index)
			if parseErr != nil {
				return 0, nil, parseErr
			}
			index = newIdx
			entries = append(entries, newEntries...)
			last = ""
		}
	}

	if last != "" {
		entries = append(entries, prefix+"/"+last)
	}
	return index, entries, nil
}

func CollectKeys(paths []string) map[string][]string {
	m := make(map[string][]string)
	for _, p := range paths {
		parts := strings.Split(p, "/")
		key := parts[len(parts)-1]
		secret := strings.Join(parts[:len(parts)-1], "/")
		m[secret] = append(m[secret], key)
	}
	return m
}

func subFolder(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "/" + child
}

func (s *Store) ListSecrets(root string) ([]string, error) {
	args := []string{"ls"}
	if root != "" {
		args = append(args, root)
	}
	stdout, stderr, err := s.RunPass(args, nil)
	if err != nil {
		return nil, fmt.Errorf("pass ls: %w", err)
	}
	if stderr != "" {
		fmt.Fprint(os.Stderr, stderr)
	}
	lines := strings.Split(stdout, "\n")
	_, result, parseErr := ParsePassLines(root, lines, 1)
	if parseErr != nil {
		return nil, parseErr
	}
	return result, nil
}

func (s *Store) GetSecret(path string) (string, error) {
	stdout, stderr, err := s.RunPass([]string{"show", path}, nil)
	if err != nil {
		return "", fmt.Errorf("pass show: %w", err)
	}
	if stderr != "" {
		fmt.Fprint(os.Stderr, stderr)
	}
	return strings.TrimRight(stdout, "\n"), nil
}

func (s *Store) InsertSecret(namespace, name, key, value string) error {
	path := fmt.Sprintf("%s/%s/%s/%s", Root(), namespace, name, key)
	_, stderr, err := s.RunPass([]string{"insert", "--echo", path}, strings.NewReader(value))
	if err != nil {
		return fmt.Errorf("pass insert: %w", err)
	}
	if stderr != "" {
		fmt.Fprint(os.Stderr, stderr)
	}
	return nil
}
