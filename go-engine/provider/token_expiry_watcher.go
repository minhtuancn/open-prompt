package provider

import (
	"log"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

// TokenExpiryWatcher kiểm tra tokens sắp hết hạn và trigger refresh
type TokenExpiryWatcher struct {
	tokenRepo *repos.ProviderTokenRepo
	onExpiry  func(token repos.ProviderToken)
	interval  time.Duration
	stopCh    chan struct{}
}

// NewTokenExpiryWatcher tạo watcher mới
// onExpiry callback được gọi khi token sắp hết hạn (< 10 phút)
func NewTokenExpiryWatcher(tokenRepo *repos.ProviderTokenRepo, onExpiry func(repos.ProviderToken), interval time.Duration) *TokenExpiryWatcher {
	if interval == 0 {
		interval = 2 * time.Minute
	}
	return &TokenExpiryWatcher{
		tokenRepo: tokenRepo,
		onExpiry:  onExpiry,
		interval:  interval,
		stopCh:    make(chan struct{}),
	}
}

// Start bắt đầu kiểm tra định kỳ
func (w *TokenExpiryWatcher) Start() {
	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				w.checkExpiry()
			case <-w.stopCh:
				return
			}
		}
	}()
}

// Stop dừng watcher
func (w *TokenExpiryWatcher) Stop() {
	close(w.stopCh)
}

// checkExpiry kiểm tra tất cả tokens active
func (w *TokenExpiryWatcher) checkExpiry() {
	// Kiểm tra cho tất cả users (0 = system, 1 = first user)
	for _, userID := range []int64{0, 1} {
		tokens, err := w.tokenRepo.GetByUser(userID)
		if err != nil {
			continue
		}
		now := time.Now()
		threshold := now.Add(10 * time.Minute) // Cảnh báo trước 10 phút

		for _, tok := range tokens {
			if tok.ExpiresAt != nil && tok.ExpiresAt.Before(threshold) {
				log.Printf("[token-watcher] token %s/%s sắp hết hạn (expires: %v)", tok.ProviderID, tok.AuthType, tok.ExpiresAt)
				if w.onExpiry != nil {
					w.onExpiry(tok)
				}
			}
		}
	}
}
