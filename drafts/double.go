package double

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func notmain1() {

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	discordToken := os.Getenv("DISCORD_TOKEN")

	dg, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Panic(err)
	}
	tg, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", tg.Self.UserName)
	tg.Debug = false
	var ucfg tgbotapi.UpdateConfig = tgbotapi.NewUpdate(0)
	ucfg.Timeout = 60
	upd, _ := tg.GetUpdatesChan(ucfg)

	dg.AddHandler(messageCreate)
	err = dg.Open()
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	go func() {
		for sg := range sc {
			log.Println(sg, "initiating exiting seq.")
			dg.Close()
			<-sc
		}
		log.Println("Exiting!")
	}()

	for {
		select {
		case update := <-upd:
			if update.Message == nil {
				fmt.Println("ASD")
				continue
			}

			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			msg.ReplyToMessageID = update.Message.MessageID

			tg.Send(msg)

		}
	}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	if m.Author.ID == s.State.User.ID {
		return
	}
	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}
	if m.Content == "pong" {
		s.ChannelMessageSend(m.ChannelID, "Ping!")
	}
}

func reverse(s string) (result string) {
	for _, v := range s {
		result = string(v) + result
	}
	return
}
