package repos

import (
	"strings"

	"github.com/minhtuancn/open-prompt/go-engine/db"
)

// SharedPrompt đại diện cho prompt được chia sẻ trong marketplace
type SharedPrompt struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Tags        string `json:"tags"`
	Downloads   int    `json:"downloads"`
	IsPublic    bool   `json:"is_public"`
	CreatedAt   string `json:"created_at"`
}

// MarketplaceRepo quản lý shared_prompts table
type MarketplaceRepo struct {
	db *db.DB
}

// NewMarketplaceRepo tạo repo mới
func NewMarketplaceRepo(db *db.DB) *MarketplaceRepo {
	return &MarketplaceRepo{db: db}
}

// List trả về danh sách public prompts, sắp xếp theo downloads
func (r *MarketplaceRepo) List(limit, offset int) ([]SharedPrompt, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(`
		SELECT id, user_id, title, content, COALESCE(description,''), COALESCE(category,''), COALESCE(tags,''), downloads, is_public, created_at
		FROM shared_prompts WHERE is_public = 1
		ORDER BY downloads DESC, created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SharedPrompt
	for rows.Next() {
		var p SharedPrompt
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.Description, &p.Category, &p.Tags, &p.Downloads, &p.IsPublic, &p.CreatedAt); err != nil {
			continue
		}
		result = append(result, p)
	}
	if result == nil {
		result = []SharedPrompt{}
	}
	return result, nil
}

// sanitizeLikePattern escape LIKE wildcards để tránh DoS/timing attacks
func sanitizeLikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	if len(s) > 100 {
		s = s[:100]
	}
	return s
}

// Search tìm prompts theo keyword
func (r *MarketplaceRepo) Search(query string, limit int) ([]SharedPrompt, error) {
	if limit <= 0 {
		limit = 50
	}
	like := "%" + sanitizeLikePattern(query) + "%"
	rows, err := r.db.Query(`
		SELECT id, user_id, title, content, COALESCE(description,''), COALESCE(category,''), COALESCE(tags,''), downloads, is_public, created_at
		FROM shared_prompts
		WHERE is_public = 1 AND (title LIKE ? ESCAPE '\' OR description LIKE ? ESCAPE '\' OR tags LIKE ? ESCAPE '\')
		ORDER BY downloads DESC
		LIMIT ?
	`, like, like, like, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SharedPrompt
	for rows.Next() {
		var p SharedPrompt
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.Description, &p.Category, &p.Tags, &p.Downloads, &p.IsPublic, &p.CreatedAt); err != nil {
			continue
		}
		result = append(result, p)
	}
	if result == nil {
		result = []SharedPrompt{}
	}
	return result, nil
}

// PublishInput chứa thông tin để publish prompt
type PublishInput struct {
	UserID      int64
	Title       string
	Content     string
	Description string
	Category    string
	Tags        string
}

// Publish thêm prompt vào marketplace
func (r *MarketplaceRepo) Publish(input PublishInput) (int64, error) {
	res, err := r.db.Exec(`
		INSERT INTO shared_prompts (user_id, title, content, description, category, tags)
		VALUES (?, ?, ?, ?, ?, ?)
	`, input.UserID, input.Title, input.Content, input.Description, input.Category, input.Tags)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Install copy shared prompt vào user's prompts table + tăng downloads
func (r *MarketplaceRepo) Install(sharedID, userID int64) error {
	// Copy vào prompts table
	_, err := r.db.Exec(`
		INSERT INTO prompts (user_id, title, content, category, tags)
		SELECT ?, title, content, category, tags FROM shared_prompts WHERE id = ?
	`, userID, sharedID)
	if err != nil {
		return err
	}
	// Tăng download count
	_, err = r.db.Exec(`UPDATE shared_prompts SET downloads = downloads + 1 WHERE id = ?`, sharedID)
	return err
}
