package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/mebaranov/aioncraft/database"
	"github.com/mebaranov/aioncraft/input"
	"github.com/mebaranov/aioncraft/scrapper"
)

var paths = map[database.CraftType]string{
	database.Alchemy:    "data/alchemy.json",
	database.Armor:      "data/armorsmith.json",
	database.Cooking:    "data/cooking.json",
	database.Handicraft: "data/handiwork.json",
	database.Tailor:     "data/tailoring.json",
	database.Weapon:     "data/weaponsmith.json",
}

var dbPath = "data/database.json"
var discPath = "data/discord.json"

type MainStr struct {
	db        *database.Database
	scrap     *scrapper.Scrapper
	processor *input.Processor
	discInp   *input.Discord
}

func main() {
	var (
		discToken string
		cli       bool
	)

	flag.StringVar(&discToken, "t", "", "Bot token")
	flag.BoolVar(&cli, "cli", false, "Use CLI")

	m := &MainStr{
		scrap: scrapper.New(),
	}

	err := m.InitDatabase()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if m.db.CurState != database.Named {
		fmt.Printf("Naming %v + %v items", len(m.db.Items[database.Elyos]), len(m.db.Items[database.Asmodian]))
		m.scrap.Name(m.db.Items[database.Elyos])
		m.scrap.Name(m.db.Items[database.Asmodian])
		m.db.CurState = database.Named

		m.SaveDatabase()
	}

	m.processor = input.NewProcessor(m.db)
	go m.Saver()

	controllers := []input.InputController{}
	if cli {
		controllers = append(controllers, &input.CLI{})
	}
	if discToken != "" {
		m.discInp = input.NewDiscord(discToken)
		controllers = append(controllers, m.discInp)
	}
	m.processor.Work(controllers)
}

func (m *MainStr) InitDatabase() error {
	if file, err := os.Open(dbPath); err == nil {
		data, err := ioutil.ReadAll(file)
		if err != nil {
			return fmt.Errorf("Could not read file. Error: %v", err)
		}

		m.db, err = database.NewFromJson(data)
		if err != nil {
			return fmt.Errorf("Could not unmarshal database. Error: %v", err)
		}

		fmt.Printf("Info: DB loaded from file\n")
		return nil
	}

	m.db = database.New()
	for t, p := range paths {
		file, err := os.Open(p)
		if err != nil {
			fmt.Printf("Could not open file (%v). Error: %v", p, err)
			continue
		}

		data, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Printf("Could not read file with data (%v). Error: %v", p, err)
		}

		m.scrap.Scrap(data, m.db.Items[database.Elyos], m.db.Recipes[database.Elyos][t], m.db.Items[database.Asmodian], m.db.Recipes[database.Asmodian][t])
	}

	m.db.CurState = database.Scrapped
	err := m.SaveDatabase()
	fmt.Printf("DB scrapped and saved\n")

	return err
}

func (m *MainStr) SaveDatabase() error {
	data, err := m.db.Save()
	if err != nil {
		return fmt.Errorf("Could not marshal DB. Error: %v", err)
	}

	err = ioutil.WriteFile(dbPath, data, 0777)
	if err != nil {
		return fmt.Errorf("Could not save DB file. Error: %v", err)
	}

	return nil
}

func (m *MainStr) SaveDiscord() error {
	data, err := m.db.Save()
	if err != nil {
		return fmt.Errorf("Could not marshal DB. Error: %v", err)
	}

	err = ioutil.WriteFile(dbPath, data, 0777)
	if err != nil {
		return fmt.Errorf("Could not save DB file. Error: %v", err)
	}

	return nil
}

func (m *MainStr) Saver() {
	for {
		if m.db.SaveNeeded {
			m.db.SaveNeeded = false
			m.SaveDatabase()
		}
		if m.discInp != nil && m.discInp.SaveNeeded {
			m.discInp.SaveNeeded = false
			m.SaveDiscord()
		}
		time.Sleep(time.Second * 30)
	}
}
