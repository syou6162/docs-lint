// Package config loads and validates docs-lint rule definitions.
package config

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// FieldType is the YAML type a front-matter field must have.
type FieldType string

const (
	// TypeString requires a non-empty YAML string.
	TypeString FieldType = "string"
	// TypeStringArray requires a YAML sequence of non-empty strings.
	TypeStringArray FieldType = "string_array"
)

// Config is the whole rule set, normally read from docs-lint.yaml.
type Config struct {
	Rules []Rule `yaml:"rules"`
}

// Rule applies a front-matter schema to the files matched by Include.
type Rule struct {
	Name               string           `yaml:"name"`
	Include            []string         `yaml:"include"`
	Exclude            []string         `yaml:"exclude"`
	FilenameField      string           `yaml:"filename_field"`
	AllowUnknownFields bool             `yaml:"allow_unknown_fields"`
	Fields             map[string]Field `yaml:"fields"`

	fieldOrder []string
}

// Field is the schema of a single front-matter field.
type Field struct {
	Type        FieldType `yaml:"type"`
	Required    bool      `yaml:"required"`
	Enum        []string  `yaml:"enum"`
	Pattern     string    `yaml:"pattern"`
	Unique      bool      `yaml:"unique"`
	References  string    `yaml:"references"`
	SelfAllowed bool      `yaml:"self_reference_allowed"`
	Acyclic     bool      `yaml:"acyclic"`

	pattern *regexp.Regexp
}

// Regexp returns the compiled Pattern, or nil when no pattern is configured.
func (f Field) Regexp() *regexp.Regexp {
	return f.pattern
}

// FieldNames returns the field names in the order they are written in the
// config, so that a document is reported by the first field that fails.
func (r Rule) FieldNames() []string {
	if len(r.fieldOrder) == len(r.Fields) {
		return r.fieldOrder
	}
	names := make([]string, 0, len(r.Fields))
	for name := range r.Fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Load reads a config file and validates it.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Parse validates a config document. Unknown keys are rejected so that typos in
// a rule definition fail loudly instead of silently disabling a check.
func Parse(data []byte) (*Config, error) {
	var cfg Config
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	for i, order := range fieldOrders(data) {
		if i < len(cfg.Rules) {
			cfg.Rules[i].fieldOrder = order
		}
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}

// fieldOrders returns, for each rule, the field names in document order.
func fieldOrders(data []byte) [][]string {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil
	}
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = *root.Content[0]
	}

	rules := mappingValue(&root, "rules")
	if rules == nil || rules.Kind != yaml.SequenceNode {
		return nil
	}

	orders := make([][]string, 0, len(rules.Content))
	for _, rule := range rules.Content {
		var names []string
		if fields := mappingValue(rule, "fields"); fields != nil {
			for i := 0; i+1 < len(fields.Content); i += 2 {
				names = append(names, fields.Content[i].Value)
			}
		}
		orders = append(orders, names)
	}
	return orders
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func (c *Config) validate() error {
	if len(c.Rules) == 0 {
		return fmt.Errorf("rules must not be empty")
	}

	seen := map[string]struct{}{}
	for i := range c.Rules {
		rule := &c.Rules[i]
		if rule.Name == "" {
			return fmt.Errorf("rules[%d]: name must not be empty", i)
		}
		if _, ok := seen[rule.Name]; ok {
			return fmt.Errorf("duplicate rule name %q", rule.Name)
		}
		seen[rule.Name] = struct{}{}
		if err := rule.validate(); err != nil {
			return fmt.Errorf("rule %q: %w", rule.Name, err)
		}
	}
	return nil
}

func (r *Rule) validate() error {
	if len(r.Include) == 0 {
		return fmt.Errorf("include must not be empty")
	}
	if len(r.Fields) == 0 {
		return fmt.Errorf("fields must not be empty")
	}
	if r.FilenameField != "" {
		field, ok := r.Fields[r.FilenameField]
		if !ok {
			return fmt.Errorf("filename_field %q is not defined in fields", r.FilenameField)
		}
		if field.Type != TypeString {
			return fmt.Errorf("filename_field %q must be of type string", r.FilenameField)
		}
	}

	for _, name := range r.FieldNames() {
		field := r.Fields[name]
		if err := r.validateField(name, &field); err != nil {
			return fmt.Errorf("field %q: %w", name, err)
		}
		r.Fields[name] = field
	}
	return nil
}

func (r *Rule) validateField(name string, field *Field) error {
	switch field.Type {
	case TypeString, TypeStringArray:
	case "":
		return fmt.Errorf("type must be set (string or string_array)")
	default:
		return fmt.Errorf("type %q is invalid (expected string or string_array)", field.Type)
	}

	if field.Pattern != "" {
		re, err := regexp.Compile(field.Pattern)
		if err != nil {
			return fmt.Errorf("pattern is not a valid regexp: %w", err)
		}
		field.pattern = re
	}

	if field.Unique && field.Type != TypeString {
		return fmt.Errorf("unique is only supported for type string")
	}
	if len(field.Enum) > 0 && field.Type != TypeString {
		return fmt.Errorf("enum is only supported for type string")
	}

	if field.References != "" {
		target, ok := r.Fields[field.References]
		if !ok {
			return fmt.Errorf("references %q is not defined in fields", field.References)
		}
		if target.Type != TypeString {
			return fmt.Errorf("references %q must be of type string", field.References)
		}
		if field.References == name {
			return fmt.Errorf("references must not point at itself")
		}
	}
	if field.Acyclic {
		if field.Type != TypeStringArray {
			return fmt.Errorf("acyclic is only supported for type string_array")
		}
		if field.References == "" {
			return fmt.Errorf("acyclic requires references")
		}
	}
	if field.SelfAllowed && field.References == "" {
		return fmt.Errorf("self_reference_allowed requires references")
	}
	return nil
}
