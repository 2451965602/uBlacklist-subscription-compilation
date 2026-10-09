package main

import (
	"reflect"
	"testing"
)

func TestNormalizeAndDedupe(t *testing.T) {
	input := "  HTTPS://Example.COM/  \r\nhttps://example.com\r\n  title/Keep Case  \n/regex/\n# comment\n! comment\n\n"
	want := []string{"/regex/", "HTTPS://Example.COM/", "title/Keep Case"}
	if got := normalizeAndDedupe(input); !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeAndDedupe() = %#v, want %#v", got, want)
	}
}

func TestExclusionExactness(t *testing.T) {
	exact := []string{normalizeRule("*://*.baidu.com/*")}
	domains := []string{"csdn.net", "q.qq.com"}
	tests := []struct {
		rule string
		want bool
	}{
		{"https://csdn.net/", true},
		{"csdn.net", true},
		{"https://www.csdn.net/path", true},
		{"https://notcsdn.net/", false},
		{"https://example.com/?next=csdn.net", false},
		{"*://*.baidu.com/*", true},
		{"https://baidu.com/", false},
		{"https://q.qq.com/", true},
		{"https://qq.com/", false},
	}
	for _, test := range tests {
		if got := isExcluded(test.rule, exact, domains); got != test.want {
			t.Errorf("isExcluded(%q) = %v, want %v", test.rule, got, test.want)
		}
	}
}

func TestNormalizeRulePreservesExpressions(t *testing.T) {
	for _, rule := range []string{"title/Some Title", "/foo\\/bar/i"} {
		if got := normalizeRule(rule); got != rule {
			t.Errorf("normalizeRule(%q) = %q", rule, got)
		}
	}
	if got := normalizeRule("Example.COM/"); got != "example.com" {
		t.Errorf("normalizeRule(domain) = %q, want %q", got, "example.com")
	}
}
