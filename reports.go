package main

import (
	"database/sql"
	"time"
)

const (
	KillEventsQuery = `
		SELECT
			DISTINCT ON (ke.created_at, ke.killed_player_id, ke.killer_player_id)
			k1.name AS killed,
			k2.name AS killer,
			COALESCE(w.name_ru, w.name) AS weapon,
			COALESCE(bp.name_ru, bp.name) AS body_part,
			ke.created_at AS created_at
		FROM
			kill_events ke
		LEFT JOIN players k1 ON 
			ke.killed_player_id = k1.id
		LEFT JOIN players k2 ON 
			ke.killer_player_id = k2.id
		LEFT JOIN damage_events de ON 
			ke.killed_player_id = de.received_player_id 
			AND ke.killer_player_id = de.dealt_player_id
			AND de.created_at BETWEEN ke.created_at - INTERVAL '3 seconds' AND ke.created_at
		LEFT JOIN public.weapons w ON 
			de.weapon_id = w.id
		LEFT JOIN public.body_parts bp ON 
			de.body_part_id = bp.id
		WHERE 
			ke.created_at::date = ?::date
		GROUP BY 
			k1.name, 
			k2.name, 
			weapon, 
			body_part, 
			ke.created_at,
			ke.killed_player_id,
			ke.killer_player_id
		ORDER BY 		
			ke.created_at ASC,
			ke.killed_player_id DESC,
			ke.killer_player_id DESC
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