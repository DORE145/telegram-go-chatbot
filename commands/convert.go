package commands

import (
	"bytes"
	"os"
	"strings"

	"github.com/NexonSU/telegram-go-chatbot/utils"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	tele "gopkg.in/telebot.v3"
)

// Convert given file
func Convert(context tele.Context) error {
	if context.Message().ReplyTo == nil {
		return context.Reply("Пример использования: <code>/convert</code> в ответ на какое-либо сообщение с медиа-файлом.\nДопольнительные параметры: gif,mp3,ogg,jpg.")
	}
	if context.Message().ReplyTo.Media() == nil {
		return context.Reply("Какого-либо медиа файла нет в указанном сообщении.")
	}
	var err error
	var fileName string
	var mime string
	var KwArgs map[string]interface{}
	media := context.Message().ReplyTo.Media()
	KwArgs = ffmpeg.KwArgs{"loglevel": "debug", "map": "0", "format": "mp4", "c:v": "libx264", "preset": "fast", "crf": 26, "movflags": "frag_keyframe+empty_moov+faststart", "c:a": "aac"}
	mime = "video/mp4"
	fileName = media.MediaFile().FileID + ".mp4"
	arg := media.MediaType()
	if len(context.Args()) == 1 {
		arg = strings.ToLower(context.Args()[0])
		switch arg {
		case "mp3", "audio", "ogg", "voice":
			if !utils.StringInSlice(media.MediaType(), []string{"video", "voice", "audio", "document"}) {
				return context.Reply("Неподдерживаемая операция")
			}
		case "jpg", "photo":
			if !utils.StringInSlice(media.MediaType(), []string{"photo", "animation", "video", "document"}) {
				return context.Reply("Неподдерживаемая операция")
			}
		case "gif", "animation":
			if !utils.StringInSlice(media.MediaType(), []string{"video", "animation", "document"}) {
				return context.Reply("Неподдерживаемая операция")
			}
		case "video", "video_note", "document":
			break
		default:
			return context.Reply("Неподдерживаемая операция")
		}
	}
	if arg == "sticker" && (context.Message().ReplyTo.Sticker.Animated || context.Message().ReplyTo.Sticker.Video) {
		arg = "gif"
	}
	switch arg {
	case "mp3", "audio":
		KwArgs = ffmpeg.KwArgs{"map": "a:0", "format": "mp3", "c:a": "libmp3lame"}
		mime = "audio/mp3"
		fileName = media.MediaFile().FileID + ".mp3"
	case "ogg", "voice":
		KwArgs = ffmpeg.KwArgs{"map": "a:0", "format": "ogg", "c:a": "libopus"}
		mime = "audio/ogg"
		fileName = media.MediaFile().FileID + ".ogg"
	case "jpg", "photo", "sticker":
		KwArgs = ffmpeg.KwArgs{"vf": "select=eq(n\\,0)", "format": "image2"}
		mime = "image/jpeg"
		fileName = media.MediaFile().FileID + ".jpg"
	case "gif", "animation":
		KwArgs = ffmpeg.KwArgs{"map": "v:0", "format": "mp4", "c:v": "libx264", "an": "", "preset": "fast", "crf": 26, "movflags": "frag_keyframe+empty_moov+faststart"}
	}
	buf := bytes.NewBuffer(nil)
	fileReader, err := utils.Bot.File(media.MediaFile())
	if err != nil {
		context.Reply("Ошибка загрузки файла")
		return err
	}
	err = ffmpeg.Input("pipe:").Output("pipe:", KwArgs).WithInput(fileReader).WithOutput(buf, os.Stdout).Run()
	if err != nil {
		context.Reply("Ошибка конвертации")
		return err
	}
	return context.Reply(&tele.Document{
		File:     tele.FromReader(buf),
		MIME:     mime,
		FileName: fileName,
	}, &tele.SendOptions{AllowWithoutReply: true})
}
