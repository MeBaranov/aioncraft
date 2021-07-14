package database

import "encoding/json"

type Race int

const (
	Elyos = Race(iota)
	Asmodian
)

type CraftType int

const (
	Handicraft = CraftType(iota)
	Weapon
	Armor
	Tailor
	Alchemy
	Cooking
	Morph
)

type State int

const (
	Created = State(iota)
	Scrapped
	Named
)

var Races = []Race{
	Elyos,
	Asmodian,
}

var Crafts = []CraftType{
	Handicraft,
	Weapon,
	Armor,
	Tailor,
	Alchemy,
	Cooking,
}

type Database struct {
	Recipes    map[Race]map[CraftType]map[string]*Recipe
	Items      map[Race]map[string]*Item
	CurState   State
	SaveNeeded bool
}

func New() *Database {
	rv := &Database{}

	rv.Items = make(map[Race]map[string]*Item)
	rv.Recipes = make(map[Race]map[CraftType]map[string]*Recipe)
	rv.CurState = Created

	for _, r := range Races {
		rv.Recipes[r] = make(map[CraftType]map[string]*Recipe)
		rv.Items[r] = make(map[string]*Item)

		for _, c := range Crafts {
			rv.Recipes[r][c] = make(map[string]*Recipe)
		}
	}

	return rv
}

func NewFromJson(in []byte) (*Database, error) {
	rv := &Database{}
	err := json.Unmarshal(in, rv)
	rv.SaveNeeded = false

	return rv, err
}

func (d *Database) Save() ([]byte, error) {
	d.SaveNeeded = false
	return json.Marshal(d)
}

func (d *Database) RecipeByItem(race Race, ct CraftType, itemId string) *Recipe {
	recipes := d.Recipes[race][ct]
	var possibleRv *Recipe
	for _, r := range recipes {
		if r.ItemID == itemId {
			if r.Count == 1 {
				return r
			}
			if possibleRv == nil || possibleRv.Count > r.Count {
				possibleRv = r
			}
		}
	}

	return possibleRv
}
