package main

import (
	"fmt"
	"github.com/360EntSecGroup-Skylar/excelize"
	"strconv"
	"time"
)

func DumpDayReport(day time.Time) {
	xlsx := excelize.NewFile()

	rows, err := GetKillEventsList(day)

	if err != nil {
		panic(err)
	}

	headerStyle, _ := xlsx.NewStyle(`
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

	cellHeaderStyle, _ := xlsx.NewStyle(`
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

	cellStyle, _ := xlsx.NewStyle(`
		{
			"font": {
				"color":"000000",
				"size":11,
				"bold":false
			},
			"alignment": {
				"horizontal":"left",
				"vertical":"center"
			},
			"border": [
				{"type":"left","color":"000000","style":1},
				{"type":"top","color":"000000","style":1},
				{"type":"bottom","color":"000000","style":1},
				{"type":"right","color":"000000","style":1}
			]
		}`)

	cellStyleEven, _ := xlsx.NewStyle(`
		{
			"font": {
				"color":"000000",
				"name":"Calibri",
				"size":11,
				"bold":false
			},
			"fill": {
				"type":"pattern",
				"color":["#DDDDDD"],
				"pattern":1
			},
			"alignment": {
				"horizontal":"left",
				"vertical":"center"
			},
			"border": [
				{"type":"left","color":"000000","style":1},
				{"type":"top","color":"000000","style":1},
				{"type":"bottom","color":"000000","style":1},
				{"type":"right","color":"000000","style":1}
			]
		}`)

	if err != nil {
		panic(err)
	}

	xlsx.SetRowHeight("Sheet1", 2, 24.0)
	xlsx.SetColWidth("Sheet1", "B", "E", 30)
	xlsx.SetColWidth("Sheet1", "D", "D", 30)
	xlsx.SetColWidth("Sheet1", "E", "E", 15)
	xlsx.SetColWidth("Sheet1", "A", "A", 10)
	xlsx.SetColWidth("Sheet1", "H", "H", 5)
	xlsx.SetColWidth("Sheet1", "I", "I", 30)
	xlsx.SetColWidth("Sheet1", "J", "K", 10)
	xlsx.SetCellStyle("Sheet1", "A3", "E3", headerStyle)
	xlsx.SetCellStyle("Sheet1", "H3", "K3", headerStyle)
	xlsx.SetCellStr("Sheet1", "A3", "Время")
	xlsx.SetCellStr("Sheet1", "B3", "Убитый")
	xlsx.SetCellStr("Sheet1", "C3", "Убийца")
	xlsx.SetCellStr("Sheet1", "D3", "Оружие")
	xlsx.SetCellStr("Sheet1", "E3", "Часть тела")
	xlsx.SetCellStr("Sheet1", "H3", "#")
	xlsx.SetCellStr("Sheet1", "I3", "Игрок")
	xlsx.SetCellStr("Sheet1", "J3", "Убийств")
	xlsx.SetCellStr("Sheet1", "K3", "Смертей")

	xlsx.SetCellStyle("Sheet1", "A2", "E2", cellHeaderStyle)
	xlsx.MergeCell("Sheet1", "A2", "E2")
	xlsx.SetCellStr(
		"Sheet1",
		"A2",
		fmt.Sprintf(
			"Статистика убийств на сервере \"%s\" за %s",
			Config.Servers[CurrentServerIndex].FullName,
			day.Format("02.01.2006"),
		),
	)

	xlsx.SetCellStyle("Sheet1", "H2", "K2", cellHeaderStyle)
	xlsx.MergeCell("Sheet1", "H2", "K2")
	xlsx.SetCellStr(
		"Sheet1",
		"H2",
		"СПИСКИ ЛИДЕРОВ",
	)

	r := 4
	for rows.Next() {
		var killed, killer, weapon, bodyPart string
		var createdAt time.Time

		rows.Scan(&killed, &killer, &weapon, &bodyPart, &createdAt)

		i := strconv.Itoa(r)
		xlsx.SetCellStr("Sheet1", "A"+i, createdAt.Format("15:04:05"))
		xlsx.SetCellStr("Sheet1", "B"+i, killed)
		xlsx.SetCellStr("Sheet1", "C"+i, killer)
		xlsx.SetCellStr("Sheet1", "D"+i, weapon)
		xlsx.SetCellStr("Sheet1", "E"+i, bodyPart)

		if r%2 != 0 {
			xlsx.SetCellStyle("Sheet1", "A"+i, "E"+i, cellStyleEven)
		} else {
			xlsx.SetCellStyle("Sheet1", "A"+i, "E"+i, cellStyle)
		}

		r++
	}

	rows, err = GetLeadersList(day)

	if err != nil {
		panic(err)
	}

	r = 4
	n := 1
	for rows.Next() {
		i := strconv.Itoa(r)

		var name string
		var kills, deaths int
		rows.Scan(&name, &kills, &deaths)

		xlsx.SetCellInt("Sheet1", "H"+i, n)
		xlsx.SetCellStr("Sheet1", "I"+i, name)
		xlsx.SetCellInt("Sheet1", "J"+i, kills)
		xlsx.SetCellInt("Sheet1", "K"+i, deaths)

		if r%2 != 0 {
			xlsx.SetCellStyle("Sheet1", "H"+i, "K"+i, cellStyleEven)
		} else {
			xlsx.SetCellStyle("Sheet1", "H"+i, "K"+i, cellStyle)
		}

		r++
		n++
	}

	xlsx.SetActiveSheet(0)
	xlsx.SetSheetName("Sheet1", "За день")

	fileName := fmt.Sprintf(
		"%s_%s.xlsx",
		Config.Servers[CurrentServerIndex].Name,
		day.Format("02.01.2006"),
	)

	err = xlsx.SaveAs(fileName)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Report saved to \"%s\"!\n", fileName)
}
