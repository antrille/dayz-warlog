package main

import (
	"regexp"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/jinzhu/gorm"
	"fmt"
	"time"
	"github.com/360EntSecGroup-Skylar/excelize"
	"strconv"
	"sort"
)

type PlayerSummary struct {
	Name string
	Kills int
	Deaths int
}

type PlayerList []PlayerSummary

var (
	LogStartRegexp *regexp.Regexp
	KilledRegexp   *regexp.Regexp
	HitRegexp      *regexp.Regexp
	ShotRegexp     *regexp.Regexp

	db *gorm.DB
)

func main() {
	fmt.Print("\nDayZ Warlog Server \nVersion 1.0\n---------------------------\n\n")

	LogStartRegexp = regexp.MustCompile(`\x00AdminLog started on (?P<_0>.+) at (?P<_1>.+)`)
	KilledRegexp = regexp.MustCompile(`(?P<_0>.+) \| Player \"(?P<_1>.+)\"\(id=(?P<_2>.+)\) has been killed by player \"(?P<_3>.+)\"\(id=(?P<_4>.+)\)`)
	HitRegexp = regexp.MustCompile(`(?P<_0>.+) \| \"(?P<_1>.+)\(uid=(?P<_2>.+)\) HIT (?P<_3>.+)\(uid=(?P<_4>.+)\) by (?P<_5>.+) into (?P<_6>.+)\.\"`)
	ShotRegexp = regexp.MustCompile(`(?P<_0>.+) \| \"(?P<_1>.+)\(uid=(?P<_2>.+)\) SHOT (?P<_3>.+)\(uid=(?P<_4>.+)\) by (?P<_5>.+) into (?P<_6>.+)\.\"`)

	fmt.Println("Connection to database...")

	var err error
	db, err = gorm.Open(
		"postgres",
		"host=localhost port=5432 user=postgres dbname=dayzone password=postgres sslmode=disable",
	)

	if err != nil {
		fmt.Println(err)
		panic("failed to connect database")
	}

	defer db.Close()

	//db.LogMode(true)

	fmt.Println("Applying migrations...")
	db.AutoMigrate(&Player{},  &Weapon{}, &BodyPart{}, &ServerEvent{}, &KillEvent{}, &DamageEvent{})

	fmt.Println("Ready to parse!")
	ParseLogFile("dayz/DayZServer_x64.ADM")

	GenerateLogForDay(time.Date(2018, 4, 7, 0, 0, 0, 0, time.UTC))
}

func GenerateLogForDay(day time.Time) {
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

	xlsx.SetColWidth("Sheet1", "B", "E", 30)
	xlsx.SetColWidth("Sheet1", "D", "D", 25)
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


	xlsx.MergeCell("Sheet1", "A2", "E2")
	xlsx.SetCellStr(
		"Sheet1",
		"A2",
		fmt.Sprintf(
			"Статистика сервера \"DAYZONE #1 (Hardcore)\" за %s. Сгенерировано %s.",
			day.Format("02.01.2006"),
			time.Now().Format("02.01.2006 15:04:05"),
		),
	)
	xlsx.MergeCell("Sheet1", "H2", "K2")
	xlsx.SetCellStr(
		"Sheet1",
		"H2",
		"Списки лидеров",
	)
	r := 4
	kills := make(map[string]int)
	deaths := make(map[string]int)

	for rows.Next() {
		var killed, killer, weapon, bodyPart string
		var createdAt time.Time

		rows.Scan(&killed, &killer, &weapon, &bodyPart, &createdAt)

		kills[killer]++
		deaths[killed]++

		i := strconv.Itoa(r)
		xlsx.SetCellStr("Sheet1", "A"+i, createdAt.Format("15:04:05"))
		xlsx.SetCellStr("Sheet1", "B"+i, killed)
		xlsx.SetCellStr("Sheet1", "C"+i, killer)
		xlsx.SetCellStr("Sheet1", "D"+i, weapon)
		xlsx.SetCellStr("Sheet1", "E"+i, bodyPart)

		if r % 2 != 0 {
			xlsx.SetCellStyle("Sheet1", "A"+i, "E"+i, cellStyleEven)
		} else {
			xlsx.SetCellStyle("Sheet1", "A"+i, "E"+i, cellStyle)
		}

		r++
	}

	// Make player summary list
	pl := make(PlayerList, len(kills))
	i := 0
	for k, v := range kills {
		pl[i] = PlayerSummary{Name:k, Kills:v, Deaths:deaths[k]}
		i++
	}
	sort.Sort(pl)

	r = 4
	for n, v := range pl {
		i := strconv.Itoa(r)

		xlsx.SetCellInt("Sheet1", "H"+i, n+1)
		xlsx.SetCellStr("Sheet1", "I"+i, v.Name)
		xlsx.SetCellInt("Sheet1", "J"+i, v.Kills)
		xlsx.SetCellInt("Sheet1", "K"+i, v.Deaths)

		if r % 2 != 0 {
			xlsx.SetCellStyle("Sheet1", "H"+i, "K"+i, cellStyleEven)
		} else {
			xlsx.SetCellStyle("Sheet1", "H"+i, "K"+i, cellStyle)
		}
		r++
	}

	xlsx.SetSheetName("Sheet1", "Статистика")
	err = xlsx.SaveAs("dayz/"+day.Format("02.01.2006")+".xlsx")
	if err != nil {
		panic(err)
	}
}

func (p PlayerList) Len() int { return len(p) }
func (p PlayerList) Less(i, j int) bool {
	if p[i].Kills == p[j].Kills {return p[i].Deaths < p[j].Deaths} else {return p[i].Kills > p[j].Kills}
}
func (p PlayerList) Swap(i, j int){ p[i], p[j] = p[j], p[i] }