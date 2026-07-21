package database

import (
	"context"
	ejson "encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type MultiPanel struct {
	Id                    int                    `json:"id"`
	MessageId             uint64                 `json:"message_id,string"`
	ChannelId             uint64                 `json:"channel_id,string"`
	GuildId               uint64                 `json:"guild_id,string"`
	SelectMenu            bool                   `json:"select_menu"`
	SelectMenuPlaceholder *string                `json:"select_menu_placeholder"`
	Embed                 *CustomEmbedWithFields `json:"embed"`
	// Components holds an optional Discord "Components V2" layout (a JSON array of
	// component objects). When present, the multi-panel message is rendered as a
	// Components V2 message instead of the classic embed + action row.
	Components ejson.RawMessage `json:"components,omitempty"`
}

type MultiPanelTable struct {
	*pgxpool.Pool
}

func newMultiMultiPanelTable(db *pgxpool.Pool) *MultiPanelTable {
	return &MultiPanelTable{
		db,
	}
}

func (MultiPanelTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS multi_panels(
	"id" SERIAL NOT NULL,
	"message_id" int8 NOT NULL,
	"channel_id" int8 NOT NULL,
	"guild_id" int8 NOT NULL,
	"select_menu" bool DEFAULT 'f',
	"select_menu_placeholder" VARCHAR(150) DEFAULT NULL,
	"embed" JSONB DEFAULT NULL,
	"components" JSONB DEFAULT NULL,
	PRIMARY KEY("id")
);
CREATE INDEX IF NOT EXISTS multi_panels_guild_id ON multi_panels("guild_id");
CREATE INDEX IF NOT EXISTS multi_panels_message_id ON multi_panels("message_id");`
}

func (p *MultiPanelTable) Get(ctx context.Context, id int) (MultiPanel, bool, error) {
	query := `
SELECT
	"id", "message_id", "channel_id", "guild_id", "select_menu", "select_menu_placeholder", "embed", "components"
FROM
	multi_panels
WHERE
	"id" = $1
;`

	var panel MultiPanel
	var embedRaw *string
	var componentsRaw *string
	err := p.QueryRow(ctx, query, id).Scan(
		&panel.Id, &panel.MessageId, &panel.ChannelId, &panel.GuildId, &panel.SelectMenu, &panel.SelectMenuPlaceholder, &embedRaw, &componentsRaw,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MultiPanel{}, false, nil
		} else {
			return MultiPanel{}, false, err
		}
	}

	if embedRaw != nil {
		if err := json.Unmarshal([]byte(*embedRaw), &panel.Embed); err != nil {
			return MultiPanel{}, false, err
		}
	}

	if componentsRaw != nil {
		panel.Components = ejson.RawMessage(*componentsRaw)
	}

	return panel, true, nil
}

func (p *MultiPanelTable) GetByMessageId(ctx context.Context, messageId uint64) (MultiPanel, bool, error) {
	query := `
SELECT
	"id", "message_id", "channel_id", "guild_id", "select_menu", "select_menu_placeholder", "embed", "components"
FROM
	multi_panels
WHERE
	"message_id" = $1
;`

	var panel MultiPanel
	var embedRaw *string
	var componentsRaw *string
	err := p.QueryRow(ctx, query, messageId).Scan(
		&panel.Id, &panel.MessageId, &panel.ChannelId, &panel.GuildId, &panel.SelectMenu, &panel.SelectMenuPlaceholder, &embedRaw, &componentsRaw,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MultiPanel{}, false, nil
		} else {
			return MultiPanel{}, false, err
		}
	}

	if embedRaw != nil {
		if err := json.Unmarshal([]byte(*embedRaw), &panel.Embed); err != nil {
			return MultiPanel{}, false, err
		}
	}

	if componentsRaw != nil {
		panel.Components = ejson.RawMessage(*componentsRaw)
	}

	return panel, true, nil
}

func (p *MultiPanelTable) GetByGuild(ctx context.Context, guildId uint64) ([]MultiPanel, error) {
	query := `
SELECT "id", "message_id", "channel_id", "guild_id", "select_menu", "select_menu_placeholder", "embed", "components"
FROM multi_panels
WHERE "guild_id" = $1;
`

	rows, err := p.Query(ctx, query, guildId)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	var panels []MultiPanel
	for rows.Next() {
		var panel MultiPanel
		var embedRaw *string
		var componentsRaw *string
		err := rows.Scan(
			&panel.Id, &panel.MessageId, &panel.ChannelId, &panel.GuildId, &panel.SelectMenu, &panel.SelectMenuPlaceholder, &embedRaw, &componentsRaw,
		)

		if err != nil {
			return nil, err
		}

		if embedRaw != nil {
			if err := json.Unmarshal([]byte(*embedRaw), &panel.Embed); err != nil {
				return nil, err
			}
		}

		if componentsRaw != nil {
			panel.Components = ejson.RawMessage(*componentsRaw)
		}

		panels = append(panels, panel)
	}

	return panels, nil
}

func (p *MultiPanelTable) Create(ctx context.Context, panel MultiPanel) (int, error) {
	query := `
INSERT INTO
	multi_panels("message_id", "channel_id", "guild_id", "select_menu", "select_menu_placeholder", "embed", "components")
VALUES
	($1, $2, $3, $4, $5, $6, $7)
RETURNING
	"id"
;
`

	var embedRaw *string
	if panel.Embed != nil {
		embedRawBytes, err := json.Marshal(panel.Embed)
		if err != nil {
			return 0, err
		}

		embedRaw = ptr(string(embedRawBytes))
	}

	var componentsRaw *string
	if raw := strings.TrimSpace(string(panel.Components)); raw != "" && raw != "null" {
		componentsRaw = ptr(raw)
	}

	var multiPanelId int
	if err := p.QueryRow(ctx, query,
		panel.MessageId, panel.ChannelId, panel.GuildId, panel.SelectMenu, panel.SelectMenuPlaceholder, embedRaw, componentsRaw,
	).Scan(&multiPanelId); err != nil {
		return 0, err
	}

	return multiPanelId, nil
}

func (p *MultiPanelTable) Update(ctx context.Context, multiPanelId int, multiPanel MultiPanel) (err error) {
	query := `
UPDATE multi_panels
	SET "message_id" = $2,
		"channel_id" = $3,
		"select_menu" = $4,
		"select_menu_placeholder" = $5,
		"embed" = $6,
		"components" = $7
	WHERE
		"id" = $1
;`

	var embedRaw *string
	if multiPanel.Embed != nil {
		embedRawBytes, err := json.Marshal(multiPanel.Embed)
		if err != nil {
			return err
		}

		embedRaw = ptr(string(embedRawBytes))
	}

	var componentsRaw *string
	if raw := strings.TrimSpace(string(multiPanel.Components)); raw != "" && raw != "null" {
		componentsRaw = ptr(raw)
	}

	_, err = p.Exec(ctx, query,
		multiPanelId, multiPanel.MessageId, multiPanel.ChannelId, multiPanel.SelectMenu, multiPanel.SelectMenuPlaceholder, embedRaw, componentsRaw,
	)

	return
}

func (p *MultiPanelTable) UpdateMessageId(ctx context.Context, multiPanelId int, messageId uint64) (err error) {
	query := `
UPDATE multi_panels
SET "message_id" = $1
WHERE "id" = $2;
`

	_, err = p.Exec(ctx, query, messageId, multiPanelId)
	return
}

func (p *MultiPanelTable) Delete(ctx context.Context, guildId uint64, multiPanelId int) (success bool, err error) {
	query := `
WITH deleted AS (
	DELETE FROM
		multi_panels
	WHERE
		"guild_id" = $1
		AND
		"id" = $2
	RETURNING *
)

SELECT
	COUNT(*)
FROM
	deleted
;
`

	var count int
	err = p.QueryRow(ctx, query, guildId, multiPanelId).Scan(&count)
	success = count > 0

	return
}
