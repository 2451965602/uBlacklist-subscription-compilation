package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

// normalizeRule canonicalizes URL and uBlacklist match-pattern scheme/host
// casing, and removes a redundant trailing slash. Other expressions are kept
// intact after surrounding whitespace is trimmed.
func normalizeRule(rule string) string {
	rule = strings.TrimSpace(strings.ReplaceAll(rule, "\r", ""))
	if rule == "" {
		return ""
	}

	separator := strings.Index(rule, "://")
	if separator < 0 {
		domain := strings.TrimSuffix(rule, "/")
		if strings.Contains(domain, ".") && !strings.ContainsAny(domain, "/?#: \t") {
			return strings.ToLower(domain)
		}
		return rule
	}
	scheme := strings.ToLower(rule[:separator])
	remainder := rule[separator+3:]
	endHost := strings.IndexAny(remainder, "/?#")
	host := remainder
	suffix := ""
	if endHost >= 0 {
		host, suffix = remainder[:endHost], remainder[endHost:]
	}
	host = strings.ToLower(host)
	if suffix == "/" {
		suffix = ""
	}
	return scheme + "://" + host + suffix
}

func isComment(line string) bool {
	return strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!")
}

// normalizeAndDedupe retains the first spelling of each canonical rule,
// excludes comments and blank lines, then sorts the retained representatives.
func normalizeAndDedupe(input string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, line := range strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(strings.ReplaceAll(line, "\r", ""))
		if line == "" || isComment(line) {
			continue
		}
		canonical := normalizeRule(line)
		if _, found := seen[canonical]; found {
			continue
		}
		seen[canonical] = struct{}{}
		result = append(result, line)
	}
	sort.Strings(result)
	return result
}

// loadExclusions reads exact rules and domain selectors (domain:example.com).
func loadExclusions(path string) ([]string, []string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	var exact, domains []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimSuffix(scanner.Text(), "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "domain:") {
			domains = append(domains, strings.ToLower(strings.TrimPrefix(line, "domain:")))
		} else {
			exact = append(exact, normalizeRule(line))
		}
	}
	return exact, domains, scanner.Err()
}

func ruleHost(rule string) string {
	separator := strings.Index(rule, "://")
	if separator < 0 {
		return ""
	}
	host := rule[separator+3:]
	if slash := strings.IndexAny(host, "/?#"); slash >= 0 {
		host = host[:slash]
	}
	return strings.TrimPrefix(strings.ToLower(host), "*.")
}

func isExcluded(rule string, exact, domains []string) bool {
	canonical := normalizeRule(rule)
	for _, excluded := range exact {
		if canonical == excluded {
			return true
		}
	}
	host := ruleHost(canonical)
	for _, domain := range domains {
		if canonical == domain || host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

func filterExclusions(rules []string, exact, domains []string) []string {
	filtered := make([]string, 0, len(rules))
	for _, rule := range rules {
		if !isExcluded(rule, exact, domains) {
			filtered = append(filtered, rule)
		}
	}
	return filtered
}

func processFile(inputPath, outputPath, backupPath, exclusionsPath string) error {
	content, err := ioutil.ReadFile(inputPath)
	if err != nil {
		return err
	}
	exact, domains, err := loadExclusions(exclusionsPath)
	if err != nil {
		return err
	}
	rules := filterExclusions(normalizeAndDedupe(string(content)), exact, domains)
	output := strings.Join(rules, "\n")
	if len(rules) > 0 {
		output += "\n"
	}
	if err := ioutil.WriteFile(backupPath, content, 0644); err != nil {
		return err
	}
	return ioutil.WriteFile(outputPath, []byte(output), 0644)
}

func main() {
	input := flag.String("input", "uBlacklist.txt", "input list")
	output := flag.String("output", "uBlacklist.txt", "compiled output")
	backup := flag.String("backup", "uBlacklist_backup.txt", "input backup")
	exclusions := flag.String("exclusions", "../exclusions.txt", "exclusion rules")
	flag.Parse()
	if err := processFile(*input, *output, *backup, *exclusions); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
