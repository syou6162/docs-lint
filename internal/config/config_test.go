package config

import (
	"strings"
	"testing"
)

const validConfig = `
rules:
  - name: roadmap-task
    include:
      - "docs/roadmap/**/*.md"
    exclude:
      - "**/overview.md"
    filename_field: id
    fields:
      id:
        type: string
        required: true
        pattern: "^[a-z0-9]+(-[a-z0-9]+)*$"
        unique: true
      priority:
        type: string
        required: true
        enum: [high, medium, low]
      depends_on:
        type: string_array
        required: true
        references: id
        acyclic: true
`

func TestParseValid(t *testing.T) {
	cfg, err := Parse([]byte(validConfig))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(cfg.Rules) != 1 {
		t.Fatalf("len(Rules) = %d, want 1", len(cfg.Rules))
	}

	rule := cfg.Rules[0]
	if got, want := rule.Name, "roadmap-task"; got != want {
		t.Errorf("Name = %q, want %q", got, want)
	}
	if got, want := len(rule.FieldNames()), 3; got != want {
		t.Errorf("len(FieldNames()) = %d, want %d", got, want)
	}
	if re := rule.Fields["id"].Regexp(); re == nil || !re.MatchString("scheduler-slack-auth") {
		t.Errorf("id pattern did not compile or does not match a valid id")
	}
	if rule.AllowUnknownFields {
		t.Errorf("AllowUnknownFields = true, want false by default")
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{
			name:    "unknown key",
			config:  "rules:\n  - name: a\n    includes: [\"*.md\"]\n",
			wantErr: "field includes not found",
		},
		{
			name:    "no rules",
			config:  "rules: []\n",
			wantErr: "rules must not be empty",
		},
		{
			name:    "missing name",
			config:  "rules:\n  - include: [\"*.md\"]\n    fields:\n      id:\n        type: string\n",
			wantErr: "name must not be empty",
		},
		{
			name:    "duplicate rule name",
			config:  "rules:\n  - name: a\n    include: [\"*.md\"]\n    fields:\n      id:\n        type: string\n  - name: a\n    include: [\"*.md\"]\n    fields:\n      id:\n        type: string\n",
			wantErr: `duplicate rule name "a"`,
		},
		{
			name:    "empty include",
			config:  "rules:\n  - name: a\n    include: []\n    fields:\n      id:\n        type: string\n",
			wantErr: "include must not be empty",
		},
		{
			name:    "empty fields",
			config:  "rules:\n  - name: a\n    include: [\"*.md\"]\n    fields: {}\n",
			wantErr: "fields must not be empty",
		},
		{
			name:    "missing type",
			config:  "rules:\n  - name: a\n    include: [\"*.md\"]\n    fields:\n      id:\n        required: true\n",
			wantErr: "type must be set",
		},
		{
			name:    "invalid type",
			config:  "rules:\n  - name: a\n    include: [\"*.md\"]\n    fields:\n      id:\n        type: number\n",
			wantErr: `type "number" is invalid`,
		},
		{
			name:    "invalid pattern",
			config:  "rules:\n  - name: a\n    include: [\"*.md\"]\n    fields:\n      id:\n        type: string\n        pattern: \"([\"\n",
			wantErr: "pattern is not a valid regexp",
		},
		{
			name:    "enum on array",
			config:  "rules:\n  - name: a\n    include: [\"*.md\"]\n    fields:\n      tags:\n        type: string_array\n        enum: [a]\n",
			wantErr: "enum is only supported for type string",
		},
		{
			name:    "unique on array",
			config:  "rules:\n  - name: a\n    include: [\"*.md\"]\n    fields:\n      tags:\n        type: string_array\n        unique: true\n",
			wantErr: "unique is only supported for type string",
		},
		{
			name:    "filename_field not defined",
			config:  "rules:\n  - name: a\n    include: [\"*.md\"]\n    filename_field: slug\n    fields:\n      id:\n        type: string\n",
			wantErr: `filename_field "slug" is not defined`,
		},
		{
			name:    "references not defined",
			config:  "rules:\n  - name: a\n    include: [\"*.md\"]\n    fields:\n      depends_on:\n        type: string_array\n        references: id\n",
			wantErr: `references "id" is not defined`,
		},
		{
			name:    "acyclic without references",
			config:  "rules:\n  - name: a\n    include: [\"*.md\"]\n    fields:\n      depends_on:\n        type: string_array\n        acyclic: true\n",
			wantErr: "acyclic requires references",
		},
		{
			name:    "acyclic on string",
			config:  "rules:\n  - name: a\n    include: [\"*.md\"]\n    fields:\n      id:\n        type: string\n      parent:\n        type: string\n        references: id\n        acyclic: true\n",
			wantErr: "acyclic is only supported for type string_array",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse([]byte(test.config))
			if err == nil {
				t.Fatalf("Parse() error = nil, want error containing %q", test.wantErr)
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("Parse() error = %v, want it to contain %q", err, test.wantErr)
			}
		})
	}
}
