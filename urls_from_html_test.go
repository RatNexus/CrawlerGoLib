package main

import (
	"reflect"
	"testing"
)

func TestGetURLsFromHTML(t *testing.T) {
	tests := []struct {
		name      string
		inputURL  string
		inputBody string
		expected  []string
	}{
		{
			name:     "absolute and relative URLs",
			inputURL: "https://blog.boot.dev",
			inputBody: `
			<html>
				<body>
					<a href="/path/one">
						<span>Boot.dev</span>
					</a>
					<a href="https://other.com/path/one">
						<span>Boot.dev</span>
					</a>
				</body>
			</html>
			`,
			expected: []string{"https://blog.boot.dev/path/one", "https://other.com/path/one"},
		},
		{
			name:     "no links",
			inputURL: "https://blog.boot.dev",
			inputBody: `
			<html>
				<body>
					<p>No links here</p>
				</body>
			</html>
			`,
			expected: []string{},
		},
		{
			name:     "multiple relative URLs",
			inputURL: "https://example.com",
			inputBody: `
			<html>
				<body>
					<a href="/page1">Page 1</a>
					<a href="/page2">Page 2</a>
					<a href="/nested/page3">Page 3</a>
				</body>
			</html>
			`,
			expected: []string{
				"https://example.com/page1",
				"https://example.com/page2",
				"https://example.com/nested/page3",
			},
		},
		{
			name:     "multiple absolute URLs",
			inputURL: "https://example.com",
			inputBody: `
			<html>
				<body>
					<a href="https://google.com">Google</a>
					<a href="https://github.com">GitHub</a>
					<a href="http://insecure.com">Insecure</a>
				</body>
			</html>
			`,
			expected: []string{
				"https://google.com",
				"https://github.com",
				"http://insecure.com",
			},
		},
		{
			name:     "mixed valid and invalid URLs",
			inputURL: "https://example.com",
			inputBody: `
			<html>
				<body>
					<a href="/valid">Valid</a>
					<a href="">Empty</a>
					<a href="https://valid.com">Valid Absolute</a>
					<a>No href</a>
				</body>
			</html>
			`,
			expected: []string{
				"https://example.com/valid",
				"https://valid.com",
			},
		},
		{
			name:     "URLs with query parameters and fragments",
			inputURL: "https://example.com",
			inputBody: `
			<html>
				<body>
					<a href="/search?q=test">Search</a>
					<a href="/page#section">Section</a>
					<a href="https://other.com/path?param=value#anchor">External</a>
				</body>
			</html>
			`,
			expected: []string{
				"https://example.com/search?q=test",
				"https://example.com/page#section",
				"https://other.com/path?param=value#anchor",
			},
		},
		{
			name:     "malformed HTML",
			inputURL: "https://example.com",
			inputBody: `
			<html>
				<body>
					<a href="/page1">Unclosed link
					<a href="/page2">Another link</a>
					<div><a href="/nested">Nested</a></div>
				</body>
			`,
			expected: []string{
				"https://example.com/page1",
				"https://example.com/page2",
				"https://example.com/nested",
			},
		},
		{
			name:     "base URL with path",
			inputURL: "https://example.com/blog/posts",
			inputBody: `
			<html>
				<body>
					<a href="/home">Home</a>
					<a href="./related">Related</a>
					<a href="../category">Category</a>
				</body>
			</html>
			`,
			expected: []string{
				"https://example.com/blog/posts/home",
				"./related",
				"../category",
			},
		},
		{
			name:     "duplicate URLs",
			inputURL: "https://example.com",
			inputBody: `
			<html>
				<body>
					<a href="/page">Page</a>
					<a href="/page">Same Page</a>
					<a href="https://example.com/page">Absolute Same</a>
				</body>
			</html>
			`,
			expected: []string{
				"https://example.com/page",
			},
		},
		{
			name:     "all URLs should not be normalised",
			inputURL: "https://example.com",
			inputBody: `
			<html>
				<body>
					<a href="/page">Page</a>
					<a href="illegal">Absolute Same</a>
				</body>
			</html>
			`,
			expected: []string{
				"https://example.com/page",
				"illegal",
			},
		},
		{
			name:     "empty URLs should not be ignored",
			inputURL: "https://example.com",
			inputBody: `
			<html>
				<body>
					<a href="">Same Page</a>
					<a href="/page">Page</a>
					<a href="">Same Page</a>
				</body>
			</html>
			`,
			expected: []string{
				"https://example.com/page",
			},
		},
		{
			name:     "multiple URLs should yeld only one in anchor",
			inputURL: "https://example.com",
			inputBody: `
			<html>
				<body>
					<a href="/page", href="/other">Page</a>
				</body>
			</html>
			`,
			expected: []string{
				"https://example.com/page",
			},
		},
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := getURLsFromHTML(tc.inputBody, tc.inputURL)
			if err != nil {
				t.Errorf("Test %v - '%s' FAIL: unexpected error: %v", i, tc.name, err)
				return
			}
			if !reflect.DeepEqual(tc.expected, actual) && (len(tc.expected) != 0 && len(actual) != 0) {
				t.Errorf("Test %v - %s FAIL: expected URL: \"%v\", actual: \"%v\"", i, tc.name, tc.expected, actual)
			}
		})
	}
}
