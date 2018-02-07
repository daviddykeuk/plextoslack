package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

// event enumerations

const (
	play   = "media.play"
	pause  = "media.pause"
	resume = "media.resume"
	stop   = "media.stop"
	viewed = "media.scrobble"
	rated  = "media.rate"
)

// set up all the structs

type PlexMessage struct {
	Event    string `json:"event"`
	User     bool
	Owner    bool
	Account  PlexAccount
	Server   PlexServer
	Player   PlexPlayer
	Metadata PlexMetadata
}

type PlexAccount struct {
	Id    int32
	Thumb string
	Title string
}

type PlexServer struct {
	Title string
	Uuid  string
}

type PlexPlayer struct {
	Local         bool
	PublicAddress string
	Title         string
	Uuid          string
}

type PlexMetadata struct {
	LibrarySectionType   string
	LibrarySectionTitle  string
	RatingKey            string
	Key                  string
	ParentRatingKey      string
	GrandparentRatingKey string
	Guid                 string
	LibrarySectionID     int16
	MediaType            string `json:"type"`
	Title                string
	GrandparentKey       string
	GrandparentTitle     string
	ParentTitle          string
	Summary              string
	Index                int16
	ParentIndex          int16
	RatingCount          int16
	Thumb                string
	Art                  string
	ParentThumb          string
	GrandparentThumb     string
	GrandparentArt       string
	AddedAt              int32
	UpdatedAt            int32
}

type SlackMessage struct {
	Username    string            `json:"username"`
	Attachments []SlackAttachment `json:"attachments"`
	Send        string
}

type SlackAttachment struct {
	Fallback    string `json:"fallback"`
	Color       string `json:"color"`
	Pretext     string `json:"pretext"`
	Author_name string `json:"author_name"`
	Author_link string `json:"author_link"`
	Author_icon string `json:"author_icon"`
	Title       string `json:"title"`
	Title_link  string `json:"title_link"`
	Text        string `json:"text"`
	Image_url   string `json:"image_url"`
	Thumb_url   string `json:"thumb_url"`
	Footer      string `json:"footer"`
	Footer_icon string `json:"footer_icon"`
	Ts          int32  `json:"ts"`
}

// convert a Plex webhook to a Slack webhook
func createSlackMessage(message PlexMessage) SlackMessage {
	m := SlackMessage{}
	m.Username = "Plex"
	a := SlackAttachment{}

	m.Send = "false"

	title := message.Metadata.Title

	if message.Metadata.MediaType == "episode" {
		title = message.Metadata.Title + " from " + message.Metadata.ParentTitle + " of " + message.Metadata.GrandparentTitle
	}

	switch message.Event {
	case play:
		m.Send = os.Getenv("SEND_PLAY")
		a.Title = "Started watching " + title
		a.Color = "good"
	case pause:
		m.Send = os.Getenv("SEND_PAUSE")
		a.Title = "Paused " + title
		a.Color = "warning"
	case resume:
		m.Send = os.Getenv("SEND_RESUME")
		a.Title = "Continued watching " + title
		a.Color = "good"
	case stop:
		m.Send = os.Getenv("SEND_STOP")
		a.Title = "Stopped watching " + title
		a.Color = "danger"
	case viewed:
		m.Send = os.Getenv("SEND_VIEWED")
		a.Title = "Finished watching " + title
		a.Color = "danger"
	case rated:
		m.Send = os.Getenv("SEND_RATED")
		a.Title = "Rated " + title
		a.Color = "#439FE0"
	}

	a.Author_name = message.Account.Title
	a.Author_icon = message.Account.Thumb
	a.Text = message.Metadata.Summary
	a.Footer = "Played from " + message.Metadata.LibrarySectionTitle + " on " + message.Player.Title
	a.Fallback = "Fallback message"

	as := make([]SlackAttachment, 1)
	as[0] = a

	m.Attachments = as

	return m
}

// handle the inbound, multipat webhook
func handlePlexMessage(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(15485760)
	payload := r.MultipartForm.Value["payload"][0]

	log.Print(payload)

	var m PlexMessage
	err := json.Unmarshal([]byte(payload), &m)
	if err != nil {
		log.Panic("Covert error: ", err)
	}

	s := createSlackMessage(m)

	url := os.Getenv("SLACK_URL")
	fmt.Println("URL:>", url)

	json, err := json.Marshal(s)
	log.Println(string(json))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(json))
	req.Header.Set("Content-Type", "application/json")

	if s.Send == "true" {
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		fmt.Println("response Status:", resp.Status)
		fmt.Println("response Headers:", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("response Body:", string(body))
	}
}

// run for it Marty!
func main() {
	e := godotenv.Load()
	if e != nil {
		log.Print("Error loading .env file")
	}

	http.HandleFunc("/", handlePlexMessage)

	err := http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
