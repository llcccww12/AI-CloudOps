package utils

import (
	"strings"
	"testing"
)

func TestValidateCommentAttachmentFileName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		file    string
		wantExt string
		wantErr bool
	}{
		{"png ok", "shot.png", ".png", false},
		{"pdf ok", "doc.PDF", ".pdf", false},
		{"exe reject", "malware.exe", "", true},
		{"sh reject", "run.sh", "", true},
		{"empty reject", "", "", true},
		{"no ext reject", "readme", "", true},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ext, ct, err := ValidateCommentAttachmentFileName(c.file)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", c.file)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ext != c.wantExt {
				t.Fatalf("ext got %q want %q", ext, c.wantExt)
			}
			if ct == "" {
				t.Fatal("content type empty")
			}
		})
	}
}

func TestSanitizeAttachmentFileName(t *testing.T) {
	got := SanitizeAttachmentFileName("../../etc/passwd.png")
	if strings.Contains(got, "..") || strings.Contains(got, "/") {
		t.Fatalf("unsafe name retained: %q", got)
	}
	if !strings.HasSuffix(got, ".png") {
		t.Fatalf("expected png suffix, got %q", got)
	}
}
