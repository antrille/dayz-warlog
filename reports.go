package main

import (
	"database/sql"
	"time"
	"log"
	"strconv"

	"github.com/360EntSecGroup-Skylar/excelize"
	"fmt"
)

func GenerateDailyReport(serverIndex int, day time.Time) (*excelize.File, error) {
	xlsx := excelize.NewFile()
	schema := "srv_" + config.Servers[serverIndex].Name
	fullName := config.Servers[serverIndex].FullName

	killEventsList, err := GetKillEventsList(schema, day)
	if err != nil {
		return nil, err
	}

	leadersList, err := GetLeadersList(schema, day)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	sheet := xlsx.GetSheetName(xlsx.GetActiveSheetIndex())
	
	captionStyle, _ := xlsx.NewStyle(`
		{
			"font": {
				"color":"FFFFFF",
				"size":12,
				"bold":true
			},
			"fill": {
				"type":"pattern",
				"color":["#4B5320"],
				"pattern":1
			},
			"alignment": {
				"horizontal":"center",
				"vertical":"center"
			},
			"border": [
				{"type":"left","color":"000000","style":1},
				{"type":"top","color":"000000","style":1},
				{"type":"bottom","color":"000000","style":1},
				{"type":"right","color":"000000","style":1}
			]
		}`)

	tableHeaderStyle, _ := xlsx.NewStyle(`
		{
			"font": {
				"color":"000000",
				"size":12,
				"bold":true
			},
			"fill": {
				"type":"pattern",
				"color":["#FAFAFA"],
				"pattern":1
			},
			"alignment": {
				"horizontal":"center",
				"vertical":"center"
			},
			"border": [
				{"type":"left","color":"000000","style":1},
				{"type":"top","color":"000000","style":1},
				{"type":"bottom","color":"000000","style":1},
				{"type":"right","color":"000000","style":1}
			]
		}`)

	tableEvenRowStyle, _ := xlsx.NewStyle(`
		{
			"fill": {
				"type":"pattern",
				"color":["#FFFFFF"],
				"pattern":1
			},
			"border": [
				{"type":"left","color":"000000","style":1},
				{"type":"top","color":"000000","style":1},
				{"type":"bottom","color":"000000","style":1},
				{"type":"right","color":"000000","style":1}
			]
		}`)

	tableOddRowStyle, _ := xlsx.NewStyle(`
		{
			"fill": {
				"type":"pattern",
				"color":["#D9D9D9"],
				"pattern":1
			},
			"border": [
				{"type":"left","color":"000000","style":1},
				{"type":"top","color":"000000","style":1},
				{"type":"bottom","color":"000000","style":1},
				{"type":"right","color":"000000","style":1}
			]
		}`)

	xlsx.MergeCell(sheet, "A1", "E1")
	xlsx.SetCellStyle(sheet, "A1", "E1", tableHeaderStyle)
	xlsx.SetCellStyle(sheet, "A2", "E2", captionStyle)

	xlsx.MergeCell(sheet, "H1", "K1")
	xlsx.SetCellStyle(sheet, "H1", "K1", tableHeaderStyle)
	xlsx.SetCellStyle(sheet, "H2", "K2", captionStyle)

	xlsx.SetCellStr(sheet, "A1", fmt.Sprintf(lang.Header, fullName, day.Format("02.01.2006")))
	xlsx.SetCellStr(sheet, "A2", lang.Time)
	xlsx.SetCellStr(sheet, "B2", lang.Killed)
	xlsx.SetCellStr(sheet, "C2", lang.Killer)
	xlsx.SetCellStr(sheet, "D2", lang.Weapon)
	xlsx.SetCellStr(sheet, "E2", lang.BodyPart)

	xlsx.SetCellStr(sheet, "H1", lang.LeadersList)
	xlsx.SetCellStr(sheet, "H2", "#")
	xlsx.SetCellStr(sheet, "I2", lang.Player)
	xlsx.SetCellStr(sheet, "J2", lang.Kills)
	xlsx.SetCellStr(sheet, "K2", lang.Deaths)

	xlsx.SetRowHeight(sheet, 1, 24.0)
	xlsx.SetColWidth(sheet, "A", "A", 12)
	xlsx.SetColWidth(sheet, "B", "E", 30)
	xlsx.SetColWidth(sheet, "D", "D", 32)
	xlsx.SetColWidth(sheet, "E", "E", 16)
	xlsx.SetColWidth(sheet, "H", "H", 6)
	xlsx.SetColWidth(sheet, "I", "I", 30)
	xlsx.SetColWidth(sheet, "J", "K", 10)

	r := 3
	for killEventsList.Next() {
		var killed, killer string
		var weapon, bodyPart sql.NullString
		var createdAt time.Time

		killEventsList.Scan(&killed, &killer, &weapon, &bodyPart, &createdAt)

		i := strconv.Itoa(r)
		xlsx.SetCellStr(sheet, "A"+i, createdAt.Format("15:04:05"))
		xlsx.SetCellStr(sheet, "B"+i, killed)
		xlsx.SetCellStr(sheet, "C"+i, killer)

		if !weapon.Valid {
			weapon.Scan(lang.Unknown)
		}
		xlsx.SetCellStr(sheet, "D"+i, weapon.String)

		if !bodyPart.Valid {
			bodyPart.Scan(lang.Unknown)
		}
		xlsx.SetCellStr(sheet, "E"+i, bodyPart.String)

		r++
	}

	xlsx.AddTable(sheet, "A2", "E"+strconv.Itoa(r-1), `
		{
			"table_style": "TableStyleMedium15",
			"show_row_stripes": true
		}
	`)

	r = 3
	n := 1
	for leadersList.Next() {
		i := strconv.Itoa(r)

		var name string
		var kills, deaths int
		leadersList.Scan(&name, &kills, &deaths)

		xlsx.SetCellInt(sheet, "H"+i, n)
		xlsx.SetCellStr(sheet, "I"+i, name)
		xlsx.SetCellInt(sheet, "J"+i, kills)
		xlsx.SetCellInt(sheet, "K"+i, deaths)

		if r % 2 == 0 {
			xlsx.SetCellStyle(sheet, "H"+i, "K"+i, tableEvenRowStyle)
		} else {
			xlsx.SetCellStyle(sheet, "H"+i, "K"+i, tableOddRowStyle)
		}

		r++
		n++
	}

	xlsx.SetActiveSheet(0)
	xlsx.SetSheetName(sheet, lang.DailyReport)

	return xlsx, nil
}

func GetKillEventsList(schema string, day time.Time) (*sql.Rows, error) {
	db.Exec("SET timezone to 'UTC'")
	db.Exec("SET search_path TO "+schema+", public")

	return db.Raw(`
		SELECT
			DISTINCT ON (ke.created_at, ke.killed_player_id, ke.killer_player_id)
			k1.name AS killed,
			k2.name AS killer,
			COALESCE(w.report_name, w.name) AS weapon,
			COALESCE(bp.report_name, bp.name) AS body_part,
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
	`, day.Format("2006-01-02")).Rows()
}

func GetLeadersList(schema string, day time.Time) (*sql.Rows, error) {
	db.Exec("SET timezone to 'UTC'")
	db.Exec("SET search_path TO "+schema+", public")

	date := day.Format("2006-01-02")
	return db.Raw(`
		SELECT
			p.name AS name,
			as_killer.kills AS kills,
			as_killed.deaths AS deaths
		FROM
			players p
		RIGHT JOIN (
			SELECT
				DISTINCT(killer_player_id) as player_id,
				COUNT(killer_player_id) as kills
			FROM kill_events
			WHERE
				created_at::date = ?::date
			GROUP BY player_id
		) as_killer ON (as_killer.player_id = p.id)
		LEFT JOIN  (
			SELECT
				DISTINCT(killed_player_id) as player_id,
				COUNT(killed_player_id) as deaths
			FROM kill_events
			WHERE
				kill_events.created_at::date = ?::date
			GROUP BY player_id
		) as_killed ON (as_killed.player_id = p.id)
		GROUP BY
			p.id, p.name, kills, deaths
		ORDER BY
			kills DESC, deaths IS NOT NULL, deaths ASC
	`, date, date).Rows()
}