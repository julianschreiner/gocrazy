package persistance

import "todocli/todo"

type JSONRepository struct {
	Path string
}

func NewJSONRepository(path string) *JSONRepository {
	return &JSONRepository{
		Path: path,
	}
}

func (r *JSONRepository) LoadTodos() ([]todo.Todo, error) {
	// Read from r.Path.
}

func (r *JSONRepository) SaveTodo(todo todo.Todo) error {
	// Write to r.Path.
}
