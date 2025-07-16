package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

type Postgres struct {
	DB *sql.DB
}

func NewPostgres(url string) (*Postgres, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &Postgres{DB: db}, nil
}

func (p *Postgres) Close() error {
	return p.DB.Close()
}

func (p *Postgres) InsertMessage(tenantID string, payload []byte) error {
	_, err := p.DB.Exec(`
		INSERT INTO messages (id, tenant_id, payload, created_at)
		VALUES (gen_random_uuid(), $1, $2, NOW())
	`, tenantID, payload)
	return err
}

func (p *Postgres) GetMessages(cursor string, limit int) ([]map[string]interface{}, string, error) {
	rows, err := p.DB.Query(`
		SELECT id, tenant_id, payload, created_at
		FROM messages
		WHERE id > $1
		ORDER BY id ASC
		LIMIT $2
	`, cursor, limit)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var results []map[string]interface{}
	var lastCursor string
	for rows.Next() {
		var id, tenantID, payload string
		var createdAt string
		if err := rows.Scan(&id, &tenantID, &payload, &createdAt); err != nil {
			return nil, "", err
		}
		results = append(results, map[string]interface{}{
			"id":         id,
			"tenant_id":  tenantID,
			"payload":    payload,
			"created_at": createdAt,
		})
		lastCursor = id
	}

	return results, lastCursor, nil
}

func (p *Postgres) EnsurePartitionForTenant(tenantID string) error {
	tableName := fmt.Sprintf("messages_tenant_%s", strings.ReplaceAll(tenantID, "-", "_"))

	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s PARTITION OF messages
		FOR VALUES IN ('%s')
	`, tableName, tenantID)

	_, err := p.DB.Exec(query)
	return err
}
