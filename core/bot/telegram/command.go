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

func (s *Store) AddCommand(command string, f func() string) {
	s.command[command] = f
}

func (s *Store) getCommand(command string) (func() string, bool) {
	f, ok := s.command[command]
	return f, ok
}

func init() {
	BotCommand = NewStore()
	BotCommand.AddCommand("help", func() string {
		return "help"
	})
}
