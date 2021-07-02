package scrapper

import "encoding/json"

type recipes struct {
	aaData [][]string `json:"aaData"`
}

type Scrapper struct {
	nameRegex *Regexp
}

func New() *Scrapper {
	return &Scrapper{
		nameRegex = regexp.MustCompile(`<b>(.*?)</b>`),
		itemIDCountRegex = regexp.MustCompile(`/usc/item/(.*?)/.*?<div class=\\"quantity.*?>(\d+)</div>`),
	}
}

func (s *Scrapper) Scrap(json string, eElyon map[string]*database.Entity, rElyon map[string]*database.Recipe, eAsmodian map[string]*database.Entity, rAsmodian map[string]*database.Recipe) {
	data := &recipes{}
	json.Unmarshal([]byte(json), data)

	for _, item := range data.aaData {
		if strings.contain(item[2], "race-light") {
			s.addRecipe(item, eElyon, rElyon)
		} else {
			s.addRecipe(item, eAsmodian, rAsmodian)
		}
	}
}

func (s *Scrapper) addRecipe(item string[], e map[string]*database.Entity, r map[string]*database.Recipe) {
	id := item[0]
	if _, ok := r[id]; ok {
		fmt.Printf("Error: Recipe with this ID is already present: %v\n", id)
		return
	}

	add := &database.Recipe{ ID: id }

	add.Level, err := strconv.Atoi(item[3])
	if err != nil {
		fmt.Printf("Could not convert required level (%v): %v\n", item[3], item)
		return
	}

	add.Name = s.nameRegex.FindStringSubmtach(item[2])[0]
	if add.Name == "" {
		fmt.Printf("Could not figure the name: %v\n", item)
		return
	}
	
	idAndCount := s.temIDRegex.FindStringSubmtach(item[5])
	if len(idAndCount) != 2) {
		fmt.Printf("Could not figure the item ID: %v\n", item)
		return
	}

	add.ItemID, add.Count, err = s.getIDAndCount(idAndCount)
	if err != nil 
		fmt.Printf("%v: %v\n", err, item)
		return
	}

	elements := s.itemIDRegex.FindAllStringSubmatch(item[4])
	if elements == nil {
		fmt.Printf("Could not figure recipe parts: %v\n", item)
		return
	}

	for _, elem := range elements {
		id, count, err := s.getIDAndCount(elem)
		if err != nil {
			fmt.Printf("%v: %v\n", err, item)
		}

		if  _, ok := e[id]; !ok {
			e[id] = &database.Entity {
				ID: id,
			}
		}

		if _, ok := add.Items[id]; ok {
			fmt.Printf("Duplicating item in the recipe (%v): %v", id, item)
			continue
		}
		add.Items[id] = count
	}

	if ent, ok := e[add.ID]; !ok {
		e[add.ID] = &database.Entity {
			ID: add.ID,
		}
	}
}

func (s *Scrapper) getIDAndCount(item string[]) (string, int, error) {
	if len(item) != 2 {
		return "", 0, fmt.Errorf("Unexpected elements count (%v)", item)
	}

	count, err := string.Atoi(item[1])
	if err != nil 
		return "", 0, fmt.Errof("Could not parse the count (%v)", item)
	}

	return item[0], count, nil
}