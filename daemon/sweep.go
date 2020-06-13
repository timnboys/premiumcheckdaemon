package daemon

import (
	"context"
	"github.com/TicketsBot/common/premium"
	"github.com/TicketsBot/common/sentry"
	"github.com/TicketsBot/common/whitelabeldelete"
)

func (d *Daemon) sweep() {
	query := `SELECT "user_id" FROM whitelabel;`
	rows, err := d.db.Whitelabel.Query(context.Background(), query)
	defer rows.Close()

	if err != nil {
		sentry.Error(err)
		return
	}

	for rows.Next() {
		var userId uint64
		if err := rows.Scan(&userId); err != nil {
			sentry.Error(err)
			continue
		}

		go func() {
			hasWhitelabel, err := d.hasWhitelabel(userId)
			if err != nil {
				sentry.Error(err)
				return
			}

			if !hasWhitelabel {
				// get bot ID
				bot, err := d.db.Whitelabel.GetByUserId(userId)
				if err != nil {
					sentry.Error(err)
					return
				}

				if err := d.db.Whitelabel.Delete(userId); err != nil {
					sentry.Error(err)
					return
				}

				whitelabeldelete.Publish(d.redis, bot.BotId)
			}
		}()
	}
}

// use our own function w/ error handling
func (d *Daemon) hasWhitelabel(userId uint64) (bool, error) {
	tier, err := d.premium.GetTier(userId)
	if err != nil {
		return false, err
	}

	if tier >= premium.Whitelabel {
		return true, nil
	}

	for _, forced := range d.forced {
		if forced == userId {
			return true, nil
		}
	}

	return false, nil
}

