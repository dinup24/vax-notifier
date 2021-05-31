package common

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"
)

var Opt string
var OptTime time.Time

var St *Stats = &Stats{}

type City struct {
	Name            string        `yaml:"name"`
	DistrictId      []int         `yaml:"districtId"`
	PollingInterval time.Duration `yaml:"pollingInterval"`
	Channels        []*Channel    `yaml:"channels"`
}

type Channel struct {
	MinAge      []int  `yaml:"minAge"`
	ChannelName string `yaml:"channelName"`
	ChatId      string `yaml:"chatId"`
}

type Stats struct {
	CheckingSince   string
	CheckCount      int
	GoodApiResponse int
	BadApiResponse  int
	PanicCount      int
	sync.Mutex
}

func (s *Stats) AddGoodResponse() {
	s.Lock()
	defer s.Unlock()
	s.GoodApiResponse++
}

func (s *Stats) AddBadResponse() {
	s.Lock()
	defer s.Unlock()
	s.BadApiResponse++
}

func (s *Stats) AddCheckCount() {
	s.Lock()
	defer s.Unlock()
	s.CheckCount++
}

func (s *Stats) AddPanicCount() {
	s.Lock()
	defer s.Unlock()
	s.PanicCount++
}

func (s *Stats) String() string {
	return "Checking since: " + s.CheckingSince + "\n" + "Check count: " + strconv.Itoa(s.CheckCount) + "\n" + "Good api response: " + strconv.Itoa(s.GoodApiResponse) + "\n" + "Bad api response: " + strconv.Itoa(s.BadApiResponse) + "\n" + "Panic count: " + strconv.Itoa(s.PanicCount)
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
	Session_id               string   `json:"session_id"`
	Date                     string   `json:"date"`
	Available_capacity       int      `json:"available_capacity"`
	Available_capacity_dose1 int      `json:"available_capacity_dose1"`
	Available_capacity_dose2 int      `json:"available_capacity_dose2"`
	Min_age_limit            int      `json:"min_age_limit"`
	Vaccine                  string   `json:"vaccine"`
	Slots                    []string `json:"slots"`
}

func (s Session) String() string {
	return formatDate(s.Date) + ": *" + strconv.Itoa(s.Available_capacity) + "* slots " + s.Vaccine + " (Dose 1: " + strconv.Itoa(s.Available_capacity_dose1) + ", Dose 2: " + strconv.Itoa(s.Available_capacity_dose2) + ")"
}

var Tracker map[string]*TrackerData = map[string]*TrackerData{}

type TrackerData struct {
	Session         *Session
	LastCheckTime   time.Time
	LastPublishTime time.Time
}

func (td TrackerData) String() string {
	return strconv.Itoa(td.Session.Available_capacity) + "-" + td.LastCheckTime.String() + "-" + td.LastPublishTime.String()
}

func formatDate(date string) string {
	t, _ := time.Parse("2-01-2006", date)
	return t.Format("Jan 2, 2006")
}

func GetTrackerKey(center Center, session *Session) string {
	return strconv.Itoa(center.Center_id) + ":" + strconv.Itoa(center.Pincode) + ":" + session.Date + ":" + session.Vaccine + ":" + strconv.Itoa(session.Min_age_limit)
}

func UpdateTracker(center Center, session *Session, forceUpdate bool) bool {
	trackerKey := GetTrackerKey(center, session)
	currentTime := time.Now()

	// Update tracker with latest session, only if the tracker is tracking the session
	// Keeping the session upto date will help us publish more accurate data
	td := Tracker[trackerKey]

	// session is tracked
	if td != nil {
		td.Session = session
		td.LastCheckTime = currentTime
		return true
	}

	// session is not tracked, but update
	if forceUpdate {
		Tracker[trackerKey] = &TrackerData{
			Session:       session,
			LastCheckTime: currentTime,
		}
		return true
	}
	return false
}

// To be invoked only for available sessions
func CheckSessionAgainstTracker(center Center, session *Session, publishInterval time.Duration) bool {
	currentTime := time.Now()
	trackerKey := GetTrackerKey(center, session)

	td := Tracker[trackerKey]

	// Session not found in tracker -> first publish
	if td == nil {
		// td = &TrackerData{
		// 	Session:         *session,
		// 	LastCheckTime:   currentTime,
		// 	LastPublishTime: currentTime,
		// }

		//Tracker[trackerKey] = td

		log.WithFields(
			log.Fields{
				"trackerKey":  trackerKey,
				"capacity":    session.Available_capacity,
				"currentTime": currentTime,
			},
		).Info("Session was not found in tracker -> Qualified Session")

		return true
	}

	/*
	 * Session found in tracker, but available capacity is higher -> more vaccine doses added; publish
	 * Session found in tracker, but published long ago -> publish again
	 */
	if (td.Session.Available_capacity < session.Available_capacity) || (currentTime.Sub(td.LastPublishTime) > publishInterval) {
		//td.Session = *session
		//td.LastCheckTime = currentTime
		//td.LastPublishTime = currentTime

		log.WithFields(
			log.Fields{
				"trackerKey":      trackerKey,
				"newCapacity":     session.Available_capacity,
				"oldCapacity":     td.Session.Available_capacity,
				"lastPublishTime": td.LastPublishTime,
				"currentTime":     currentTime,
			},
		).Info("Session found; but higher capacity found or published long ago -> Qualified Session")

		return true
	}

	log.WithFields(
		log.Fields{
			"trackerKey":      trackerKey,
			"newCapacity":     session.Available_capacity,
			"oldCapacity":     td.Session.Available_capacity,
			"lastPublishTime": td.LastPublishTime,
			"currentTime":     currentTime,
		},
	).Info("Session found; equal/lower capcity and published recently  -> Non Qualified Session")

	//td.Session = *session
	//td.LastCheckTime = currentTime

	return false
}

func ReadConf(filename string) (map[string][]City, error) {
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

func UpdateTrackerforPublished(publishedCenter Center) {
	currentTime := time.Now()

	for i := 0; i < len(publishedCenter.Sessions); i++ {
		trackerKey := GetTrackerKey(publishedCenter, publishedCenter.Sessions[i])

		td, _ := Tracker[trackerKey]

		td.LastPublishTime = currentTime
	}
}

func GetToken() string {
	return ""
}

func RecoverFromPanic() {
	if r := recover(); r != nil {
		log.Info("Recovering from panic ", r)
		St.AddPanicCount()
	}
}
