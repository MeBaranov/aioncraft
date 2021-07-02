package input

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mebaranov/aioncraft/database"
)

type CLI struct {
	close chan bool
	cmd   chan Command
	out   chan string

	race           database.Race
	isRaceSelected bool
}

func New(close chan bool, cmd chan Command, out chan string) *CLI {
	return &CLI{
		close:          close,
		cmd:            cmd,
		out:            out,
		isRaceSelected: false,
	}
}

func (c *CLI) Start() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Let's begin")
	fmt.Println("------")

	for {
		cmd, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Some error reading input: %v", err)
			continue
		}
		if cmd == "quit" {
			c.close <- true
			return
		}

		cmdArr := strings.Split(cmd, ":")
		switch cmdArr[0] {
		case "race":
			if len(cmdArr) != 2 {
				fmt.Println("Wrong command format")
				continue
			}

			switch strings.ToLower(cmdArr[1]) {
			case "elyos":
			case "1":
				c.race = database.Elyos
				c.isRaceSelected = true
			case "":
			case "2":
				c.race = database.Asmodian
				c.isRaceSelected = true
			default:
				fmt.Println("Wrong race")
			}
		case "set":
			if !c.isRaceSelected {
				fmt.Println("Select the race first")
				continue
			}
			if len(cmdArr) != 3 {
				fmt.Println("Wrong command format")
				continue
			}

			price, err := strconv.Atoi(cmdArr[2])
			if err != nil {
				fmt.Println("Wrong command format")
				continue
			}

			c.cmd <- Command{
				Action: Set,
				Item:   cmdArr[1],
				Price:  price,
			}
			fmt.Println(<-c.out)
		case "price":
			if !c.isRaceSelected {
				fmt.Println("Select the race first")
				continue
			}
			if len(cmdArr) != 2 {
				fmt.Println("Wrong command format")
				continue
			}

			c.cmd <- Command{
				Action: Price,
				Item:   cmdArr[1],
			}
			fmt.Println(<-c.out)
		case "help":
			if !c.isRaceSelected {
				fmt.Println("Select the race first")
				continue
			}
			if len(cmdArr) != 2 {
				fmt.Println("Wrong command format")
				continue
			}

			c.cmd <- Command{
				Action: Help,
				Item:   cmdArr[1],
			}
			fmt.Println(<-c.out)
		}
	}
}
