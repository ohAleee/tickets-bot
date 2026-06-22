package daemon

import (
	"context"
	"github.com/TicketsBot/common/autoclose"
)

func (d *Daemon) scan() (tickets []autoclose.Ticket, err error) {
	query := `
SELECT
    t.id,
	t.guild_id,
    tlm.last_message_id
FROM
    tickets t
INNER JOIN auto_close ac
    ON t.guild_id = ac.guild_id
LEFT OUTER JOIN ticket_last_message tlm
    ON t.guild_id = tlm.guild_id AND t.id = tlm.ticket_id
LEFT JOIN auto_close_exclude exclude
	ON t.guild_id = exclude.guild_id and t.id = exclude.ticket_id
WHERE
    ac.enabled 
    AND
    t.open
	AND
	t.channel_id IS NOT NULL
    AND
    (
		(
			tlm.ticket_id IS null
			AND
			(NOW() - t.open_time) >= ac.since_open_with_no_response
		)
		OR
     	(
			(NOW() - tlm.last_message_time) >= ac.since_last_message
		)
	)
	AND
	exclude.guild_id IS NULL
;
`

	// doesn't matter what table we query from, all same conn
	rows, err := d.db.Tickets.Query(context.Background(), query)
	defer rows.Close()

	if err != nil {
		return
	}

	for rows.Next() {
		var ticket autoclose.Ticket
		if err = rows.Scan(&ticket.TicketId, &ticket.GuildId, &ticket.LastMessageId); err != nil {
			return
		}

		tickets = append(tickets, ticket)
	}

	return
}
