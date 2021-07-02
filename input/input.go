package input

type ActionType int

const (
	Price = ActionType(iota)
	Help
	Set
)

type Command struct {
	Action ActionType
	Item   string
	Price  int
}
