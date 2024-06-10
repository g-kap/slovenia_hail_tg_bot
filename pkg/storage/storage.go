package storage

import (
	"context"
	"database/sql"
)

type SqliteStorage struct {
	db *sql.DB
}

func (s *SqliteStorage) GetHailSubscriptions(ctx context.Context, chatID int64) ([]string, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT region FROM subscriptions WHERE chatID = $1`,
		chatID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var region string
		err := rows.Scan(&region)
		if err != nil {
			return nil, err
		}
		result = append(result, region)
	}
	return result, rows.Err()
}

func (s *SqliteStorage) AddHailSubscription(ctx context.Context, chatID int64, region string) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO subscriptions(chatID, region) VALUES ($1, $2)`,
		chatID, region,
	)
	return err
}

func (s *SqliteStorage) DeleteSubscriptions(ctx context.Context, chatID int64) error {
	_, err := s.db.ExecContext(
		ctx,
		`DELETE FROM subscriptions WHERE chatID = $1`,
		chatID,
	)
	return err
}

func (s *SqliteStorage) GetChatsForRegion(ctx context.Context, region string) ([]int64, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT chatID FROM subscriptions WHERE region = $1`,
		region,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []int64
	for rows.Next() {
		var chatID int64
		err := rows.Scan(&region)
		if err != nil {
			return nil, err
		}
		result = append(result, chatID)
	}
	return result, rows.Err()
}

func NewSqlStorage(db *sql.DB) *SqliteStorage {
	return &SqliteStorage{db: db}
}

func (s *SqliteStorage) MaybeInit(ctx context.Context) error {
	_, err := s.db.ExecContext(
		ctx,
		`CREATE TABLE  IF NOT EXISTS subscriptions(id INTEGER PRIMARY KEY AUTOINCREMENT, chatID INTEGER, region TEXT)`,
	)
	return err
}
