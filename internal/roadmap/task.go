package roadmap

// Task holds parsed front-matter from a roadmap task file.
type Task struct {
	Path      string
	FileID    string
	ID        string
	Title     string
	Type      string
	Priority  string
	DependsOn []string
}
