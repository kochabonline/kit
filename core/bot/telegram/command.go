package telegram

var BotCommand = new(Store)

type Store struct {
	command map[string]func() string
}

func NewStore() *Store {
	return &Store{
		command: make(map[string]func() string),
	}
}

func (s *Store) AddCommand(command string, fn func() string) {
	s.command[command] = fn
}

func (s *Store) getCommand(command string) (func() string, bool) {
	fn, ok := s.command[command]
	return fn, ok
}

func init() {
	BotCommand = NewStore()
}
