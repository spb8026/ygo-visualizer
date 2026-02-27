package carddb

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type CardData struct {
	Code       uint32
	Alias      uint32
	Type       uint32
	Level      uint32
	Attribute  uint32
	Race       uint64
	Attack     int32
	Defense    int32
	Lscale     uint32
	Rscale     uint32
	LinkMarker uint32
}

type DB struct {
	db *sql.DB
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open card db: %w", err)
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() {
	d.db.Close()
}

func (d *DB) GetCard(code uint32) (*CardData, error) {
	row := d.db.QueryRow(`
		SELECT id, alias, type, level,
		       attribute, race,
		       atk, def
		FROM datas
		WHERE id = ?`, code)

	var c CardData
	var rawLevel uint32

	err := row.Scan(
		&c.Code,
		&c.Alias,
		&c.Type,
		&rawLevel,
		&c.Attribute,
		&c.Race,
		&c.Attack,
		&c.Defense,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query failed for code %d: %w", code, err)
	}

	// =========================
	// Decode Level & Pendulum
	// =========================

	// bits 0–7   → monster level
	// bits 16–23 → right scale
	// bits 24–31 → left scale
	c.Lscale = (rawLevel >> 24) & 0xFF
	c.Rscale = (rawLevel >> 16) & 0xFF
	c.Level = rawLevel & 0xFF

	// =========================
	// Decode Link Monsters
	// =========================

	// TYPE_LINK = 0x4000000
	const TYPE_LINK = 0x4000000

	if c.Type&TYPE_LINK != 0 {
		// For link monsters:
		// - Defense field stores link marker bitmask
		// - Level actually stores link rating (in lowest byte)
		c.LinkMarker = uint32(c.Defense)
		c.Defense = 0
		// c.Level already contains correct link rating
	}

	return &c, nil
}

func (d *DB) GetSchema() (string, error) {
	rows, err := d.db.Query(`
		SELECT type, name, sql
		FROM sqlite_master
		WHERE name NOT LIKE 'sqlite_%'
		ORDER BY type, name;
	`)
	if err != nil {
		return "", fmt.Errorf("failed to query schema: %w", err)
	}
	defer rows.Close()

	var schema string
	for rows.Next() {
		var objType, name string
		var sqlStmt sql.NullString

		if err := rows.Scan(&objType, &name, &sqlStmt); err != nil {
			return "", fmt.Errorf("failed scanning schema row: %w", err)
		}

		if sqlStmt.Valid {
			schema += fmt.Sprintf("-- %s: %s\n%s;\n\n", objType, name, sqlStmt.String)
		}
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error iterating schema rows: %w", err)
	}

	return schema, nil
}
