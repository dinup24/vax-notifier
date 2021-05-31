package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/dinup24/vax-notifier/common"
	"github.com/dinup24/vax-notifier/publisher"
	log "github.com/sirupsen/logrus"
)

var telegramStatsGroup string
var publishInterval time.Duration

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
	common.St.CheckingSince = currentTime.Format("Jan 2, 2006 15:04:05")

	// router := mux.NewRouter()
	// router.HandleFunc("/sms", handleSms).Methods(http.MethodGet)
	// http.ListenAndServe(":9001", router)

	// time.Sleep(8000 * time.Second)

	// Initialize
	pubr := publisher.GetPublisher()
	pubr.Init()

	configFile := os.Getenv("CONFIG_FILE")
	if len(configFile) == 0 {
		panic("configFile name not passed")
	}
	log.Info("configFile: ", configFile)
	cfg, _ := common.ReadConf(configFile)
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

	publishIntervalStr := os.Getenv("PUBLISH_INTERVAL")
	if len(publishIntervalStr) == 0 {
		publishIntervalStr = "12h"
	}
	publishInterval, _ = time.ParseDuration(publishIntervalStr)
	log.Info("publishInterval: ", publishInterval)

	for i := 0; i < len(cfg["cities"]); i++ {
		go func(city common.City) {
			for i := 0; i >= 0; i++ { // infinite loop
				log.Info("Check for City " + city.Name + " #" + strconv.Itoa(i+1))

				availableCenters := findAvailableSlots(city.DistrictId)

				if len(availableCenters) > 0 {
					pubr.PublishAvailableCenters(availableCenters, city.Channels)
				}

				common.St.AddCheckCount()

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
			pubr.Publish(common.St, telegramStatsGroup)
		}

		//log.Info("Tracker: ", common.Tracker)

		time.Sleep(300 * time.Second)
	}
}

func findAvailableSlots(districtIds []int) []common.Center {
	defer common.RecoverFromPanic()

	currentTime := time.Now()
	date := currentTime.Format("2-01-2006")

	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.93 Safari/537.36"
	client := &http.Client{}

	centers := []common.Center{}

	for k := 0; k < len(districtIds); k++ {
		url := "https://cdn-api.co-vin.in/api/v2/appointment/sessions/calendarByDistrict?district_id=" + strconv.Itoa(districtIds[k]) + "&date=" + date
		log.Info("url: ", url)

		req, err := http.NewRequest("GET", url, nil)
		req.Header.Set("user-agent", userAgent)
		if err != nil {
			log.Fatalln(err)
		}
		res, err := client.Do(req)
		if err != nil {
			panic(err)
		}

		log.WithFields(
			log.Fields{
				"responseCode": res.Status,
			},
		).Info("Response for " + strconv.Itoa(districtIds[k]))

		if res.StatusCode < 400 {
			common.St.AddGoodResponse()
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
		} else {
			common.St.AddBadResponse()
		}
	}

	// Collect all available centers here...
	var availableCenters []common.Center = []common.Center{}

	for i := 0; i < len(centers); i++ {
		var availableSessions []*common.Session = []*common.Session{}
		//var qualifiedSessions []*common.Session = []*common.Session{}
		centerQualified := false

		for j := 0; j < len(centers[i].Sessions); j++ {
			session := centers[i].Sessions[j]

			//log.Info(centers[i].District_name, session.Available_capacity, session.Available_capacity_dose1, session.Available_capacity_dose2, session.Min_age_limit)

			//if session.Available_capacity == 0 && (session.Available_capacity_dose1 == 0 || session.Available_capacity_dose2 == 0) && session.Min_age_limit == 18 {
			if session.Available_capacity > 0 && (session.Available_capacity_dose1 > 0 || session.Available_capacity_dose2 > 0) && session.Min_age_limit == 18 {
				// Update the list of all available sessions
				availableSessions = append(availableSessions, session)

				// If the center is yet to be qualified, perform the checks
				if !centerQualified {
					ok := common.CheckSessionAgainstTracker(centers[i], session, publishInterval)

					// If a session becomes eligible, the center get qualified for publish
					if ok {
						centerQualified = true
					}
				}
				ok := common.UpdateTracker(centers[i], session, true)
				log.Info("available -> tracker updated: ", ok)
			} else {
				// Update the tracker with the latest session (if tracker has the object)
				_ = common.UpdateTracker(centers[i], session, false)
				//log.Info("unavailable -> tracker updated: ", ok)
			}
		}

		// If a center is qualified, all available sessions will be published
		if centerQualified {
			centers[i].Sessions = availableSessions
			availableCenters = append(availableCenters, centers[i])
		}
	}
	return availableCenters
}

func handleSms(w http.ResponseWriter, r *http.Request) {
	otpMsg := r.URL.Query().Get("msg")

	log.Info("otpMsg: ", otpMsg)

	regex, _ := regexp.Compile(".*(OTP).*([0-9]{6}).*([0-9]{1})\\s(minutes).*")

	sub := regex.FindStringSubmatch(otpMsg)

	if sub != nil && len(sub) > 0 {
		opt := sub[2]
		log.Info("opt: ", opt)

		common.Opt = opt
		common.OptTime = time.Now()
	}

	w.WriteHeader(http.StatusOK)
}
