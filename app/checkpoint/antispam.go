package checkpoint

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/NexonSU/telegram-go-chatbot/app/utils"
	m "github.com/keighl/metabolize"
	"github.com/valyala/fastjson"
	"gopkg.in/tucnak/telebot.v3"
	"gorm.io/gorm/clause"
)

type MetaData struct {
	Title       string `meta:"og:title"`
	Description string `meta:"og:description,description"`
}

var MaximumIdFromDB = GetMaximumIdFromDB()

func GetMaximumIdFromDB() int64 {
	var user telebot.User
	utils.DB.Last(&user).Limit(1)
	return user.ID
}

func CommandGetSpamChance(context telebot.Context) error {
	var user telebot.User
	var err error
	if len(context.Args()) == 0 && context.Message().ReplyTo == nil {
		user = *context.Sender()
	} else {
		user, _, err = utils.FindUserInMessage(context)
		if err != nil {
			return context.Reply(err.Error())
		}
	}
	spamchance := GetSpamChance(user)
	return context.Reply(fmt.Sprintf("%v спамер на %v процентов.", utils.UserFullName(&user), spamchance))
}

func AddToWhiteList(context telebot.Context) error {
	if context.Data() == "" {
		return context.Reply("Нужно указать URL или его часть.")
	}
	var link utils.AntiSpamLink
	link.URL = context.Data()
	link.Type = "whitelist"
	result := utils.DB.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&link)
	if result.Error != nil {
		return context.Reply(fmt.Sprintf("Ошибка запроса: <code>%v</code>", result.Error.Error()))
	}
	return context.Reply(fmt.Sprintf("URL <code>%v</code> добавлен в белый список.", link.URL))
}

func AddToBlackList(context telebot.Context) error {
	if context.Data() == "" {
		return context.Reply("Нужно указать URL или его часть.")
	}
	var link utils.AntiSpamLink
	link.URL = context.Data()
	link.Type = "blacklist"
	result := utils.DB.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&link)
	if result.Error != nil {
		return context.Reply(fmt.Sprintf("Ошибка запроса: <code>%v</code>", result.Error.Error()))
	}
	return context.Reply(fmt.Sprintf("URL <code>%v</code> добавлен в черный список.", link.URL))
}

func DelAntispamLink(context telebot.Context) error {
	if context.Data() == "" {
		return context.Reply("Нужно указать URL или его часть.")
	}
	var link utils.AntiSpamLink
	link.URL = context.Data()
	result := utils.DB.Delete(link)
	if result.Error != nil {
		return context.Reply(fmt.Sprintf("Ошибка запроса: <code>%v</code>", result.Error.Error()))
	}
	return context.Reply(fmt.Sprintf("URL <code>%v</code> удалён.", link.URL))
}

func ListAntispamLinks(context telebot.Context) error {
	var list = "Список URL фильтров:\n\n"
	result, err := utils.DB.Model(utils.AntiSpamLink{}).Rows()
	if err != nil {
		return err
	}
	for result.Next() {
		var link utils.AntiSpamLink
		err := utils.DB.ScanRows(result, &link)
		if err != nil {
			return err
		}
		list += fmt.Sprintf("%v - %v\n", link.URL, link.Type)
	}
	return context.Reply(list, &telebot.SendOptions{DisableWebPagePreview: true})
}

func GetSpamChance(user telebot.User) int {
	spamchance := 0
	//photos
	photos, _ := utils.Bot.ProfilePhotosOf(&user)
	photoCount := len(photos)
	if photoCount > 5 {
		photoCount = 5
	}
	spamchance -= photoCount*10 - 20
	//ID
	spamchance += int(float64(user.ID)/float64(MaximumIdFromDB)*100) - 50
	//Bio
	if user.Username != "" {
		res, _ := http.Get(fmt.Sprintf("https://t.me/%v", user.Username))
		data := new(MetaData)
		if m.Metabolize(res.Body, data) == nil {
			if len(data.Description) > 15 && data.Description[:15] == "You can contact" {
				spamchance += 10
			} else {
				spamchance -= 10
				if strings.Contains(data.Description, "http") {
					spamchance += 40
				}
			}
		}
	} else {
		spamchance += 10
	}
	return spamchance
}

func UrlFilter(context telebot.Context) error {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	httpResponse, _ := httpClient.Get(fmt.Sprintf("https://api.cas.chat/check?user_id=%v", context.Sender().ID))
	defer func(Body io.ReadCloser) {
		Body.Close()
	}(httpResponse.Body)
	jsonBytes, _ := ioutil.ReadAll(httpResponse.Body)
	if fastjson.GetBool(jsonBytes, "ok") {
		return context.Delete()
	}
	for _, entity := range context.Message().Entities {
		if entity.Type == "url" {
			var link utils.AntiSpamLink
			runes := []rune(context.Message().Text)
			url := string(runes[entity.Offset : entity.Offset+entity.Length])
			result := utils.DB.Where("instr(?, lower(url)) > 1", strings.ToLower(url)).First(&link)
			if result.Error != nil {
				return nil
			}
			if result.RowsAffected != 0 && GetSpamChance(*context.Sender()) > 50 && link.Type == "blacklist" {
				return context.Delete()
			}
		}
	}
	return nil
}