package common

import (
	"strconv"
	"time"
)

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
	CheckingSince string
	CheckCount    int
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
	return formatDate(s.Date) + ": *" + strconv.Itoa(s.Available_capacity) + "* slots  " + s.Vaccine + " (Dose 1: " + strconv.Itoa(s.Available_capacity_dose1) + ", Dose 2: " + strconv.Itoa(s.Available_capacity_dose2) + ")"
}

var Tracker map[string]time.Time = map[string]time.Time{}

func formatDate(date string) string {
	t, _ := time.Parse("2-01-2006", date)
	return t.Format("Jan 2, 2006")
}
