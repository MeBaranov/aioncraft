package input

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mebaranov/aioncraft/database"
	"github.com/mebaranov/aioncraft/utility"
)

var intents = [...]discordgo.Intent{
	discordgo.IntentsGuildMessages,
	discordgo.IntentsDirectMessages,
}

type Guild struct {
	Race           database.Race
	IsRaceSelected bool
	cmdc           chan Command
	outc           chan string
}

type Discord struct {
	Token      string
	Guilds     map[string]*Guild
	SaveNeeded bool
	s          *discordgo.Session
	readyChan  chan bool
	cmdc       chan Command
	outc       chan string
}

func NewDiscord(token string) *Discord {
	return &Discord{
		Token:      token,
		Guilds:     map[string]*Guild{},
		SaveNeeded: true,
	}
}

func NewDiscordFromJson(data []byte) (*Discord, error) {
	rv := &Discord{}
	err := json.Unmarshal(data, rv)
	rv.SaveNeeded = false

	return rv, err
}

const timeout = time.Second * 10

func (d *Discord) Start(cmdc chan Command, outc chan string) {
	var err error
	d.s, err = discordgo.New("Bot " + d.Token)
	if err != nil {
		panic(fmt.Sprintf("Could not start discord part. Error: %v\n", err))
	}
	d.readyChan = make(chan bool)
	d.cmdc = cmdc
	d.outc = outc

	d.s.AddHandler(d.ready)
	d.s.AddHandler(d.guildCreate)
	d.s.AddHandler(d.messageCreate)
	err = d.s.Open()
	if err != nil {
		panic(err)
	}

	select {
	case <-d.readyChan:
		fmt.Println("Bot connected sucessfully")
	case <-time.After(timeout):
		panic("Bot could not connect in time with token: " + d.Token)
	}
}

func (d *Discord) Save() ([]byte, error) {
	d.SaveNeeded = false
	return json.Marshal(d)
}

func (d *Discord) ready(s *discordgo.Session, r *discordgo.Ready) {
	d.readyChan <- true
}

func (d *Discord) guildCreate(s *discordgo.Session, r *discordgo.GuildCreate) {
	gid := r.Guild.ID
	if g, ok := d.Guilds[gid]; ok {
		g.cmdc = d.cmdc
		g.outc = d.outc
		return
	}
	d.Guilds[gid] = &Guild{
		cmdc:           d.cmdc,
		outc:           d.outc,
		IsRaceSelected: false,
	}

	fmt.Printf("Added guild with ID: %v, Name: %v\n", r.Guild.ID, r.Guild.Name)
}

func (d *Discord) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || m.Author.Bot || !strings.HasPrefix(m.Content, "/c ") {
		return
	}
	g := d.Guilds[m.GuildID]
	if g == nil {
		return
	}

	msg := strings.TrimSpace(m.Content[3:])
	cmds := strings.SplitN(msg, " ", 2)

	cmd := strings.ToLower(cmds[0])
	switch cmd {
	case "race":
		r := strings.ToLower(cmds[1])
		switch r {
		case "1", "elyos":
			g.Race = database.Elyos
			g.IsRaceSelected = true
			d.SaveNeeded = true
			msg := "Race is set to Elyos"
			utility.SendMonitored(s, &m.ChannelID, &msg)
		case "2", "asmodian":
			g.Race = database.Asmodian
			g.IsRaceSelected = true
			d.SaveNeeded = true
			msg := "Race is set to Asmodian"
			utility.SendMonitored(s, &m.ChannelID, &msg)
		default:
			msg := "Wrong race selected"
			utility.SendMonitored(s, &m.ChannelID, &msg)
		}
		return
	case "set":
		if !g.IsRaceSelected {
			msg := "Select the race first (see /c help)"
			utility.SendMonitored(s, &m.ChannelID, &msg)
			return
		}
		idx := strings.LastIndex(cmds[1], " ")
		priceStr := strings.TrimSpace(cmds[1][idx+1:])
		item := strings.TrimSpace(cmds[1][:idx])
		if priceStr == "" || item == "" {
			msg := "Wrong command format: Could not find item or price section"
			utility.SendMonitored(s, &m.ChannelID, &msg)
			return
		}

		price, err := strconv.Atoi(priceStr)
		if err != nil {
			msg := "Wrong command format: Could not parse price: " + priceStr
			utility.SendMonitored(s, &m.ChannelID, &msg)
			return
		}

		g.cmdc <- Command{
			Action: Set,
			Race:   g.Race,
			Item:   item,
			Price:  price,
			Out:    g.outc,
		}
		msg := <-g.outc
		utility.SendMonitored(s, &m.ChannelID, &msg)
	case "price":
		if !g.IsRaceSelected {
			msg := "Select the race first (see /c help)"
			utility.SendMonitored(s, &m.ChannelID, &msg)
			return
		}

		g.cmdc <- Command{
			Action: Price,
			Race:   g.Race,
			Item:   cmds[1],
			Out:    g.outc,
		}
		msg := <-g.outc
		utility.SendMonitored(s, &m.ChannelID, &msg)
	case "how":
		if !g.IsRaceSelected {
			msg := "Select the race first (see /c help)"
			utility.SendMonitored(s, &m.ChannelID, &msg)
			return
		}

		g.cmdc <- Command{
			Action: Help,
			Race:   g.Race,
			Item:   cmds[1],
			Out:    g.outc,
		}
		msg := <-g.outc
		utility.SendMonitored(s, &m.ChannelID, &msg)
	case "help":
		msg := "" +
			"Following commands are supported: \n" +
			"\t'/c help - show this help\n'" +
			"\t'/c set <item name> <price>' - set a price for an item. Exact name is required.\n" +
			"\t'/c price <item name>' - shows a craft price estimate. You can use regular expressions for the name.\n" +
			"\t'/c how <item name>' - shows how to craft an item. Exact name is required."
		if !g.IsRaceSelected {
			msg = "You should select a race using one of the following commands:\n\t'/c race Elyos' - for Elyos\n\t'/c race Asmodian' - for Asmodian.\n\n You can change the race in the future."
		}

		msg += "\n\nTo add me to your server use this link: https://discord.com/oauth2/authorize?client_id=862485931013177354&scope=bot+messages.read\n"
		msg += "My source code is there: https://github.com/MeBaranov/aioncraft"
		utility.SendMonitored(s, &m.ChannelID, &msg)
	default:
		msg := fmt.Sprintf("Command \"%v\" is not known", cmd)
		utility.SendMonitored(s, &m.ChannelID, &msg)
		return
	}
}
