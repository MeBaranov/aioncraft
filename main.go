package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"time"

	"cloud.google.com/go/storage"

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
	client    *storage.Client
	bucket    *storage.BucketHandle
	ctx       context.Context
}

func main() {
	var (
		discToken string
		gcsBucket string
		cli       bool
	)

	flag.StringVar(&discToken, "t", "", "Bot token")
	flag.BoolVar(&cli, "cli", false, "Use CLI")
	flag.StringVar(&gcsBucket, "b", "", "GCS Bucket")
	flag.Parse()

	if discToken == "" {
		discToken = os.Getenv("BOT_TOKEN")
	}
	if gcsBucket == "" {
		gcsBucket = os.Getenv("GCS_BUCKET")
	}

	m := &MainStr{
		scrap: scrapper.New(),
		ctx:   context.Background(),
	}
	var err error

	if gcsBucket != "" {
		m.client, err = storage.NewClient(context.Background())
		if err != nil {
			fmt.Printf("Could not create client. Error: %v", err)
			return
		}
		defer m.client.Close()

		m.bucket = m.client.Bucket(gcsBucket)
	}

	err = m.InitDatabase()
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

	controllers := []input.InputController{}
	if cli {
		controllers = append(controllers, &input.CLI{})
	}
	m.InitDiscord(discToken)
	if m.discInp != nil {
		controllers = append(controllers, m.discInp)
	}
	if len(controllers) == 0 {
		fmt.Println("Nothing to start. I'm out")
		return
	}

	port := os.Getenv("PORT")
	if port != "nil" {
		go m.listen(port)
	}

	go m.Saver()
	m.processor.Work(controllers)
}

func (m *MainStr) listen(port string) {
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + ":" + port)
	for {
		// Listen for an incoming connection.
		_, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
		}
	}
}

func (m *MainStr) dbFromReader(r io.Reader) error {
	data, err := ioutil.ReadAll(r)
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

func (m *MainStr) discFromReader(r io.Reader) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("Could not read file. Error: %v", err)
	}

	m.discInp, err = input.NewDiscordFromJson(data)
	if err != nil {
		return fmt.Errorf("Could not unmarshal discord. Error: %v", err)
	}

	fmt.Printf("Info: Discord loaded from file\n")
	return nil
}

func (m *MainStr) InitDatabase() error {
	if m.bucket != nil {
		rc, err := m.bucket.Object(dbPath).NewReader(m.ctx)
		if err == nil {
			return m.dbFromReader(rc)
		}
	}

	if file, err := os.Open(dbPath); err == nil {
		err := m.dbFromReader(file)
		m.db.SaveNeeded = m.bucket != nil
		return err
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

func (m *MainStr) InitDiscord(token string) error {
	if m.bucket != nil {
		rc, err := m.bucket.Object(discPath).NewReader(m.ctx)
		if err == nil {
			return m.discFromReader(rc)
		}
	}

	if file, err := os.Open(discPath); err == nil {
		err := m.discFromReader(file)
		m.discInp.SaveNeeded = m.bucket != nil
		return err
	}

	if token != "" {
		m.discInp = input.NewDiscord(token)
	}

	return nil
}

func (m *MainStr) SaveDatabase() error {
	data, err := m.db.Save()
	if err != nil {
		return fmt.Errorf("Could not marshal DB. Error: %v", err)
	}

	if m.bucket != nil {
		wc := m.bucket.Object(dbPath).NewWriter(m.ctx)
		_, err = wc.Write(data)
	} else {
		err = ioutil.WriteFile(dbPath, data, 0777)
	}

	if err != nil {
		return fmt.Errorf("Could not save DB file. Error: %v", err)
	}

	return nil
}

func (m *MainStr) SaveDiscord() error {
	data, err := m.discInp.Save()
	if err != nil {
		return fmt.Errorf("Could not marshal Discord. Error: %v", err)
	}

	if m.bucket != nil {
		wc := m.bucket.Object(discPath).NewWriter(m.ctx)
		_, err = wc.Write(data)
	} else {
		err = ioutil.WriteFile(discPath, data, 0777)
	}

	if err != nil {
		return fmt.Errorf("Could not save Discord file. Error: %v", err)
	}

	return nil
}

func (m *MainStr) Saver() {
	for {
		if m.db.SaveNeeded {
			m.SaveDatabase()
		}
		if m.discInp != nil && m.discInp.SaveNeeded {
			m.SaveDiscord()
		}
		time.Sleep(time.Second * 30)
	}
}
