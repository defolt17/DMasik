package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/rylio/ytdl"
)

type Voice struct {
	VoiceConnection *discordgo.VoiceConnection
	Channel         string
	Guild           string
	PlayerStatus    int
}

type Song struct {
	Link    string
	Type    string
	Guild   string
	Channel string
}

const ShellToUse string = "bash"

var (
	dg               *discordgo.Session
	stopChannel      chan bool
	commandArgs      []string
	voiceConnections []Voice
	queue            []Song

	nowPlaying Song

	discordPrefix = "."
	commands      = map[string]func(*discordgo.Session, *discordgo.MessageCreate){
		"text":       getText,
		"ping":       pong,
		"pong":       ping,
		"connect":    connectToVC,
		"disconnect": disconnectFromVoiceChannel,
		"join":       connectToVC,
		"leave":      disconnectFromVoiceChannel,
		"j":          connectToVC,
		"l":          disconnectFromVoiceChannel,
		"bruh":       playBruhSound,
		"stal":       playStalMusic,
		"stop":       stopMusic,
		"yt":         playYoutubeLink,
		"play":       playAudioLink,
		"library":    playLibraryMusic,
		"lib":        playLibraryMusic,
		"skip":       nextSong,
		"next":       nextSong,
		"flex":       flex,
	}

	IS_PLAYING     = 0
	IS_NOT_PLAYING = 0

	bruhSoundPath       = "./audio/bruh.opus"
	stalMusicPath       = "./audio/stal.opus"
	imageMeNaniFilePath = "./images/memes/Nani.png"
	imageMeURL          = "https://avatars3.githubusercontent.com/u/22434204?s=460&u=cc62b75ba8a868b3c0af3b2b0ef7df7830963a5b&v=4"

	embedExample = &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{},
		Color:       0x00ff00, // Green
		Description: "This is a discordgo embed",
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "I am a field1",
				Value:  "I am a value2",
				Inline: true,
			},
		},
		Image: &discordgo.MessageEmbedImage{
			URL: imageMeURL,
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: imageMeURL,
		},
		Timestamp: time.Now().Format(time.RFC3339), // Discord wants ISO8601; RFC3339 is an extension of ISO8601 and should be completely compatible.
		Title:     "I am an Embed",
	}
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
	if strings.ToLower(m.Content) == "да" {
		s.ChannelMessageSend(m.ChannelID, "П-ворд")
	}
	msgIsCommand, command = isCommand(m.Content)

	if msgIsCommand {
		commandArgs = strings.Split(command, " ")

		if function, ok := commands[commandArgs[0]]; ok {
			log.Println("Executing {", commandArgs[0], "} command")
			function(s, m)
		} else {
			log.Println("{", commandArgs[0], "} not in map[string]func")
			s.ChannelMessageSend(m.ChannelID, "oWu sowwy but I do not posess such a command, if you would be so kind to contribute to github.com/defolt17/DMasik by adding it or provodong desirable functional.")
		}
	} else {

	}

}

func isCommand(str string) (bool, string) {
	for i, n := range discordPrefix {
		if n != rune(str[i]) {
			return false, ""
		}
	}
	log.Println("[", str, "] starts with [", discordPrefix, "]")
	return true, str[len(discordPrefix):]
}

func getText(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSendEmbed(m.ChannelID, embedExample)
	s.ChannelMessageSend(m.ChannelID, commandArgs[0])
}

func ping(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, "Ping!")
}

func pong(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, "Pong!")
}

func connectToVC(s *discordgo.Session, m *discordgo.MessageCreate) {
	channel, err := s.State.Channel(m.ChannelID)
	if err != nil {
		fmt.Println(err)
	}
	guild, err := s.State.Guild(channel.GuildID)
	if err != nil {
		fmt.Println(err)
	}
	voiceChannel := findVoiceChannelID(guild, m)
	for _, vs := range guild.VoiceStates {
		log.Println(vs.UserID, vs.ChannelID, m.Author.ID, m.Author.Username)

	}
	voiceConnections = append(voiceConnections, connectToVoiceChannel(s, channel.GuildID, voiceChannel))
}

func findVoiceChannelID(guild *discordgo.Guild, message *discordgo.MessageCreate) string {
	var channelID string

	for _, vs := range guild.VoiceStates {
		if vs.UserID == message.Author.ID {
			channelID = vs.ChannelID
		}
	}
	return channelID
}

func connectToVoiceChannel(bot *discordgo.Session, guild string, channel string) Voice {
	vs, err := bot.ChannelVoiceJoin(guild, channel, false, true)

	checkForDoubleVoiceConnection(guild, channel)

	if err != nil {
		fmt.Println(err)
	}
	return Voice{
		VoiceConnection: vs,
		Channel:         channel,
		Guild:           guild,
		PlayerStatus:    IS_NOT_PLAYING,
	}

}

func checkForDoubleVoiceConnection(guild string, channel string) {
	for index, voice := range voiceConnections {
		if voice.Guild == guild {
			voiceConnections = append(voiceConnections[:index], voiceConnections[index+1:]...)
		}
	}
}

func disconnectFromVoiceChannel(s *discordgo.Session, m *discordgo.MessageCreate) {
	channel, err := s.State.Channel(m.ChannelID)
	if err != nil {
		fmt.Println(err)
	}

	for index, voice := range voiceConnections {
		if voice.Guild == channel.GuildID {
			err := voice.VoiceConnection.Disconnect()
			if err != nil {
				log.Fatalln(err)
			}
			stopChannel <- true
			voiceConnections = append(voiceConnections[:index], voiceConnections[index+1:]...)
		}
	}
}

func playBruhSound(s *discordgo.Session, m *discordgo.MessageCreate) {
	channel, err := s.State.Channel(m.ChannelID)
	if err != nil {
		fmt.Println(err)
	}
	guild, err := s.State.Guild(channel.GuildID)
	if err != nil {
		fmt.Println(err)
	}
	voiceChannel := findVoiceChannelID(guild, m)
	go playAudioFile(bruhSoundPath, channel.GuildID, voiceChannel, "web")
}

func playStalMusic(s *discordgo.Session, m *discordgo.MessageCreate) {
	channel, err := s.State.Channel(m.ChannelID)
	if err != nil {
		fmt.Println(err)
	}
	guild, err := s.State.Guild(channel.GuildID)
	if err != nil {
		fmt.Println(err)
	}
	voiceChannel := findVoiceChannelID(guild, m)
	go playAudioFile(stalMusicPath, channel.GuildID, voiceChannel, "web")
}

func playAudioFile(file string, guild string, channel string, linkType string) {
	voiceConnection, index := findVoiceConnection(guild, channel)

	switch voiceConnection.PlayerStatus {
	case IS_NOT_PLAYING:
		voiceConnections[index].PlayerStatus = IS_PLAYING

		dgvoice.PlayAudioFile(voiceConnection.VoiceConnection, file, stopChannel)

		voiceConnections[index].PlayerStatus = IS_NOT_PLAYING
	case IS_PLAYING:
		addSong(Song{
			Link:    file,
			Type:    linkType,
			Guild:   guild,
			Channel: channel,
		})
	}
}

func findVoiceConnection(guild string, channel string) (Voice, int) {
	var voiceConnection Voice
	var index int
	for i, vc := range voiceConnections {
		if vc.Guild == guild {
			voiceConnection = vc
			index = i
		}
	}
	return voiceConnection, index

}

func addSong(song Song) {
	queue = append(queue, song)
}

func stopMusic(s *discordgo.Session, m *discordgo.MessageCreate) {
	stopChannel <- true
}

func playYoutubeLink(s *discordgo.Session, m *discordgo.MessageCreate) {

	if len(commandArgs) < 2 {
		s.ChannelMessageSend(m.ChannelID, "The [ .yt ] command needs argument: .yt <URL>")
		return
	}
	channel, err := s.State.Channel(m.ChannelID)
	if err != nil {
		fmt.Println(err)
	}
	guild, err := s.State.Guild(channel.GuildID)
	if err != nil {
		fmt.Println(err)
	}

	voiceChannel := findVoiceChannelID(guild, m)
	audioURL, err := getYoutubeAudioLink(commandArgs[1])
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "uWo sowwy but I couldn't extract audio track from this video")
	}

	go playAudioFile(audioURL, channel.GuildID, voiceChannel, "web")
}

func getYoutubeAudioLink(URL string) (string, error) {

	video, err := ytdl.GetVideoInfo(context.Background(), URL)
	if err != nil {
		log.Fatalln(err)
	}
	client := ytdl.Client{}

	for _, format := range video.Formats {
		if format.AudioEncoding == "opus" || format.AudioEncoding == "aac" || format.AudioEncoding == "vorbis" {
			data, err := client.GetDownloadURL(context.Background(), video, format)
			if err != nil {
				fmt.Println(err)
			}
			url := data.String()
			return url, nil
		}
	}
	return "", errors.New("Coudn't extract audio track from given video")
}

func playAudioLink(s *discordgo.Session, m *discordgo.MessageCreate) {
	channel, err := s.State.Channel(m.ChannelID)
	if err != nil {
		fmt.Println(err)
	}
	guild, err := s.State.Guild(channel.GuildID)
	if err != nil {
		fmt.Println(err)
	}
	voiceChannel := findVoiceChannelID(guild, m)

	s.ChannelMessageSend(m.ChannelID, commandArgs[1])

	go playAudioFile(commandArgs[1], channel.GuildID, voiceChannel, "web")
}

func playLibraryMusic(s *discordgo.Session, m *discordgo.MessageCreate) {
	var musicArr []string
	var musicNameArr []string
	var musicStrList string
	var c int = 0
	var musicPath string = "./audio"
	var itemsPerPage = 10

	err := filepath.Walk(musicPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if musicPath == path {
				return nil
			}
			musicArr = append(musicArr, path)
			musicNameArr = append(musicNameArr, info.Name())
			c++
			return nil
		})
	if err != nil {
		log.Println(err)
	}

	if len(commandArgs) < 2 {
		s.ChannelMessageSend(m.ChannelID, "oWu use some sub-command: list, play")
		return

	} else if commandArgs[1] == "list" {
		if len(commandArgs) != 3 {
			s.ChannelMessageSend(m.ChannelID, "OwU sowwy, but youw shouwd pwowide a pwage.")
			return
		} else {
			page, err := strconv.Atoi(commandArgs[2])
			if err != nil {
				log.Println(err)
			}
			if page < 1 {
				s.ChannelMessageSend(m.ChannelID, "Libraries list page should be > 0")
				return
			}
			for i := (page - 1) * itemsPerPage; i < (page)*itemsPerPage; i++ {
				if i+10 > len(musicArr) {
					for j := i; j < len(musicNameArr); j++ {
						musicStrList += strconv.Itoa(j) + ") " + musicNameArr[j] + "\n"
					}
					break
				}
				musicStrList += strconv.Itoa(i+1) + ") " + musicNameArr[i] + "\n"
			}
		}
		if musicStrList == "" {
			s.ChannelMessageSend(m.ChannelID, "OwU sowwy, but you page is too big for my small music library\n ( ͡° ͜ʖ ͡°).")
			return

		}
		embedExample = &discordgo.MessageEmbed{
			Author:      &discordgo.MessageEmbedAuthor{},
			Color:       0x000000,
			Description: musicStrList,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: "https://i.ytimg.com/vi/zI3EHVxS110/maxresdefault.jpg",
			},
			Title: "Music Library Page: [" + commandArgs[2] + " / " + strconv.Itoa(int(len(musicNameArr)/itemsPerPage+1)) + "]",
		}

		_, err = s.ChannelMessageSendEmbed(m.ChannelID, embedExample)
		if err != nil {
			log.Println(err)
		}

	} else if commandArgs[1] == "play" {
		if len(commandArgs) < 3 {
			s.ChannelMessageSend(m.ChannelID, "OwU sowwy, but you should provide music index.")
			return
		}
		musicIndex, err := strconv.Atoi(commandArgs[2])
		if err != nil {
			log.Println("Error parsing index")
			return
		}
		channel, err := s.State.Channel(m.ChannelID)
		if err != nil {
			fmt.Println(err)
		}
		guild, err := s.State.Guild(channel.GuildID)
		if err != nil {
			fmt.Println(err)
		}
		voiceChannel := findVoiceChannelID(guild, m)
		log.Println("./" + musicArr[musicIndex])
		playAudioFile("./"+musicArr[musicIndex], channel.GuildID, voiceChannel, "web")
	}

}

func nextSong(s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(queue) > 0 {
		s.ChannelMessageSend(m.ChannelID, "Skipped")
		go playAudioFile(queue[0].Link, queue[0].Guild, queue[0].Channel, queue[0].Type)
		queue = append(queue[:0], queue[1:]...)
	}
	s.ChannelMessageSend(m.ChannelID, "Nothing to skip!")

}

// TODO: Implement callable queue and sequential playing stuff
// TODO: Use folders for music listing

func flex(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, "Ayy LMAO dats a huge cringe u just posted bro")
}
