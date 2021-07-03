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
	race           database.Race
	isRaceSelected bool
}

func (c *CLI) Start(cmdc chan Command, outc chan string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Let's begin")
	fmt.Println("------")

	for {
		cmd, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Some error reading input: %v", err)
			continue
		}
		cmd = strings.Trim(cmd, "\n\r ")
		if cmd == "quit" {
			cmdc <- Command{
				Action: Close,
				Race:   c.race,
				Out:    outc,
			}
			fmt.Println(<-outc)
			continue
		}

		cmdArr := strings.Split(cmd, ":")
		switch cmdArr[0] {
		case "race":
			if len(cmdArr) != 2 {
				fmt.Println("Wrong command format")
				continue
			}

			switch strings.ToLower(cmdArr[1]) {
			case "1", "elyos":
				c.race = database.Elyos
				c.isRaceSelected = true
				fmt.Println("Race is set to Elyos")
			case "2", "asmodian":
				c.race = database.Asmodian
				c.isRaceSelected = true
				fmt.Println("Race is set to Asmodian")
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

			cmdc <- Command{
				Action: Set,
				Race:   c.race,
				Item:   cmdArr[1],
				Price:  price,
				Out:    outc,
			}
			fmt.Println(<-outc)
		case "price":
			if !c.isRaceSelected {
				fmt.Println("Select the race first")
				continue
			}
			if len(cmdArr) != 2 {
				fmt.Println("Wrong command format")
				continue
			}

			cmdc <- Command{
				Action: Price,
				Race:   c.race,
				Item:   cmdArr[1],
				Out:    outc,
			}
			fmt.Println(<-outc)
		case "help":
			if !c.isRaceSelected {
				fmt.Println("Select the race first")
				continue
			}
			if len(cmdArr) != 2 {
				fmt.Println("Wrong command format")
				continue
			}

			cmdc <- Command{
				Action: Help,
				Race:   c.race,
				Item:   cmdArr[1],
				Out:    outc,
			}
			fmt.Println(<-outc)
		default:
			fmt.Printf("Command \"%v\" is not known\n", cmd)
		}
	}
}
