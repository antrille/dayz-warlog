package main

import (
	"time"
	"database/sql"
)

const (
	KillEventsQuery = `
		WITH de AS (
			SELECT DISTINCT ON (created_at) *
			FROM 
				damage_events
			ORDER BY 
				created_at DESC, 
				id DESC
		)
		SELECT
			k1.name AS killed,
			k2.name AS killer,
			COALESCE(w.name_ru, w.name) AS weapon,
			COALESCE(bp.name_ru, bp.name) AS body_part,
			kill_events.created_at AS created_at
		FROM
			kill_events
		LEFT JOIN players k1 ON 
			kill_events.killed_player_id = k1.id
		LEFT JOIN players k2 ON 
			kill_events.killer_player_id = k2.id
		LEFT JOIN de ON 
			kill_events.killed_player_id = de.received_player_id 
			AND kill_events.killer_player_id = de.dealt_player_id
		LEFT JOIN weapons w ON 
			de.weapon_id = w.id
		LEFT JOIN body_parts bp ON 
			de.body_part_id = bp.id
		WHERE 
			kill_events.created_at = de.created_at 
			AND kill_events.created_at::date = ?::date
		GROUP BY 
			k1.name, 
			k2.name, 
			weapon, 
			body_part, 
			kill_events.created_at
		ORDER BY 
			kill_events.created_at ASC
	`
)

func GetKillEventsList(day time.Time) (*sql.Rows, error) {
	return db.Raw(KillEventsQuery, day.Format("2006-01-02")).Rows()
}