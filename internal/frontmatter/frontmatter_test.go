package frontmatter

import (
	"strings"
	"testing"
)

func TestExtract(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr string
	}{
		{
			name:    "lf",
			content: "---\nid: a\n---\n\n# title\n",
			want:    "id: a",
		},
		{
			name:    "crlf",
			content: "---\r\nid: a\r\n---\r\n",
			want:    "id: a\r",
		},
		{
			name:    "no front-matter",
			content: "# title\n",
			wantErr: "missing YAML front-matter",
		},
		{
			name:    "not a fence",
			content: "----\nid: a\n",
			wantErr: "missing YAML front-matter",
		},
		{
			name:    "unclosed",
			content: "---\nid: a\n",
			wantErr: "unclosed YAML front-matter",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Extract(test.content)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("Extract() error = %v, want it to contain %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Extract() error = %v", err)
			}
			if got != test.want {
				t.Errorf("Extract() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	fields, err := Parse("id: a\ntags: [x, y]\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(fields) != 2 {
		t.Fatalf("len(fields) = %d, want 2", len(fields))
	}
	if got := fields["id"].Value; got != "a" {
		t.Errorf("id = %q, want %q", got, "a")
	}
}

func TestParseNonMappingHasNoFields(t *testing.T) {
	fields, err := Parse("- a\n- b\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(fields) != 0 {
		t.Errorf("len(fields) = %d, want 0", len(fields))
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{name: "duplicate field", content: "id: a\nid: b\n", wantErr: `duplicate field "id"`},
		{name: "invalid yaml", content: "id: [a\n", wantErr: "invalid YAML front-matter"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse(test.content)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Parse() error = %v, want it to contain %q", err, test.wantErr)
			}
		})
	}
}
