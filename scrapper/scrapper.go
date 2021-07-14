package scrapper

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/martian/v3/log"
	"github.com/mebaranov/aioncraft/database"
	"github.com/mebaranov/aioncraft/utility"
)

type recipes struct {
	AaData [][]string `json:"aaData"`
}

type Scrapper struct {
	nameRegex        *regexp.Regexp
	itemIDCountRegex *regexp.Regexp
	itemNameRegex    *regexp.Regexp
}

func New() *Scrapper {
	return &Scrapper{
		nameRegex:        regexp.MustCompile(`<b>(.*?)</b>`),
		itemIDCountRegex: regexp.MustCompile(`(?s)/usc/item/(.*?)/.*?<div class=\\?"quantity.*?>(\d+)</div>`),
		itemNameRegex:    regexp.MustCompile(`\<span class="item_title.*?" id="item_name"\>\s*<b>(.*?)</b>`),
	}
}

func (s *Scrapper) Scrap(in []byte, eElyon map[string]*database.Item, rElyon map[string]*database.Recipe, eAsmodian map[string]*database.Item, rAsmodian map[string]*database.Recipe) {
	data := &recipes{}
	json.Unmarshal(in, data)

	for _, item := range data.AaData {
		if strings.Contains(item[2], "race-light") {
			s.addRecipe(item, eElyon, rElyon)
		} else {
			s.addRecipe(item, eAsmodian, rAsmodian)
		}
	}
}

func (s *Scrapper) addRecipe(item []string, e map[string]*database.Item, r map[string]*database.Recipe) {
	id := item[0]
	if _, ok := r[id]; ok {
		log.Errorf("Error: Recipe with this ID is already present: %v\n", id)
		return
	}

	var err error

	add := &database.Recipe{ID: id}

	add.Level, err = strconv.Atoi(item[3])
	if err != nil {
		log.Errorf("Could not convert required level (%v): %v\n", item[3], item)
		return
	}

	add.Name = s.nameRegex.FindStringSubmatch(item[2])[1]
	if add.Name == "" {
		log.Errorf("Could not figure the name: %v\n", item)
		return
	}

	idAndCount := s.itemIDCountRegex.FindStringSubmatch(item[5])
	add.ItemID, add.Count, err = s.getIDAndCount(idAndCount)
	if err != nil {
		log.Errorf("Error at base recipe. %v: %v\n", err, item[5])
		return
	}

	elements := s.itemIDCountRegex.FindAllStringSubmatch(item[4], -1)
	if elements == nil {
		log.Errorf("Could not figure recipe parts: %v\n", item)
		return
	}

	add.Items = make(map[string]int)
	for _, elem := range elements {
		id, count, err := s.getIDAndCount(elem)
		if err != nil {
			log.Errorf("Error at elements. %v: %v\n", err, item[4])
		}

		if _, ok := e[id]; !ok {
			e[id] = &database.Item{
				ID: id,
			}
		}

		if _, ok := add.Items[id]; ok {
			log.Errorf("Duplicating item in the recipe (%v): %v", id, item[4])
			continue
		}
		add.Items[id] = count
	}

	if _, ok := e[add.ItemID]; !ok {
		e[add.ItemID] = &database.Item{
			ID: add.ItemID,
		}
	}

	r[id] = add
	add.Name = strings.Replace(add.Name, "&#39;", "'", -1)
}

const addressFmt = "https://aioncodex.com/usc/item/%s/"

func (s *Scrapper) Name(items map[string]*database.Item) {
	req := NewRequester()
	for id, item := range items {
		if item.Name == "" {
			data, err := req.GetData(fmt.Sprintf(addressFmt, id))
			if err != nil {
				log.Errorf("Could not load item data (%v). Error: %v", id, err)
				continue
			}

			strData := string(data)
			tmp := s.itemNameRegex.FindStringSubmatch(strData)
			if len(tmp) != 2 {
				log.Errorf("Wrong amount of sections in name (%v). %v\n", id, tmp)
				continue
			}

			item.Name = tmp[1]
			item.Name = strings.Replace(item.Name, "&#39;", "'", -1)
		}
		item.Price = utility.NewInt(0, item.Name)
	}
}

func (s *Scrapper) getIDAndCount(item []string) (string, int, error) {
	if len(item) != 3 {
		return "", 0, fmt.Errorf("Unexpected elements count (%v)", item)
	}

	count, err := strconv.Atoi(item[2])
	if err != nil {
		return "", 0, fmt.Errorf("Could not parse the count (%v)", item)
	}

	return item[1], count, nil
}
