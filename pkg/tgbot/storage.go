package tgbot

import "context"

type HailAlertSubscriptioner interface {
	GetHailSubscriptions(ctx context.Context, chatID int64) ([]string, error)
	AddHailSubscription(ctx context.Context, chatID int64, region string) error
	DeleteSubscriptions(ctx context.Context, chatID int64) error
	GetChatsForRegion(ctx context.Context, region string) ([]int64, error)
}
