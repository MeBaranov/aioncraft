package input

import "github.com/mebaranov/aioncraft/database"

type ActionType int

const (
	Price = ActionType(iota)
	Help
	Set
	Close
)

type Command struct {
	Action ActionType
	Race   database.Race
	Item   string
	Price  int
	Out    chan string
}

type InputController interface {
	Start(cmd chan Command, out chan string)
}
