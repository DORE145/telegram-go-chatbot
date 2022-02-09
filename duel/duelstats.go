package duel

import (
	"fmt"

	"github.com/NexonSU/telegram-go-chatbot/utils"
	"gopkg.in/telebot.v3"
)

//Send user utils.Duelist stats on /duelstats
func Duelstats(context telebot.Context) error {
	var duelist utils.Duelist
	result := utils.DB.Model(utils.Duelist{}).Where(context.Sender().ID).First(&duelist)
	if result.RowsAffected == 0 {
		return context.Reply("У тебя нет статистики.")
	}
	return context.Reply(fmt.Sprintf("Побед: %v\nСмертей: %v", duelist.Kills, duelist.Deaths))
}