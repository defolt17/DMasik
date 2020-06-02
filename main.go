package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	dg            *discordgo.Session
	stopChannel   chan bool
	discordPrefix = "."
)

func main() {
	var discordToken string

	discordToken = getDiscordToken()
	dg, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Fatal("Error creating Discord session,", err)
	}
	dg.AddHandler(discordMessageHandler)
	err = dg.Open()
	if err != nil {
		log.Println("Error opening connection,", err)
		return
	}
	stopChannel = make(chan bool)

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}

func getDiscordToken() string {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	discordToken := os.Getenv("DISCORD_TOKEN")
	if discordToken == "" {
		log.Fatalf("Error, DISCORD_TOKEN in .env not found")
	}
	return discordToken
}

func discordMessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	var msgIsCommand bool
	var command string

	if m.Author.ID == s.State.User.ID {
		return
	}
	msgIsCommand, command = isCommand(m.Content)

	if msgIsCommand {
		commandArgs := strings.Split(command, " ")
		var res string
		for _, i := range commandArgs {
			res += i + " "
		}
		s.ChannelMessageSend(m.ChannelID, res)
	} else {
		return
	}

}

func isCommand(str string) (bool, string) {
	for i, n := range discordPrefix {
		if n != rune(str[i]) {
			return false, ""
		}
	}

	return true, str[len(discordPrefix):]
}
