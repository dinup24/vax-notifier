package publisher

import (
	"github.com/dinup24/vax-notifier/common"
)

type Publisher interface {
	Init()
	PublishAvailableCenters(availableCenters []common.Center, channels []*common.Channel) error
	Publish(interface{}, string) error
}

var pubr Publisher = nil

func GetPublisher() Publisher {
	if pubr == nil {
		pubr = &TelegramBot{}
	}
	return pubr
}
