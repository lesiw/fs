package path

import (
	"slices"
	"strings"
	"testing"
)

func TestJoin(t *testing.T) {
	tests := []struct {
		name  string
		elems []string
		want  string
	}{
		// Unix-style paths
		{"UnixSimple", []string{"foo", "bar"}, "./foo/bar"},
		{"UnixNested", []string{"a", "b", "c"}, "./a/b/c"},
		{"UnixRoot", []string{"/", "foo"}, "/foo"},
		{"UnixTrailingSlash", []string{"foo", "bar", ""}, "./foo/bar/"},
		{"UnixTrailingInArg", []string{"/tmp", "test_createdir/"},
			"/tmp/test_createdir/"},
		{"UnixTrailingInLastArg", []string{"foo", "bar/"}, "./foo/bar/"},
		{"UnixNoDoubleSep", []string{"foo/", "/bar/"}, "./foo/bar/"},
		{"UnixEmpty", []string{"", "foo", "", "bar"}, "./foo/bar"},
		{"UnixSingle", []string{"foo"}, "./foo"},
		{"UnixAllEmpty", []string{"", "", ""}, "."},
		{"UnixDotDot", []string{"foo", "..", "bar"}, "./bar"},
		{"UnixLocalDot", []string{"./foo", "bar"}, "./foo/bar"},
		{"UnixLocalDotNested", []string{"./foo", "./bar"}, "./foo/bar"},

		// Windows-style paths
		{"WindowsDrive", []string{`C:\`, "foo"}, `C:\foo`},
		{"WindowsDriveLower", []string{`c:\`, "foo"}, `c:\foo`},
		{"WindowsNested", []string{`C:\`, "Users", "foo"}, `C:\Users\foo`},
		{"WindowsTrailing", []string{`C:\`, "foo", ""}, `C:\foo\`},
		{"WindowsTrailingInArg", []string{`C:\`, `foo\`}, `C:\foo\`},
		{"WindowsTrailingInLastArg", []string{`.\foo`, `bar\`}, `.\foo\bar\`},
		{"WindowsBackslash", []string{`foo\bar`, "baz"}, `.\foo\bar\baz`},
		{"WindowsMixed", []string{`C:\foo`, "bar/baz"}, `C:\foo\bar\baz`},
		{"WindowsLocalDot", []string{`.\foo`, "bar"}, `.\foo\bar`},
		{"WindowsLocalDotNested", []string{`.\foo`, `.\bar`}, `.\foo\bar`},

		// URL-style paths
		{"URLSimple", []string{"https://example.com", "foo"},
			"https://example.com/foo"},
		{"URLNested", []string{"https://example.com", "foo", "bar"},
			"https://example.com/foo/bar"},
		{"URLTrailing", []string{"https://example.com/foo", ""},
			"https://example.com/foo/"},
		{"URLS3", []string{"s3://bucket", "key", "path"},
			"s3://bucket/key/path"},
		{"URLFile", []string{"file:///", "home", "user"},
			"file:///home/user"},
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
		{"UnixLocalDot", "./foo", "./", "foo"},
		{"UnixLocalDotPath", "./foo/bar", "./foo", "bar"},

		// Windows-style
		{"WindowsFile", `C:\foo`, `C:\`, "foo"},
		{"WindowsPath", `C:\Users\foo`, `C:\Users`, "foo"},
		{"WindowsDir", `C:\foo\`, `C:\foo`, ""},
		{"WindowsRoot", `C:\`, `C:\`, ""},
		{"WindowsLocalDot", `.\foo`, `.\`, "foo"},
		{"WindowsLocalDotPath", `.\foo\bar`, `.\foo`, "bar"},

		// URL-style
		{"URLPath", "https://example.com/foo",
			"https://example.com/", "foo"},
		{"URLNested", "https://example.com/foo/bar",
			"https://example.com/foo", "bar"},
		{"URLRoot", "https://example.com/",
			"https://example.com/", ""},
		{"URLRootNoSlash", "https://example.com",
			"https://example.com", ""},
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
		{"./foo", "./"},
		{"./foo/bar", "./foo"},
		{`.\foo`, `.\`},
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
		{"https://example.com", true},
		{"https://example.com/foo", false},
		{"s3://bucket/", true},
		{"s3://bucket", true},
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
		{"UnixSimple", "foo/bar", "./foo/bar"},
		{"UnixDot", "foo/./bar", "./foo/bar"},
		{"UnixDotDot", "foo/../bar", "./bar"},
		{"UnixMultipleSep", "foo//bar", "./foo/bar"},
		{"UnixTrailing", "foo/bar/", "./foo/bar/"},
		{"UnixEmpty", "", "."},
		{"UnixDotDotRelative", "../foo", "./../foo"},
		{"UnixRootEscape", "/..", "/"},
		{"UnixRootEscape2", "/../foo", "/foo"},
		{"UnixLocalDot", "./foo", "./foo"},
		{"UnixLocalDotSlash", "./foo/./bar", "./foo/bar"},
		{"UnixLocalDotDot", "./foo/../bar", "./bar"},
		{"UnixDoubleDotDot", "../../foo", "./../../foo"},
		{"UnixTripleDotDot", "../../../foo", "./../../../foo"},
		{"UnixDotDotMiddle", "a/../../b", "./../b"},

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
		{"URLSimple", "https://example.com/foo/bar",
			"https://example.com/foo/bar"},
		{"URLDot", "https://example.com/foo/./bar",
			"https://example.com/foo/bar"},
		{"URLDotDot", "https://example.com/foo/../bar",
			"https://example.com/bar"},
		{"URLRootEscape", "https://example.com/..",
			"https://example.com/"},
		{"URLRootEscape2", "https://example.com/../foo",
			"https://example.com/foo"},
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

func TestSegments(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		// Unix-style
		{"UnixRel", "foo/bar", []string{"foo", "bar"}},
		{"UnixAbs", "/foo/bar", []string{"foo", "bar"}},
		{"UnixNested", "/a/b/c/d", []string{"a", "b", "c", "d"}},
		{"UnixDotDot", "../foo", []string{"..", "foo"}},
		{"UnixDoubleDotDot", "../../foo", []string{"..", "..", "foo"}},
		{"UnixLocalDot", "./foo/bar", []string{"foo", "bar"}},
		{"UnixDot", ".", nil},
		{"UnixEmpty", "", nil},
		{"UnixRoot", "/", nil},
		{"UnixSingle", "foo", []string{"foo"}},
		{"UnixTrailing", "foo/bar/", []string{"foo", "bar"}},

		// Windows-style
		{"WinAbs", `C:\foo\bar`, []string{"foo", "bar"}},
		{"WinRoot", `C:\`, nil},
		{"WinRel", `foo\bar`, []string{"foo", "bar"}},
		{"WinLocalDot", `.\foo\bar`, []string{"foo", "bar"}},
		{"WinTrailing", `foo\bar\`, []string{"foo", "bar"}},

		// URL-style
		{"URLPath", "https://example.com/foo/bar",
			[]string{"foo", "bar"}},
		{"URLRoot", "https://example.com/", nil},
		{"URLRootNoSlash", "https://example.com", nil},
		{"URLS3", "s3://bucket/key/path",
			[]string{"key", "path"}},

		// Mixed separators: first separator determines style.
		{"MixedFwdFirst", `./foo\bar`, []string{`foo\bar`}},
		{"MixedBackFirst", `.\foo/bar`, []string{"foo/bar"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments(tt.path)
			if !slices.Equal(got, tt.want) {
				t.Errorf("segments(%q) = %v, want %v",
					tt.path, got, tt.want)
			}
		})
	}
}

func TestRel(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		targ    string
		want    string
		wantErr bool
	}{
		// Unix-style: same directory
		{"UnixIdentity", "/a/b", "/a/b", ".", false},
		{"UnixDotIdentity", ".", ".", ".", false},
		{"UnixEmptyIdentity", "", "", ".", false},

		// Unix-style: relative paths
		{"UnixRelSimple", "a", "b", "../b", false},
		{"UnixRelNested", "a/b", "a/c", "../c", false},
		{"UnixRelDeeper", "a/b", "a/b/c", "c", false},
		{"UnixRelUp", "a/b/c", "a/b", "..", false},
		{"UnixRelUpTwo", "a/b/c", "a", "../..", false},
		{"UnixRelNoCommon", "a/b", "c/d", "../../c/d", false},
		{"UnixRelDotBase", ".", "a/b", "a/b", false},
		{"UnixRelDotTarg", "a/b", ".", "../..", false},
		{"UnixRelDotDotCommon", "../a", "../b", "../b", false},
		{"UnixRelLocalDot", "./a", "./b", "../b", false},
		{"UnixRelLocalDotMixed", "./a", "b", "../b", false},

		// Unix-style: absolute paths
		{"UnixAbsSimple", "/a/b", "/a/c", "../c", false},
		{"UnixAbsChild", "/a/b", "/a/b/c", "c", false},
		{"UnixAbsParent", "/a/b/c", "/a/b", "..", false},
		{"UnixAbsNoCommon", "/a/b", "/c/d", "../../c/d", false},
		{"UnixAbsRoot", "/", "/a/b", "a/b", false},
		{"UnixAbsToRoot", "/a/b", "/", "../..", false},

		// Unix-style: errors
		{"UnixMixed", "/a", "b", "", true},
		{"UnixMixedReverse", "a", "/b", "", true},
		{"UnixDotDotBase", "../../a", "../b", "", true},

		// Windows-style: same drive
		{"WinSameDrive", `C:\a\b`, `C:\a\c`, `..\c`, false},
		{"WinChild", `C:\a\b`, `C:\a\b\c`, `c`, false},
		{"WinParent", `C:\a\b\c`, `C:\a\b`, `..`, false},
		{"WinRoot", `C:\`, `C:\a\b`, `a\b`, false},
		{"WinToRoot", `C:\a\b`, `C:\`, `..\..`, false},
		{"WinCaseInsensitiveDrive", `C:\a`, `c:\a\b`, `b`, false},

		// Windows-style: errors
		{"WinDiffDrive", `C:\a`, `D:\a`, "", true},

		// URL-style: same host
		{"URLSameHost", "https://example.com/a/b",
			"https://example.com/a/c", "../c", false},
		{"URLChild", "https://example.com/a",
			"https://example.com/a/b", "b", false},
		{"URLParent", "https://example.com/a/b",
			"https://example.com/a", "..", false},
		{"URLRoot", "https://example.com",
			"https://example.com/a/b", "a/b", false},
		{"URLRootSlash", "https://example.com/",
			"https://example.com/a", "a", false},
		{"URLToRoot", "https://example.com/a",
			"https://example.com", "..", false},
		{"URLS3SameBucket", "s3://bucket/a/b",
			"s3://bucket/a/c", "../c", false},
		{"URLCaseInsensitiveHost", "https://Example.Com/a",
			"https://example.com/a/b", "b", false},

		// URL-style: errors
		{"URLDiffHost", "https://a.com/foo",
			"https://b.com/foo", "", true},
		{"URLDiffProto", "https://example.com/a",
			"http://example.com/a", "", true},
		{"URLS3DiffBucket", "s3://bucket1/key",
			"s3://bucket2/key", "", true},

		// Mixed absolute styles: errors (different filesystems)
		{"MixedAbsUnixWin", "/unix/path", `C:\win\path`, "", true},
		{"MixedAbsWinUnix", `C:\win\path`, "/unix/path", "", true},
		{"MixedAbsUnixURL", "/unix/path",
			"https://example.com/foo", "", true},
		{"MixedAbsURLUnix", "https://example.com/foo",
			"/unix/path", "", true},
		{"MixedAbsWinURL", `C:\win\path`,
			"https://example.com/foo", "", true},
		{"MixedAbsURLWin", "https://example.com/foo",
			`C:\win\path`, "", true},

		// Mixed separators in relative paths: uses basepath's style
		{"MixedSepBaseUnix", "./foo", `bar\baz`, "../bar/baz", false},
		{"MixedSepBaseWin", `.\foo`, "bar/baz", `..\bar\baz`, false},
		{"MixedSepRelUnix", "foo/bar", `baz\qux`, "../../baz/qux", false},
		{"MixedSepRelWin", `foo\bar`, "baz/qux", `..\..\baz\qux`, false},
		{"MixedSepChild", "foo", `foo\bar`, "bar", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Rel(tt.base, tt.targ)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Rel(%q, %q) = %q, want error",
						tt.base, tt.targ, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Rel(%q, %q) error: %v",
					tt.base, tt.targ, err)
			}
			if got != tt.want {
				t.Errorf("Rel(%q, %q) = %q, want %q",
					tt.base, tt.targ, got, tt.want)
			}

			// Verify roundtrip: Join(base, Rel(base, targ))
			// must produce equivalent segments to targ.
			gotSeg := segments(Join(tt.base, got))
			wantSeg := segments(tt.targ)
			if !slices.Equal(gotSeg, wantSeg) {
				t.Errorf(
					"roundtrip: segments(Join(%q, %q)) = %v, want %v",
					tt.base, got, gotSeg, wantSeg,
				)
			}
		})
	}
}

func TestVolume(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"Unix", "/foo/bar", ""},
		{"UnixRoot", "/", ""},
		{"UnixRel", "foo/bar", ""},
		{"WinDrive", `C:\foo\bar`, "C:"},
		{"WinRoot", `C:\`, "C:"},
		{"WinLower", `c:\foo`, "c:"},
		{"WinRel", `foo\bar`, ""},
		{"URL", "https://example.com/foo", "https://example.com"},
		{"URLRoot", "https://example.com/", "https://example.com"},
		{"URLNoSlash", "https://example.com", "https://example.com"},
		{"S3", "s3://bucket/key", "s3://bucket"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := detectStyle([]string{tt.path})
			got := volume(tt.path, style)
			if got != tt.want {
				t.Errorf(
					"volume(%q) = %q, want %q",
					tt.path, got, tt.want,
				)
			}
		})
	}
}

// FuzzRel tests the roundtrip property with arbitrary inputs: for any pair
// where Rel succeeds, joining base with the result must yield the same
// segments as targ.
func FuzzRel(f *testing.F) {
	f.Fuzz(func(t *testing.T, base, targ string) {
		base = Clean(base)
		targ = Clean(targ)

		// A forward slash cannot appear in a filename on any real
		// filesystem. A backslash in a segment is valid on Unix but
		// becomes a separator when mixed with Windows-style paths.
		// Skip these impossible cross-style filenames.
		for _, seg := range append(segments(base), segments(targ)...) {
			if strings.ContainsAny(seg, `/\`) {
				t.Skip("segment contains separator character")
			}
		}

		rel, err := Rel(base, targ)
		if err != nil {
			return
		}
		got := segments(Join(base, rel))
		want := segments(targ)
		if !slices.Equal(got, want) {
			t.Errorf(
				"Rel(%q, %q) = %q; segments(Join) = %v, want %v",
				base, targ, rel, got, want)
		}
	})
}

// FuzzJoinSplit tests that Split/Join round-trips on canonical paths:
// for any path p, Join(Split(Clean(p))) == Clean(p).
//
// The comparison is an exact string match, not per-segment. This
// ensures that path style (Unix/Windows/URL) survives the roundtrip,
// which matters for callers doing naive path manipulation — e.g.
// splitting a Windows path to insert a directory, then joining it
// back. Without style preservation, the result could silently change
// separators and break on another OS.
func FuzzJoinSplit(f *testing.F) {
	f.Fuzz(func(t *testing.T, p string) {
		p = Clean(p)

		// Skip paths where Clean is not idempotent. This happens with
		// degenerate mixed-separator paths (e.g., \ and / interleaved)
		// that don't represent real filesystem paths.
		if Clean(p) != p {
			t.Skip("Clean not idempotent")
		}

		dir, file := Split(p)

		got := Join(dir, file)
		if got != p {
			t.Errorf(
				"Join(Split(%q)) = %q (dir=%q, file=%q)",
				p, got, dir, file)
		}
	})
}
