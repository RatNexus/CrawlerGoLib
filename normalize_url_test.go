package main

import "testing"

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		inputURL string
		expected string
	}{
		{
			name:     "to lowercase",
			inputURL: "https://Example.Com/Path",
			expected: "example.com/path",
		},
		{
			name:     "remove scheme",
			inputURL: "https://example.com/path",
			expected: "example.com/path",
		},
		{
			name:     "remove default port",
			inputURL: "https://example.com:443/path",
			expected: "example.com/path",
		},
		{
			name:     "do not remove non-default port",
			inputURL: "https://example.com:0451/path",
			expected: "example.com:0451/path",
		},
		{
			name:     "remove dot-segments of less than three",
			inputURL: "https://example.com/a/./b/../.../c/./",
			expected: "example.com/a/b/.../c",
		},
		{
			name:     "remove trailing slash",
			inputURL: "https://example.com/folder/",
			expected: "example.com/folder",
		},
		{
			name:     "sort query parameters",
			inputURL: "https://example.com/path?b=2&a=1",
			expected: "example.com/path?a=1&b=2",
		},
		{
			name:     "remove fragments",
			inputURL: "https://example.com/path#section",
			expected: "example.com/path",
		},
		{
			name:     "remove default filename",
			inputURL: "https://example.com/index.html",
			expected: "example.com",
		},
		{
			name:     "multiple slashes",
			inputURL: "http://example.com///thing//andanother",
			expected: "example.com/thing/andanother",
		},
		{
			name:     "everywhere all at once",
			inputURL: "https://Example.cOm:443/./Folder/.///Index.html?z=2&c=1#section",
			expected: "example.com/folder?c=1&z=2",
		},
		{
			name:     "everywhere all at once 2",
			inputURL: "http://Example.cOm:420///./FOLDER/./Index.html?e=2&d=1&e=1#suction/",
			expected: "example.com:420/folder?d=1&e=1&e=2",
		},
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := normalizeURL(tc.inputURL)
			if err != nil {
				t.Errorf("Test %v - '%s' FAIL: unexpected error: %v", i, tc.name, err)
				return
			}
			if actual != tc.expected {
				t.Errorf("Test %v - %s FAIL: expected URL: \"%v\", actual: \"%v\"", i, tc.name, tc.expected, actual)
			}
		})
	}
}
