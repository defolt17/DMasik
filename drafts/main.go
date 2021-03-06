package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
	"github.com/franela/goreq"
	"github.com/joho/godotenv"
	"github.com/rylio/ytdl"
)

type Configuration struct {
	Token           string `json:"token"`
	Prefix          string `json:"prefix"`
	SoundcloudToken string `json:"soundcloud_token"`
	YoutubeToken    string `json:"youtube_token"`
}

type SoundcloudResponse struct {
	Link  string `json:"stream_url"`
	Title string `json:"title"`
}

type YoutubeRoot struct {
	Items []YoutubeVideo `json:"items"`
}

type YoutubeVideo struct {
	Snippet YoutubeSnippet `json:"snippet"`
}

type YoutubeSnippet struct {
	Resource ResourceID `json:"resourceId"`
}

type ResourceID struct {
	VideoID string `json:"videoId"`
}


func discordMessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	
	} else if commandArgs[0] == ".youtube" {
		log.Println("YOUTUBE play")
		go playYoutubeLink(sanitizeLink(commandArgs[1]), channel.GuildID, voiceChannel)
	} else if commandArgs[0] == ".soundcloud" {
		log.Println("SOUNDCLOUD play")
		go playSoundcloudLink(sanitizeLink(commandArgs[1]), channel.GuildID, voiceChannel)
	} else if commandArgs[0] == ".playlist" {
		go playYoutubePlaylist(commandArgs[1], channel.GuildID, voiceChannel)

}


// This function is used to extract the id of a playlist given a youtube plaulist link
func parseYoutubePlaylistLink(link string) string {
	standardPlaylistSanitize := strings.Replace(link, "https://www.youtube.com/playlist?list=", "", 1)
	return standardPlaylistSanitize
}

// This function will crawl the voice connections and try to find and return a voice connection
// and its index if one is found
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

// This function will call the playAudioFile function in a new goroutine if songs are remaining in the
// queue. If there is no song left in the queue, the function return false
func nextSong() bool {
	if len(queue) > 0 {
		go playAudioFile(queue[0].Link, queue[0].Guild, queue[0].Channel, queue[0].Type)
		queue = append(queue[:0], queue[1:]...)
		return true
	} else {
		return false
	}
}

// This function is used to add an item to the queue
func addSong(song Song) {
	queue = append(queue, song)
}

// This function is used to play every audio files, if the program is already playing, the function
// will add the song to the queue and call the nextSonng function when the current song is over
func playAudioFile(file string, guild string, channel string, linkType string) {
	voiceConnection, index := findVoiceConnection(guild, channel)
	log.Println(guild, channel)
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

// This function allow the user to stop the current playing file
func stopAudioFile(guild string, channel string) {
	_, index := findVoiceConnection(guild, channel)
	voiceConnections[index].PlayerStatus = IS_NOT_PLAYING
	//dgvoice.KillPlayer()
}

// This function allow the bot to find the voice channel id of the user who called the connect command
func findVoiceChannelID(guild *discordgo.Guild, message *discordgo.MessageCreate) string {
	var channelID string

	for _, vs := range guild.VoiceStates {
		if vs.UserID == message.Author.ID {
			channelID = vs.ChannelID
		}
	}
	return channelID
}

// This function allow the user to connect the bot to a channel. It will ask the voice channel id
// of the user to the findVoiceChannelID function and will then call the ChannelVoiceJoin
// of the discordgo.Session instance. Then it checks if the voice connection already exist and
// return a new Voice object
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

// This function check if there is already an existing voice connection for the givent params
func checkForDoubleVoiceConnection(guild string, channel string) {
	for index, voice := range voiceConnections {
		if voice.Guild == guild {
			voiceConnections = append(voiceConnections[:index], voiceConnections[index+1:]...)
		}
	}
}

// This function is used to play the audio of a youtube video. It use the ytdl pacakge to get
// the video informations and then look for the good format. When a link is found, it calls the
// playAudioFile function in a new goroutine
func playYoutubeLink(link string, guild string, channel string) {
	video, err := ytdl.GetVideoInfoFromID(link)

	log.Println("Video: ", video)

	if err != nil {
		log.Fatalln(err)
	}

	for _, format := range video.Formats {
		log.Print(format)
		if format.AudioEncoding == "opus" || format.AudioEncoding == "aac" || format.AudioEncoding == "vorbis" {
			data, err := video.GetDownloadURL(format)
			if err != nil {
				fmt.Println(err)
			}
			url := data.String()
			log.Println(url)
			go playAudioFile(url, guild, channel, "youtube")
			return
		}
	}

}

// This function is used to play a soundcloud link. It make a call to the API and to get the stream url
// it then call the playAudioFile function in a new goroutine
func playSoundcloudLink(link string, guild string, channel string) {
	var scRequestUri string = "https://api.soundcloud.com/resolve?url=" + link + "&client_id=" + soundcloudToken
	res, err := goreq.Request{
		Uri:          scRequestUri,
		MaxRedirects: 2,
		Timeout:      5000 * time.Millisecond,
	}.Do()
	if err != nil {
		fmt.Println(err)
	}
	var soundcloudData SoundcloudResponse
	res.Body.FromJsonTo(&soundcloudData)
	soundcloudData.Link += "&client_id=" + soundcloudToken
	go playAudioFile(soundcloudData.Link, guild, channel, "soundcloud")
}

// This function is used to play a youtube playlist, it will make a call to the youtube API to get the
// link for every video in the playlist. When the items are found, it will iterate and call the playYoutubeLink
// function for every link, they will automatically be added to the queue
func playYoutubePlaylist(link string, guild string, channel string) {
	var youtubeRequestLink string = "https://www.googleapis.com/youtube/v3/playlistItems?part=snippet&maxResults=50&playlistId=" + link + "&key=" + youtubeToken
	log.Println(youtubeRequestLink)
	res, err := goreq.Request{
		Uri:          youtubeRequestLink,
		MaxRedirects: 2,
		Timeout:      5000 * time.Millisecond,
	}.Do()
	if err != nil {
		fmt.Println(err)
	}
	var youtubeData YoutubeRoot
	res.Body.FromJsonTo(&youtubeData)
	for _, youtubeVideo := range youtubeData.Items {
		var videoURL string = "https://www.youtube.com/watch?v=" + youtubeVideo.Snippet.Resource.VideoID
		go playYoutubeLink(videoURL, guild, channel)
	}

}

var (
	// commands = map[string]func(s *discordgo.Session, m *discordgo.MessageCreate){
	// 	"ping": ping,
	// 	"pong": pong,
	// 	"help": help,
	// 	"meme": meme,
	// }

	soundcloudToken  string
	youtubeToken     string
	voiceConnections []Voice
	queue            []Song
	stopChannel      chan bool
	embedExample     = &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{},
		Color:       0x00ff00, // Green
		Description: "This is a discordgo embed",
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "I am a field1",
				Value:  "I am a value2",
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "I am a field3",
				Value:  "I am a value4",
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "I am a field5",
				Value:  "I am a value6",
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
	imageNaniPath   = "./images/memes/Nani.png"
	imageMeURL      = "https://avatars3.githubusercontent.com/u/22434204?s=460&u=cc62b75ba8a868b3c0af3b2b0ef7df7830963a5b&v=4"
	sampleRate      = 44100
	seconds         = 1
	GuildID         = "123"
	ChannelID       = "123"
	bruhFilePath    = "/home/idf3da97df/nujdiki/bruh.opus"
	giornoFilePath  = "/home/idf3da97df/nujdiki/giorno.opus"
	nujdikiFilePath = "/home/idf3da97df/nujdiki/nujdiki.opus"
	fithaFilePath   = "/home/idf3da97df/nujdiki/fitha.opus"
	kiraFilePath    = "/home/idf3da97df/nujdiki/kira.opus"
	e0eFilePath     = "/home/idf3da97df/nujdiki/808.wav"
	e0eHardFilePath = "/home/idf3da97df/nujdiki/808-hard.wav"
	stalFilePath    = "/home/idf3da97df/nujdiki/stal.opus"
)
