package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	yaml "gopkg.in/yaml.v3"

	"github.com/dinup24/vax-notifier/common"
	"github.com/dinup24/vax-notifier/publisher"
	log "github.com/sirupsen/logrus"
)

var telegramToken string
var telegramStatsGroup string
var stats common.Stats

func main() {

	log.SetOutput(os.Stdout)
	log.SetReportCaller(true)
	log.SetLevel(log.InfoLevel)

	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true

	log.Info("Vax Notifier application has started sucessfully...")

	currentTime := time.Now()
	stats = common.Stats{}
	stats.CheckingSince = currentTime.Format("Jan 2, 2006 15:04:05")

	// Initialize
	pubr := publisher.GetPublisher()
	pubr.Init()

	configFile := os.Getenv("CONFIG_FILE")
	if len(configFile) == 0 {
		panic("configFile name not passed")
	}
	log.Info("configFile: ", configFile)
	cfg, _ := readConf(configFile)
	log.Info(cfg)

	telegramStatsGroup = os.Getenv("STATS_TELEGRAM_GROUP")
	if len(telegramStatsGroup) == 0 {
		log.Warn("telegramStatsGroup name not passed")
	}

	pollingIntervalStr := os.Getenv("POLLING_INTERVAL")
	if len(pollingIntervalStr) == 0 {
		pollingIntervalStr = "60s"
	}
	pollingInterval, _ := time.ParseDuration(pollingIntervalStr)
	log.Info("pollingInterval: ", pollingInterval)

	for i := 0; i < len(cfg["cities"]); i++ {
		go func(city common.City) {
			for i := 0; i >= 0; i++ { // infinite loop
				log.Info("Check for City " + city.Name + " #" + strconv.Itoa(i+1))

				availableCenters := findAvailableSlots(city.DistrictId)

				if len(availableCenters) > 0 {
					pubr.PublishAvailableCenters(availableCenters, city.Channels)
				}

				stats.CheckCount += 1

				if city.PollingInterval > 0 {
					time.Sleep(city.PollingInterval)
				} else {
					time.Sleep(pollingInterval)
				}
			}
		}(cfg["cities"][i])
	}

	for {
		if len(telegramStatsGroup) > 0 {
			pubr.Publish(stats, telegramStatsGroup)
		}

		log.Info("Tracker", common.Tracker)

		time.Sleep(300 * time.Second)
	}
}

func findAvailableSlots(districtIds []int) []common.Center {
	currentTime := time.Now()
	date := currentTime.Format("2-01-2006")

	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.93 Safari/537.36"
	client := &http.Client{}

	centers := []common.Center{}

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

		log.Debug(res.Body)

		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatalln(err)
		}

		log.Debug(string(body))

		var data map[string][]common.Center
		err = json.Unmarshal(body, &data)
		if err != nil {
			log.Fatalln(err)
		}
		centers = append(centers, data["centers"]...)
	}

	// Collect all available centers here...
	var availableCenters []common.Center = []common.Center{}

	for i := 0; i < len(centers); i++ {
		var availableSessions []*common.Session = []*common.Session{}
		for j := 0; j < len(centers[i].Sessions); j++ {
			session := centers[i].Sessions[j]
			if session.Available_capacity > 0 && session.Min_age_limit == 18 {
				availableSessions = append(availableSessions, session)
			}
		}

		if len(availableSessions) > 0 {
			centers[i].Sessions = availableSessions
			availableCenters = append(availableCenters, centers[i])
		}
	}
	return availableCenters
}

func readConf(filename string) (map[string][]common.City, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg map[string][]common.City
	err = yaml.Unmarshal(buf, &cfg)
	if err != nil {
		return nil, fmt.Errorf("in file %q: %v", filename, err)
	}

	return cfg, nil
}
