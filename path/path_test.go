package path

import (
	"testing"
)

func TestJoin(t *testing.T) {
	tests := []struct {
		name  string
		elems []string
		want  string
	}{
		// Unix-style paths
		{"UnixSimple", []string{"foo", "bar"}, "foo/bar"},
		{"UnixNested", []string{"a", "b", "c"}, "a/b/c"},
		{"UnixRoot", []string{"/", "foo"}, "/foo"},
		{"UnixTrailingSlash", []string{"foo", "bar", ""}, "foo/bar/"},
		{"UnixEmpty", []string{"", "foo", "", "bar"}, "foo/bar"},
		{"UnixSingle", []string{"foo"}, "foo"},
		{"UnixAllEmpty", []string{"", "", ""}, ""},
		{"UnixDotDot", []string{"foo", "..", "bar"}, "bar"},
		{"UnixLocalDot", []string{"./foo", "bar"}, "./foo/bar"},
		{"UnixLocalDotNested", []string{"./foo", "./bar"}, "./foo/bar"},

		// Windows-style paths
		{"WindowsDrive", []string{`C:\`, "foo"}, `C:\foo`},
		{"WindowsDriveLower", []string{`c:\`, "foo"}, `c:\foo`},
		{"WindowsNested", []string{`C:\`, "Users", "foo"}, `C:\Users\foo`},
		{"WindowsTrailing", []string{`C:\`, "foo", ""}, `C:\foo\`},
		{"WindowsBackslash", []string{`foo\bar`, "baz"}, `foo\bar\baz`},
		{"WindowsMixed", []string{`C:\foo`, "bar"}, `C:\foo\bar`},
		{"WindowsLocalDot", []string{`.\foo`, "bar"}, `.\foo\bar`},
		{"WindowsLocalDotNested", []string{`.\foo`, `.\bar`}, `.\foo\bar`},

		// URL-style paths
		{"URLSimple", []string{"https://example.com", "foo"}, "https://example.com/foo"},
		{"URLNested", []string{"https://example.com", "foo", "bar"}, "https://example.com/foo/bar"},
		{"URLTrailing", []string{"https://example.com/foo", ""}, "https://example.com/foo/"},
		{"URLS3", []string{"s3://bucket", "key", "path"}, "s3://bucket/key/path"},
		{"URLFile", []string{"file:///", "home", "user"}, "file:///home/user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Join(tt.elems...)
			if got != tt.want {
				t.Errorf("Join(%q) = %q, want %q", tt.elems, got, tt.want)
			}
		})
	}
}

func TestSplit(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantDir  string
		wantFile string
	}{
		// Unix-style
		{"UnixFile", "foo", "", "foo"},
		{"UnixPath", "foo/bar", "foo", "bar"},
		{"UnixNested", "a/b/c", "a/b", "c"},
		{"UnixRoot", "/foo", "/", "foo"},
		{"UnixDir", "foo/bar/", "foo/bar", ""},
		{"UnixRootDir", "/", "/", ""},
		{"UnixEmpty", "", "", ""},
		{"UnixLocalDot", "./foo", ".", "foo"},
		{"UnixLocalDotPath", "./foo/bar", "./foo", "bar"},

		// Windows-style
		{"WindowsFile", `C:\foo`, `C:\`, "foo"},
		{"WindowsPath", `C:\Users\foo`, `C:\Users`, "foo"},
		{"WindowsDir", `C:\foo\`, `C:\foo`, ""},
		{"WindowsRoot", `C:\`, `C:\`, ""},
		{"WindowsLocalDot", `.\foo`, ".", "foo"},
		{"WindowsLocalDotPath", `.\foo\bar`, `.\foo`, "bar"},

		// URL-style
		{"URLPath", "https://example.com/foo", "https://example.com/", "foo"},
		{"URLNested", "https://example.com/foo/bar", "https://example.com/foo", "bar"},
		{"URLRoot", "https://example.com/", "https://example.com/", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDir, gotFile := Split(tt.path)
			if gotDir != tt.wantDir || gotFile != tt.wantFile {
				t.Errorf("Split(%q) = (%q, %q), want (%q, %q)",
					tt.path, gotDir, gotFile, tt.wantDir, tt.wantFile)
			}
		})
	}
}

func TestBase(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"foo/bar", "bar"},
		{"foo/bar/", ""},
		{"/foo/bar", "bar"},
		{"foo", "foo"},
		{"/", ""},
		{"", ""},
		{"./foo", "foo"},
		{"./foo/bar", "bar"},
		{`.\foo`, "foo"},
		{`.\foo\bar`, "bar"},
		{`C:\Users\foo`, "foo"},
		{`C:\foo\`, ""},
		{"https://example.com/foo", "foo"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := Base(tt.path)
			if got != tt.want {
				t.Errorf("Base(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestDir(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"foo/bar", "foo"},
		{"foo/bar/", "foo/bar"},
		{"/foo/bar", "/foo"},
		{"foo", ""},
		{"/", "/"},
		{"", ""},
		{"./foo", "."},
		{"./foo/bar", "./foo"},
		{`.\foo`, "."},
		{`.\foo\bar`, `.\foo`},
		{`C:\Users\foo`, `C:\Users`},
		{`C:\foo`, `C:\`},
		{"https://example.com/foo/bar", "https://example.com/foo"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := Dir(tt.path)
			if got != tt.want {
				t.Errorf("Dir(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsDir(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"foo/bar/", true},
		{"foo/bar", false},
		{"/", true},
		{"", false},
		{`C:\foo\`, true},
		{`C:\foo`, false},
		{"https://example.com/", true},
		{"https://example.com/foo", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsDir(tt.path)
			if got != tt.want {
				t.Errorf("IsDir(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsRoot(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/", true},
		{"/foo", false},
		{"foo", false},
		{`C:\`, true},
		{`C:\foo`, false},
		{`D:\`, true},
		{"https://example.com/", true},
		{"https://example.com/foo", false},
		{"s3://bucket/", true},
		{"file:///", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsRoot(tt.path)
			if got != tt.want {
				t.Errorf("IsRoot(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsAbs(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		// Unix-style
		{"/", true},
		{"/foo", true},
		{"/foo/bar", true},
		{"foo", false},
		{"foo/bar", false},
		{"./foo", false},
		{"./foo/bar", false},
		{"", false},

		// Windows-style
		{`C:\`, true},
		{`C:\foo`, true},
		{`C:/foo`, true},
		{`c:\`, true},
		{`D:\Users\foo`, true},
		{`foo\bar`, false},
		{`.\foo`, false},
		{`.\foo\bar`, false},

		// URL-style
		{"https://example.com", true},
		{"https://example.com/", true},
		{"https://example.com/foo", true},
		{"s3://bucket/key", true},
		{"file:///home/user", true},
		{"http://localhost", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsAbs(tt.path)
			if got != tt.want {
				t.Errorf("IsAbs(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestClean(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		// Unix-style
		{"UnixSimple", "foo/bar", "foo/bar"},
		{"UnixDot", "foo/./bar", "foo/bar"},
		{"UnixDotDot", "foo/../bar", "bar"},
		{"UnixMultipleSep", "foo//bar", "foo/bar"},
		{"UnixTrailing", "foo/bar/", "foo/bar/"},
		{"UnixEmpty", "", "."},
		{"UnixDotDotRelative", "../foo", "../foo"},
		{"UnixRootEscape", "/..", "/"},
		{"UnixRootEscape2", "/../foo", "/foo"},
		{"UnixLocalDot", "./foo", "./foo"},
		{"UnixLocalDotSlash", "./foo/./bar", "./foo/bar"},
		{"UnixLocalDotDot", "./foo/../bar", "./bar"},

		// Windows-style
		{"WindowsSimple", `C:\foo\bar`, `C:\foo\bar`},
		{"WindowsDot", `C:\foo\.\bar`, `C:\foo\bar`},
		{"WindowsDotDot", `C:\foo\..\bar`, `C:\bar`},
		{"WindowsRootEscape", `C:\..`, `C:\`},
		{"WindowsRootEscape2", `C:\..\foo`, `C:\foo`},
		{"WindowsLocalDot", `.\foo`, `.\foo`},
		{"WindowsLocalDotSlash", `.\foo\.\bar`, `.\foo\bar`},
		{"WindowsLocalDotDot", `.\foo\..\bar`, `.\bar`},

		// URL-style
		{"URLSimple", "https://example.com/foo/bar", "https://example.com/foo/bar"},
		{"URLDot", "https://example.com/foo/./bar", "https://example.com/foo/bar"},
		{"URLDotDot", "https://example.com/foo/../bar", "https://example.com/bar"},
		{"URLRootEscape", "https://example.com/..", "https://example.com/"},
		{"URLRootEscape2", "https://example.com/../foo", "https://example.com/foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Clean(tt.path)
			if got != tt.want {
				t.Errorf("Clean(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestJoinSplitRoundtrip(t *testing.T) {
	tests := []struct {
		name  string
		elems []string
	}{
		{"Unix", []string{"foo", "bar"}},
		{"UnixNested", []string{"a", "b", "c", "d"}},
		{"Windows", []string{`C:\`, "foo", "bar"}},
		{"URL", []string{"https://example.com", "foo", "bar"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			joined := Join(tt.elems...)
			dir, file := Split(joined)
			rejoined := Join(dir, file)
			if rejoined != joined {
				t.Errorf("Join(Split(Join(%q))) = %q, want %q",
					tt.elems, rejoined, joined)
			}
		})
	}
}
