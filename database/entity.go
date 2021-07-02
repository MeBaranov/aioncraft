package database

import (
	"github.com/mebaranov/aioncraft/utility"
)

type Item struct {
	Name  string
	ID    string
	Price *utility.TheInt
}

type Recipe struct {
	Name   string
	ID     string
	ItemID string
	Level  int
	Count  int
	Items  map[string]int
}
