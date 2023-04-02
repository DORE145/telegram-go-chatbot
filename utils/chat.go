package utils

import (
	cntx "context"
	"strings"
	"time"

	gogpt "github.com/sashabaranov/go-gpt3"
	tele "gopkg.in/telebot.v3"
)

type botcntx struct {
	ID       int
	Messages []gogpt.ChatCompletionMessage
}

var c = gogpt.NewClient(Config.OpenAIKey)
var ctx = cntx.Background()
var botContexts []botcntx

func ChatGPT(context tele.Context) error {
	if context.Message().Text == "" {
		return nil
	}

	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return context.Reply("Локация Москва не найдена. В какой мы рельности Морти?")
	}
	currentTime := time.Now().In(location)
	if currentTime.Hour() > 7 && !IsAdminOrModer(context.Message().Sender.ID) {
		return nil
	}

	var messages []gogpt.ChatCompletionMessage

	if strings.HasPrefix(context.Message().Text, "/ask ") {
		if context.Message().ReplyTo != nil {
			if context.Message().ReplyTo.Sender.ID == Bot.Me.ID {
				messages = append(messages, gogpt.ChatCompletionMessage{Role: "assistant", Content: context.Message().ReplyTo.Text})
			} else {
				messages = append(messages, gogpt.ChatCompletionMessage{Role: "user", Content: context.Message().ReplyTo.Text})
			}
		}
	} else {
		if strings.HasPrefix(context.Message().Text, "/") {
			return nil
		}
		if context.Message().ReplyTo != nil && context.Message().ReplyTo.Sender.ID == Bot.Me.ID {
			for i := range botContexts {
				if botContexts[i].ID == context.Message().ReplyTo.ID {
					messages = botContexts[i].Messages
				}
			}

			if len(messages) == 0 {
				return nil
			}
		} else {
			return nil
		}
	}

	messages = append(messages, gogpt.ChatCompletionMessage{Role: "user", Content: strings.Replace(context.Message().Text, "/ask ", "", 1)})

	req := gogpt.ChatCompletionRequest{
		Model:    gogpt.GPT3Dot5Turbo,
		Messages: append([]gogpt.ChatCompletionMessage{{Role: "system", Content: "ты отвечаешь всегда максимально кратко, одним предложением"}}, messages...),
	}

	resp, err := c.CreateChatCompletion(ctx, req)
	if err != nil {
		return err
	}

	messages = append(messages, gogpt.ChatCompletionMessage{Role: "assistant", Content: resp.Choices[0].Message.Content})

	newMessageID, err := Bot.Reply(context.Message(), resp.Choices[0].Message.Content)
	if err != nil {
		return err
	}

	botContexts = append(botContexts, botcntx{ID: newMessageID.ID, Messages: messages})

	return nil
}