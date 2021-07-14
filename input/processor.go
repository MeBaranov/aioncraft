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

			return fmt.Sprintf("Price (%v) successfully set for item %v (%v)", it.Price.Value, it.Name, it.ID)
		}
	}

	return fmt.Sprintf("Item (%v) was not found.", cmd.Item)
}

type helpStruct struct {
	str   string
	layer int
}

func (p *Processor) Price(cmd Command) string {
	items := p.db.Items[cmd.Race]
	rv := ""
	regEx := regexp.MustCompile(strings.ToLower(cmd.Item))
	naReasons := map[string]bool{}
	rvs := []*helpStruct{}

	for _, item := range items {
		if regEx.MatchString(strings.ToLower(item.Name)) {
			found := false
			for ct, ctName := range CraftTypeToName {
				rec := p.db.RecipeByItem(cmd.Race, ct, item.ID)
				if rec == nil {
					continue
				}
				price := p.priceByRecipe(cmd.Race, ct, rec.ID, true)

				tmpstr := fmt.Sprintf("Type: %v (Level %v), Item: %v (x%v), Price: %v", ctName, rec.Level, item.Name, rec.Count, price.Value)
				if len(price.NAReasons) > 0 {
					tmpstr += " + <N/A>."
					for _, na := range price.NAReasons {
						naReasons[na] = true
					}
				}
				tmpstr += "\n"
				rvs = append(rvs, &helpStruct{tmpstr, rec.Level + int(ct)*1000})
				found = true
			}

			if !found {
				str := fmt.Sprintf("Type: Base item, Item: %v, Price: %v", item.Name, item.Price.Value)
				if len(item.Price.NAReasons) != 0 {
					str += " (<N/A>)."
				}
				str += "\n"
				rvs = append(rvs, &helpStruct{str, -1})
			}
		}
	}

	if len(rvs) == 0 {
		rv = fmt.Sprintf("No items found following expression: \"%v\"", cmd.Item)
	} else {
		sort.SliceStable(rvs, func(i, j int) bool {
			return rvs[i].layer < rvs[j].layer
		})
		for _, s := range rvs {
			rv += s.str
		}
		if len(naReasons) != 0 {
			rv += "\n\nYou can improve estimation quality and get rid of '<N/A>'s by adding the following prices:\n"
			for i := range naReasons {
				rv += i + ","
			}
			rv += "\n"
		}
	}
	return rv
}

func (p *Processor) Help(cmd Command) string {
	items := p.db.Items[cmd.Race]
	rv := ""

	for _, item := range items {
		if strings.ToLower(item.Name) == strings.ToLower(cmd.Item) {
			for ct, name := range CraftTypeToName {
				rec := p.db.RecipeByItem(cmd.Race, ct, item.ID)
				if rec == nil {
					continue
				}
				help := p.gatherIngridients(cmd.Race, ct, rec.ID)
				rv += fmt.Sprintf("Type: %v (Level %v), Item: %v (x%v), Manual:\n%v", name, rec.Level, item.Name, rec.Count, help)
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
	price *utility.TheInt
}

type queueItem struct {
	id  string
	mul int
}

func (p *Processor) gatherIngridients(race database.Race, ct database.CraftType, inRecId string) string {

	rec := p.db.Recipes[race][ct][inRecId]
	item := p.db.Items[race][rec.ItemID]
	queue := []*queueItem{{inRecId, 1}}
	baseItems := map[string]*itemAndCount{}
	crafts := map[string]*itemAndCount{
		item.ID: {item.Name, rec.Count, 0, nil},
	}

	layer := 0
	for len(queue) > 0 {
		theRec := queue[0]
		queue = queue[1:]
		rec = p.db.Recipes[race][ct][theRec.id]

		for id, count := range rec.Items {
			subRec := p.db.RecipeByItem(race, ct, id)
			if subRec == nil {
				if c, ok := baseItems[id]; ok {
					c.count += count * theRec.mul
				} else {
					theItem := p.db.Items[race][id]
					baseItems[id] = &itemAndCount{theItem.Name, count * theRec.mul, -1, theItem.Price}
				}
			} else {
				layer += 1
				if c, ok := crafts[id]; ok {
					c.layer = layer
					c.count += count * theRec.mul
				} else {
					crafts[id] = &itemAndCount{
						name:  p.db.Items[race][id].Name,
						count: count * theRec.mul,
						layer: layer,
					}
					queue = append(queue, &queueItem{subRec.ID, count * theRec.mul})
				}
			}
		}
	}

	layers := []*itemAndCount{}
	for _, it := range crafts {
		layers = append(layers, it)
	}

	sort.SliceStable(layers, func(i int, j int) bool {
		return layers[i].layer > layers[j].layer
	})

	rv := "First you buy: "
	for _, it := range baseItems {
		prc := "N/A"
		if len(it.price.NAReasons) == 0 {
			prc = fmt.Sprint(it.price.Value)
		}
		rv += fmt.Sprintf("\n\t%v x %v, for %v each, ", it.count, it.name, prc)
	}
	rv += "\nThen you craft: "

	for _, it := range layers {
		rv += fmt.Sprintf("--> %v (%v) ", it.name, it.count)
	}
	rv += "\n"

	return rv
}

func (p *Processor) priceByRecipe(race database.Race, ct database.CraftType, id string, ignoreCount bool) *utility.TheInt {
	similarRecs := p.db.Recipes[race][ct]
	rec := similarRecs[id]
	mainRec := rec
	rv := &utility.TheInt{Value: 0}

	for item, count := range rec.Items {
		var recPrice *utility.TheInt

		rec := p.db.RecipeByItem(race, ct, item)
		if rec == nil {
			it := p.db.Items[race][item]
			recPrice = it.Price
		} else {
			recPrice = p.priceByRecipe(race, ct, rec.ID, false)
		}

		curPrice := recPrice.Mul(count)
		if !ignoreCount {
			curPrice = curPrice.Div(mainRec.Count)
		}
		rv = rv.Plus(curPrice)
	}

	return rv
}
