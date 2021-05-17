package publisher

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/dinup24/vax-notifier/common"
	log "github.com/sirupsen/logrus"
)

type TelegramBot struct {
	telegramToken string
}

func (t *TelegramBot) Init() {
	t.telegramToken = os.Getenv("TELEGRAM_TOKEN")
	if len(t.telegramToken) == 0 {
		panic("telegram token not passed")
	}
	log.Info("telegramToken initialized")
}

func (t *TelegramBot) PublishAvailableCenters(availableCenters []common.Center, channels []*common.Channel) error {
	client := &http.Client{}
	var str string

	for i := 0; i < len(availableCenters); i++ {
		str += "*" + strconv.Itoa(i+1) + ".* " + availableCenters[i].String() + "\n"

		str += "https://selfregistration.cowin.gov.in"

		str = strings.Replace(str, ".", "\\.", -1)
		str = strings.Replace(str, "-", "\\-", -1)
		str = strings.Replace(str, "(", "\\(", -1)
		str = strings.Replace(str, ")", "\\)", -1)
		msg := url.QueryEscape(str)

		for k := 0; k < len(channels); k++ {
			chatId := channels[k].ChatId
			url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=%s&parse_mode=MarkdownV2", t.telegramToken, chatId, msg)
			log.Info("url: ", url)

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				log.Fatalln(err)
			}
			res, err := client.Do(req)
			if err != nil {
				log.Fatalln(err)
			}

			log.Debug(res)

			body, err := io.ReadAll(res.Body)
			if err != nil {
				log.Fatalln(err)
			}

			log.Debug(string(body))
		}

		common.UpdateTrackerforPublished(availableCenters[i])
	}
	return nil
}

func (t *TelegramBot) Publish(msg interface{}, chatId string) error {
	msgStr := url.QueryEscape(msg.(common.Stats).String())

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=%s&parse_mode=MarkdownV2", t.telegramToken, chatId, msgStr)
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

	return nil
}
