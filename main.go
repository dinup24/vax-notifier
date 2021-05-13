package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"

	log "github.com/sirupsen/logrus"
)

type City struct {
	Name       string     `yaml:"name"`
	DistrictId []int      `yaml:"districtId"`
	Channels   []*Channel `yaml:"channels"`
}

type Channel struct {
	MinAge      []int  `yaml:"minAge"`
	ChannelName string `yaml:"channelName"`
	ChatId      string `yaml:"chatId"`
}

type Stats struct {
	CheckingSince   string
	CheckCount      int
	LastPublishTime time.Time
}

func (s Stats) String() string {
	return "Checking since: " + s.CheckingSince + "\n" + "Check count: " + strconv.Itoa(s.CheckCount)
}

type Center struct {
	Center_id     int        `json:"center_id"`
	Name          string     `json:"name"`
	Address       string     `json:"address"`
	State_name    string     `json:"state_name"`
	District_name string     `json:"district_name"`
	Block_name    string     `json:"block_name"`
	Pincode       int        `json:"pincode"`
	Lat           int        `json:"lat"`
	Long          int        `json:"long"`
	From          string     `json:"from"`
	To            string     `json:"to"`
	Fee_type      string     `json:"fee_type"`
	Sessions      []*Session `json:"sessions"`
}

func (c Center) String() string {
	str := "*" + c.Name + "*, " + strconv.Itoa(c.Pincode) + "\n"

	for i := 0; i < len(c.Sessions); i++ {
		str += c.Sessions[i].String() + "\n"
	}

	return str
}

type Session struct {
	Session_id         string   `json:"session_id"`
	Date               string   `json:"date"`
	Available_capacity int      `json:"available_capacity"`
	Min_age_limit      int      `json:"min_age_limit"`
	Vaccine            string   `json:"vaccine"`
	Slots              []string `json:"slots"`
}

func (s Session) String() string {
	return formatDate(s.Date) + ": *" + strconv.Itoa(s.Available_capacity) + "* slots  " + s.Vaccine + " "
}

var telegramToken string
var telegramStatsGroup string
var stats Stats

func main() {

	log.SetOutput(os.Stdout)
	log.SetReportCaller(true)

	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true

	log.Info("Vax Notifier application has started sucessfully...")

	currentTime := time.Now()
	stats = Stats{}
	stats.CheckingSince = currentTime.Format("Jan 2, 2006 15:04:05")

	// Initialize
	telegramToken = os.Getenv("TELEGRAM_TOKEN")
	telegramStatsGroup = os.Getenv("TELEGRAM_STATS_GROUP")

	configFile := os.Getenv("CONFIG_FILE")
	cfg, _ := readConf(configFile)

	log.Info(cfg)

	for i := 0; i < len(cfg["cities"]); i++ {
		go func(city City) {
			for i := 0; i >= 0; i++ { // infinite loop
				log.Info("Check for City " + city.Name + " #" + strconv.Itoa(i))

				availableCenters := findAvailableSlots(city.DistrictId)

				if len(availableCenters) > 0 {
					publish(availableCenters, city.Channels)
					stats.LastPublishTime = time.Now()
				}

				stats.CheckCount += 1
				time.Sleep(60 * time.Second)
			}
		}(cfg["cities"][i])
	}

	for {
		sendUpdates()
		time.Sleep(300 * time.Second)
	}
}

func findAvailableSlots(districtIds []int) []Center {
	currentTime := time.Now()
	date := currentTime.Format("2-01-2006")

	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.93 Safari/537.36"
	client := &http.Client{}

	centers := []Center{}

	for k := 0; k < len(districtIds); k++ {
		url := "https://cdn-api.co-vin.in/api/v2/appointment/sessions/public/calendarByDistrict?district_id=" + strconv.Itoa(districtIds[k]) + "&date=" + date
		log.Info("url: ", url)

		req, err := http.NewRequest("GET", url, nil)
		req.Header.Set("user-agent", userAgent)
		if err != nil {
			log.Fatalln(err)
		}
		res, err := client.Do(req)
		if err != nil {
			log.Fatalln(err)
		}

		log.Info(res.Body)

		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatalln(err)
		}

		log.Info(string(body))

		//var data map[string]interface{}
		var data map[string][]Center
		err = json.Unmarshal(body, &data)
		if err != nil {
			log.Fatalln(err)
		}
		centers = append(centers, data["centers"]...)
	}

	var availableCenters []Center = []Center{}

	//centers := data["centers"]
	for i := 0; i < len(centers); i++ {
		var availableSessions []*Session = []*Session{}
		for j := 0; j < len(centers[i].Sessions); j++ {
			session := centers[i].Sessions[j]
			if session.Available_capacity == 0 && session.Min_age_limit == 18 {
				availableSessions = append(availableSessions, session)
			}
		}

		if len(availableSessions) > 0 {
			centers[i].Sessions = availableSessions
			availableCenters = append(availableCenters, centers[i])
		}
	}

	fmt.Println(availableCenters)
	return availableCenters
}

func publish(availableCenters []Center, channels []*Channel) {
	client := &http.Client{}

	for k := 0; k < len(channels); k++ {
		var str string
		for i := 0; i < len(availableCenters); i++ {
			str += strconv.Itoa(i+1) + ". " + availableCenters[i].String() + "\n"
		}
		str = strings.Replace(str, ".", "\\.", -1)
		str = strings.Replace(str, "-", "\\-", -1)
		msg := url.QueryEscape(str)

		chatId := channels[k].ChatId
		url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=%s&parse_mode=MarkdownV2", telegramToken, chatId, msg)
		log.Info("url: ", url)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatalln(err)
		}
		res, err := client.Do(req)
		if err != nil {
			log.Fatalln(err)
		}

		log.Info(res)

		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatalln(err)
		}

		log.Info(string(body))
	}
}

func sendUpdates() {
	msg := url.QueryEscape(stats.String())

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=%s&parse_mode=MarkdownV2", telegramToken, telegramStatsGroup, msg)
	log.Info("url: ", url)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalln(err)
	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	log.Info(res)

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalln(err)
	}

	log.Info(string(body))
}

func formatDate(date string) string {
	t, _ := time.Parse("2-01-2006", date)
	return t.Format("Jan 2, 2006")
}

func readConf(filename string) (map[string][]City, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg map[string][]City
	err = yaml.Unmarshal(buf, &cfg)
	if err != nil {
		return nil, fmt.Errorf("in file %q: %v", filename, err)
	}

	return cfg, nil
}
