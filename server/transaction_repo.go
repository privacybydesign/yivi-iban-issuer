package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type InMemoryTokenStorage struct {
	TokenMap map[TransactonId]MerchantReference
	mutex    sync.Mutex
}

func NewInMemoryTokenStorage() *InMemoryTokenStorage {
	return &InMemoryTokenStorage{
		TokenMap: make(map[TransactonId]MerchantReference),
	}
}

type RedisTokenStorage struct {
	client   *redis.Client
	username string
}

func NewRedisTokenStorage(client *redis.Client, username string) *RedisTokenStorage {
	return &RedisTokenStorage{client: client, username: username}
}

// Should be safe to use in concurreny
type TokenStorage interface {
	StoreToken(transactionId TransactonId, merchantReference MerchantReference) error
	RetrieveToken(transactionId TransactonId) (MerchantReference, error)
	RemoveToken(transactionId TransactonId) error
}

// ------------------------------------------------------------------------------

func createKey(username string, transactionId TransactonId) string {
	return fmt.Sprintf("%v:token:%v", username, transactionId)
}

const Timeout time.Duration = 24 * time.Hour

func (s *RedisTokenStorage) StoreToken(transactionId TransactonId, merchantReference MerchantReference) error {
	ctx := context.Background()
	return s.client.Set(ctx, createKey(s.username, transactionId), string(merchantReference), Timeout).Err()
}

func (s *RedisTokenStorage) RetrieveToken(transactionId TransactonId) (MerchantReference, error) {
	ctx := context.Background()
	result, err := s.client.Get(ctx, createKey(s.username, transactionId)).Result()
	fmt.Println("result", result)
	if err != nil {
		return "", err
	}

	return MerchantReference(result), nil
}

func (s *RedisTokenStorage) RemoveToken(transactionId TransactonId) error {
	ctx := context.Background()
	return s.client.Del(ctx, createKey(s.username, transactionId)).Err()
}

// ------------------------------------------------------------------------------

func (s *InMemoryTokenStorage) StoreToken(transactionId TransactonId, merchantReference MerchantReference) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.TokenMap[transactionId] = merchantReference
	return nil
}

func (s *InMemoryTokenStorage) RetrieveToken(transactionId TransactonId) (MerchantReference, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if token, ok := s.TokenMap[transactionId]; ok {
		return token, nil
	} else {
		return "", fmt.Errorf("failed to find token for %s", transactionId)
	}
}

func (s *InMemoryTokenStorage) RemoveToken(transactionId TransactonId) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, ok := s.TokenMap[transactionId]; ok {
		delete(s.TokenMap, transactionId)
		return nil
	} else {
		return fmt.Errorf("failed to remove token for %s, because it wasn't there", transactionId)
	}
}
