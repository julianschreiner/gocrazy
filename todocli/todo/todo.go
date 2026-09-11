package todo

type Todo struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

func (t *Todo) storeTodo() error {
}

func (t *Todo) getTodos() (*Todo, error) {
}
