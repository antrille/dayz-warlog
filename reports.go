package main

import (
	"time"
	"database/sql"
)

const (
	KillEventsQuery = `
		SELECT
			DISTINCT ON (kill_events.created_at, de.received_player_id, de.created_at)
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
		LEFT JOIN damage_events de ON 
			kill_events.killed_player_id = de.received_player_id 
			AND kill_events.killer_player_id = de.dealt_player_id
			AND kill_events.created_at = de.created_at
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
			kill_events.created_at,
			de.received_player_id,
			de.created_at,
			de.id
		ORDER BY 		
			kill_events.created_at ASC,
			de.received_player_id DESC,
			de.created_at DESC,
			de.id DESC
	`
	LeaderListQuery = `
		SELECT
			players.name AS name,
			as_killer.kills AS kills,
			as_killed.deaths AS deaths
		FROM
			players
		RIGHT JOIN (
			SELECT
				DISTINCT(killer_player_id) as player_id,
				COUNT(killer_player_id) as kills
			FROM kill_events
			WHERE
				created_at::date = ?::date
			GROUP BY player_id
		) as_killer ON (as_killer.player_id = players.id)
		LEFT JOIN  (
			SELECT
				DISTINCT(killed_player_id) as player_id,
				COUNT(killed_player_id) as deaths
			FROM kill_events
			WHERE
				kill_events.created_at::date = ?::date
			GROUP BY player_id
		) as_killed ON (as_killed.player_id = players.id)
		GROUP BY
			players.id, players.name, as_killer.kills, as_killed.deaths
		ORDER BY
			kills DESC, deaths ASC
	`
)

func GetKillEventsList(day time.Time) (*sql.Rows, error) {
	return db.Raw(KillEventsQuery, day.Format("2006-01-02")).Rows()
}

func GetLeadersList(day time.Time) (*sql.Rows, error) {
	date := day.Format("2006-01-02")
	return db.Raw(LeaderListQuery, date, date).Rows()
}