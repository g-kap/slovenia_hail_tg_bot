package tgbot

import (
	"encoding/json"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"tocadanes/pkg/regions"
)

func MakeSubscribeRegionsButtonsMarkup() tgbotapi.InlineKeyboardMarkup {
	var keys [][]tgbotapi.InlineKeyboardButton
	for _, r := range regions.SupportedRegions() {
		keys = append(keys, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(r,
				newCommandCallbackData(CmdSubscribe, r).String(),
			),
		))
	}
	return tgbotapi.NewInlineKeyboardMarkup(keys...)
}

type BtnType int

const (
	btnTypeCommand BtnType = 0
)

type BtnCallbackData struct {
	Type BtnType `json:"t"`
}

type cmdCallbackData struct {
	BtnCallbackData
	Command Cmd    `json:"c"`
	Text    string `json:"txt"`
}

func (b cmdCallbackData) String() string {
	js, _ := json.Marshal(b)
	return string(js)
}

func newCommandCallbackData(cmd Cmd, txt string) cmdCallbackData {
	return cmdCallbackData{
		BtnCallbackData: BtnCallbackData{
			Type: btnTypeCommand,
		},
		Command: cmd,
		Text:    txt,
	}
}
