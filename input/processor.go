package input

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/mebaranov/aioncraft/database"
	"github.com/mebaranov/aioncraft/utility"
)

type Processor struct {
	db *database.Database
}

func NewProcessor(db *database.Database) *Processor {
	return &Processor{db: db}
}

var CraftTypeToName = map[database.CraftType]string{
	database.Alchemy:    "Alchemy",
	database.Armor:      "Armorsmith",
	database.Cooking:    "Cooking",
	database.Tailor:     "Tailoring",
	database.Weapon:     "Weaponsmith",
	database.Handicraft: "Handicraft",
}

func (p *Processor) Work(inputs []InputController) {
	length := len(inputs)
	cmdChan := make(chan Command, 15)
	outChans := make([]chan string, length, length)

	for i := 0; i < length; i++ {
		outChans[i] = make(chan string, 15)
		go inputs[i].Start(cmdChan, outChans[i])
	}

	for {
		cmd := <-cmdChan
		switch cmd.Action {
		case Close:
			cmd.Out <- "Ok. Bye bye."
			return
		case Set:
			cmd.Out <- p.Set(cmd)
		case Price:
			cmd.Out <- p.Price(cmd)
		case Help:
			cmd.Out <- p.Help(cmd)
		}
	}
}

func (p *Processor) Set(cmd Command) string {
	items := p.db.Items[cmd.Race]
	name := strings.ToLower(strings.TrimSpace(cmd.Item))
	for _, it := range items {
		if strings.ToLower(it.Name) == name {
			it.Price.NAReasons = []string{}
			it.Price.Value = cmd.Price
			p.db.SaveNeeded = true

			return fmt.Sprintf("Price (%v) successfully set for item %v (%v)", it.Price, it.Name, it.ID)
		}
	}

	return fmt.Sprintf("Item (%v) was not found.", cmd.Item)
}

func (p *Processor) Price(cmd Command) string {
	items := p.db.Items[cmd.Race]
	rv := ""
	regEx := regexp.MustCompile(cmd.Item)
	naReasons := &utility.TheInt{Value: 0, NAReasons: []string{}}

	for _, item := range items {
		if regEx.MatchString(item.Name) {
			for ct, name := range CraftTypeToName {
				rec := p.db.RecipeByItem(cmd.Race, ct, item.ID)
				if rec == nil {
					continue
				}
				price := p.priceByRecipe(cmd.Race, ct, rec.ID)

				rv += fmt.Sprintf("Type: %v (Level %v), Item: %v (%v), Price: %v", name, rec.Level, item.Name, item.ID, price.Value)
				if len(price.NAReasons) > 0 {
					rv += " + <N/A>."
					naReasons = naReasons.Plus(price)
				}
				rv += "\n"
			}
		}
	}

	if rv == "" {
		rv = fmt.Sprintf("No items found following expression: \"%v\"", cmd.Item)
	} else if len(naReasons.NAReasons) != 0 {
		rv += "\n\nYou can improve estimation quality and get rid of <N/A>'s by adding the following prices:\n" + strings.Join(naReasons.NAReasons, ", ") + "\n"
	}
	return rv
}

func (p *Processor) Help(cmd Command) string {
	items := p.db.Items[cmd.Race]
	rv := ""

	for _, item := range items {
		if item.Name == cmd.Item {
			for ct, name := range CraftTypeToName {
				rec := p.db.RecipeByItem(cmd.Race, ct, item.ID)
				if rec == nil {
					continue
				}
				help := p.gatherIngridients(cmd.Race, ct, rec.ID)
				rv += fmt.Sprintf("Type: %v (Level %v), Item: %v, Manual:\n%v", name, rec.Level, item.Name, help)
				rv += "==========================\n"
			}
		}
	}

	if rv == "" {
		rv = fmt.Sprintf("Item not found: \"%v\"", cmd.Item)
	}
	return rv
}

type itemAndCount struct {
	name  string
	count int
	layer int
}

func (p *Processor) gatherIngridients(race database.Race, ct database.CraftType, recId string) string {

	rec := p.db.Recipes[race][ct][recId]
	item := p.db.Items[race][rec.ItemID]
	queue := []string{recId}
	baseItems := []*itemAndCount{}
	crafts := map[string]*itemAndCount{
		item.ID: {item.Name, 1, 0},
	}

	layer := 0
	for len(queue) > 0 {
		recId = queue[0]
		queue = queue[1:]
		rec = p.db.Recipes[race][ct][recId]

		for id, count := range rec.Items {
			subRec := p.db.RecipeByItem(race, ct, id)
			if subRec == nil {
				baseItems = append(baseItems, &itemAndCount{p.db.Items[race][id].Name, count, -1})
			} else {
				if c, ok := crafts[id]; ok {
					c.count += count
				} else {
					layer += 1
					crafts[id] = &itemAndCount{
						name:  p.db.Items[race][id].Name,
						count: count,
						layer: layer,
					}
					queue = append(queue, subRec.ID)
				}
			}
		}
	}

	layers := []*itemAndCount{}
	for _, it := range crafts {
		layers = append(layers, it)
	}

	sort.SliceStable(layers, func(i int, j int) bool {
		return layers[i].layer < layers[j].layer
	})

	rv := "First you buy: "
	for _, it := range baseItems {
		rv += fmt.Sprintf("%v (%v), ", it.name, it.count)
	}
	rv += "\nThen you craft: "

	for _, it := range layers {
		rv += fmt.Sprintf("--> %v (%v) ", it.name, it.count)
	}
	rv += "\n"

	return rv
}

func (p *Processor) priceByRecipe(race database.Race, ct database.CraftType, id string) *utility.TheInt {
	similarRecs := p.db.Recipes[race][ct]
	rec := similarRecs[id]
	rv := &utility.TheInt{Value: 0}

	for item, count := range rec.Items {
		var recPrice *utility.TheInt

		rec := p.db.RecipeByItem(race, ct, item)
		if rec == nil {
			it := p.db.Items[race][item]
			recPrice = it.Price
		} else {
			recPrice = p.priceByRecipe(race, ct, rec.ID)
		}

		rv = rv.Plus(recPrice.Mul(count))
	}

	return rv
}
