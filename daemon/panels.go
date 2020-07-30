package daemon

import (
	"context"
	"fmt"
	"github.com/TicketsBot/common/premium"
	"github.com/TicketsBot/common/sentry"
	"github.com/jackc/pgx/v4"
)

func (d *Daemon) sweepPanels() {
	rows, err := d.db.Panel.Query(context.Background(), `SELECT "guild_id", COUNT(*) FROM panels GROUP BY guild_id HAVING COUNT(*) > 1;`)
	defer rows.Close()
	if err != nil {
		sentry.Error(err)
		return
	}

	guilds := make(map[uint64]int)
	for rows.Next() {
		var guildId uint64
		var panelCount int
		if err := rows.Scan(&guildId, &panelCount); err != nil {
			sentry.Error(err)
			continue
		}

		guilds[guildId] = panelCount
	}

	fmt.Printf("Detected %d guilds with > 1 panel\n", len(guilds))

	batch := &pgx.Batch{}

	for guildId, panelCount := range guilds {
		// get tier from patreon
		tier, err := d.patreon.GetTier(guildId)
		if err != nil {
			sentry.Error(err)
			continue
		}

		if tier < premium.Premium {
			// if not premium on patreon, we should check for a premium key
			hasKey, err := d.db.PremiumGuilds.IsPremium(guildId)
			if err != nil {
				sentry.Error(err)
				continue
			}

			if !hasKey {
				fmt.Printf("guild %d is not patreon anymore! panel count: %d\n", guildId, panelCount)
				/*query := `
				DELETE FROM
					panels
				WHERE
					"message_id" IN (
						SELECT
							"message_id"
						FROM
							panels
						WHERE
							"guild_id" = $1
						LIMIT $2
					)
				;
				`
							batch.Queue(query, guildId, panelCount - 1)*/
			}
		}
	}

	if _, err := d.db.Panel.SendBatch(context.Background(), batch).Exec(); err != nil {
		sentry.Error(err)
	}
}
