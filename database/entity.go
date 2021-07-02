package database

const {
	Elyos = 0,
	Asmodian = 1,
}

type Item Struct {
	Name string,
	ID string,
	Price *utility.TheInt,
}

type Recipe Struct {
	Name string,
	ID string,
	ItemID string,
	Level int,
	Count int,
	Items map[string]int,
}