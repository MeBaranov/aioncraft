package utility

import (
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const charLimit = 2000
const delay = 50 * time.Millisecond
const longDelay = 1 * time.Second
const limit = 5

func sendMonitored(s *discordgo.Session, c *string, msg *string) {
	if len(*msg) < charLimit {
		s.ChannelMessageSend(*c, *msg)
		return
	}
	split := strings.Split(*msg, "\n")
	l, count, cur := 0, 0, ""
	for _, str := range split {
		if l+len(str) >= charLimit {
			s.ChannelMessageSend(*c, cur)
			count += 1
			if count >= limit {
				count = 0
				time.Sleep(longDelay)
			}
			time.Sleep(delay)

			cur = str
			l = len(str)
		} else {
			cur += "\n" + str
			l += 1 + len(str)
		}
	}

	if cur != "" {
		s.ChannelMessageSend(*c, cur)
	}
}

func SendMonitored(s *discordgo.Session, c *string, msg *string) {
	go sendMonitored(s, c, msg)
}
