package tgbot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strings"
)

type Bot interface {
	Run(ctx context.Context, done func())
	TriggerEvent(ctx context.Context, ev Event)
}

func NewBot(
	ctx context.Context,
	token string,
	log Logger,
	hs HailAlertSubscriptioner,
) (Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	//bot.Debug = true
	log.InfoContext(ctx, "Authorized", "account", bot.Self.UserName)
	return &tgBot{
		bot:  bot,
		log:  log,
		hs:   hs,
		evCh: make(chan Event, 1000),
	}, nil
}

type tgBot struct {
	bot  *tgbotapi.BotAPI
	evCh chan Event
	log  Logger
	hs   HailAlertSubscriptioner
}

type Logger interface {
	InfoContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

func (t *tgBot) Run(ctx context.Context, done func()) {
	defer done()
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := t.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			t.bot.StopReceivingUpdates()
			t.log.InfoContext(ctx, "context canceled; stopping process")
			return
		case ev := <-t.evCh:
			t.log.InfoContext(ctx, "new event received", "evType", ev.Type())
			err := t.HandleEvent(ctx, ev)
			if err != nil {
				t.log.ErrorContext(ctx, "can not handle external event", err)
			}

		case update := <-updates:
			if update.Message != nil {
				t.log.InfoContext(ctx, "got message", "user", update.Message.From.UserName, "msg", update.Message.Text)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
				msg.ReplyToMessageID = update.Message.MessageID
				err := t.handleMessage(ctx, msg)
				if err != nil {
					t.log.ErrorContext(ctx, "can not handle update", "err", err)
				}
			}
			if update.CallbackQuery != nil {
				t.log.InfoContext(ctx, "got callback query", "user", update.CallbackQuery.From.UserName, "data", update.CallbackData())
				err := t.handleCallback(ctx, update.CallbackQuery)
				if err != nil {
					t.log.ErrorContext(ctx, "can not handle update", "err", err)
				}
			}
		}
	}
}

func (t *tgBot) HandleEvent(ctx context.Context, ev Event) error {
	switch ev.Type() {
	case EvTypeHailProbabilityChange:
		v, ok := ev.Desc().(eventHailProbabilityChange)
		if !ok {
			return errors.New("runtime error: interface converting")
		}
		t.log.InfoContext(ctx, "The probability of hail has changed", "region", v.region, "old", v.oldLevel, "new", v.newLevel)
		chatIds, err := t.hs.GetChatsForRegion(ctx, v.region)
		if err != nil {
			return err
		}
		msgText := fmt.Sprintf(
			"%s The probability of hail has changed from %d to %d in the %s region",
			getEmojiForHailProbability(v.newLevel), v.oldLevel, v.newLevel, v.region,
		)
		for _, chatID := range chatIds {
			_, err := t.bot.Send(tgbotapi.NewMessage(chatID, msgText))
			t.log.ErrorContext(ctx, "can not send message", "err", err)
		}
	}
	return nil
}

func (t *tgBot) handleMessage(ctx context.Context, msg tgbotapi.MessageConfig) error {
	words := strings.SplitN(msg.Text, " ", 2)
	var (
		cmd Cmd
		txt string
	)
	switch len(words) {
	case 0:
		_, err := t.bot.Send(tgbotapi.NewMessage(msg.ChatID, errTextEmptyMessage))
		return err
	case 1:
		cmd = Cmd(words[0])
	case 2:
		cmd = Cmd(words[0])
		txt = words[1]
	default:
		return errors.New("runtime error: unsupported behaviour")
	}
	err := t.handleCommand(ctx, msg.ChatID, cmd, txt)
	return err
}

func (t *tgBot) handleCallback(ctx context.Context, query *tgbotapi.CallbackQuery) error {
	var callbackData BtnCallbackData
	if query.Message == nil || query.Message.Chat == nil {
		return errors.New("unsupported callback (no message chat)")
	}
	msgID := query.Message.MessageID
	chatID := query.Message.Chat.ID
	err := json.Unmarshal([]byte(query.Data), &callbackData)
	if err != nil {
		return err
	} else {
		switch callbackData.Type {
		case btnTypeCommand:
			var data cmdCallbackData
			err = json.Unmarshal([]byte(query.Data), &data)
			if err != nil {
				return err
			}
			err = t.handleCommand(ctx, chatID, data.Command, data.Text)
			if err != nil {
				return err
			}
			_, _ = t.bot.Send(tgbotapi.NewDeleteMessage(chatID, msgID))
		}
	}
	return nil
}

func (t *tgBot) handleSubscribeCommand(ctx context.Context, chatID int64, txt string) error {
	var err error
	if txt == "" {
		msg := tgbotapi.NewMessage(chatID, "Select a region:")
		msg.ReplyMarkup = MakeSubscribeRegionsButtonsMarkup()
		_, err = t.bot.Send(msg)
	} else {
		err = t.hs.AddHailSubscription(ctx, chatID, txt)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("I will notify you about %s 👌", txt))
		if err != nil {
			msg.Text = errTextCantSubscribe
		}
		_, err = t.bot.Send(msg)
	}
	return err
}

func (t *tgBot) handleCommand(ctx context.Context, chatID int64, cmd Cmd, txt string) error {
	if !cmd.valid() {
		_, err := t.bot.Send(tgbotapi.NewMessage(chatID, errTextUnsupportedCmd))
		return err
	}
	var err error
	switch cmd {
	case CmdHelp:
		_, err = t.bot.Send(tgbotapi.NewMessage(chatID, makeHelpText()))
	case CmdSubscribe:
		err = t.handleSubscribeCommand(ctx, chatID, txt)
	case CmdUnsubscribeAll:
		e := t.hs.DeleteSubscriptions(ctx, chatID)
		if e != nil {
			_, err = t.bot.Send(tgbotapi.NewMessage(chatID, errTextCantDeleteSubscriptions))
		} else {
			_, err = t.bot.Send(tgbotapi.NewMessage(chatID, "Ok, you are now unsubscribed from all hail alerts 👌"))
		}
	case CmdListSubscriptions:
		regs, e := t.hs.GetHailSubscriptions(ctx, chatID)
		if e != nil {
			_, err = t.bot.Send(tgbotapi.NewMessage(chatID, errTextCantListSubscriptions))
		} else {
			ans := strings.Join(regs, "\n")
			if len(regs) == 0 {
				ans = "you have not active subscriptions"
			}
			_, err = t.bot.Send(tgbotapi.NewMessage(chatID, ans))
		}
	}
	return err
}

func (t *tgBot) TriggerEvent(ctx context.Context, ev Event) {
	t.evCh <- ev
}

const (
	errTextEmptyMessage            = "error: empty message"
	errTextUnsupportedCmd          = "error: unsupported command"
	errTextCantSubscribe           = "error: can not add subscription"
	errTextCantDeleteSubscriptions = "error: can not delete subscriptions"
	errTextCantListSubscriptions   = "error: can not list subscriptions"
)

func getEmojiForHailProbability(probability int) string {
	switch probability {
	case 0:
		return "🟢"
	case 1:
		return "🟡"
	case 2:
		return "🟠"
	case 3:
		return "🔴"
	}
	return ""
}
