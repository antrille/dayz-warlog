package main

import (
	"github.com/jinzhu/configor"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"gopkg.in/natefinch/lumberjack.v2"
	"github.com/360EntSecGroup-Skylar/excelize"
	"regexp"
	"fmt"
	"time"
	"strconv"
	"log"
	"os"
)

var (
	LogStartRegexp *regexp.Regexp
	KilledRegexp   *regexp.Regexp
	HitRegexp      *regexp.Regexp
	ShotRegexp     *regexp.Regexp

	db *gorm.DB
)

var Config = struct {
	LogFile string `default:"server.log"`

	DB struct {
		Host 	 string `default:"localhost"`
		Name     string `default:"dayzone"`
		User     string `default:"postgres"`
		Password string `default:"postgres"`
		Port     uint   `default:"5432"`
		SSLMode	 string	`default:"disable"`
	}

	Servers []struct {
		FullName  string
		ShortName string
	}
}{}

func main() {
	fmt.Println(os.Args)
	if len(os.Args) != 3 {
		fmt.Println("Invalid number of arguments - 2 expected")
		return
	}

	err := configor.Load(&Config, "config.yml")
	if err != nil {
		panic(err)
	}

	log.SetOutput(&lumberjack.Logger{
		Filename:   Config.LogFile,
		MaxSize:    20, // megabytes
		MaxBackups: 10,
		MaxAge:     7, // days
		Compress:   true,
	})

	log.Println("------------------------------------------------------")
	log.Println("Log started.")
	log.Println("------------------------------------------------------")

	log.Println("Initializing...")
	fmt.Print("\nDayZ Warlog Server \nVersion 1.0\n---------------------------\n\n")

	log.Println("Compiling regular expressions...")
	LogStartRegexp = regexp.MustCompile(`\x00AdminLog started on (?P<_0>.+) at (?P<_1>.+)`)
	KilledRegexp = regexp.MustCompile(`(?P<_0>.+) \| Player \"(?P<_1>.+)\"\(id=(?P<_2>.+)\) has been killed by player \"(?P<_3>.+)\"\(id=(?P<_4>.+)\)`)
	HitRegexp = regexp.MustCompile(`(?P<_0>.+) \| \"(?P<_1>.+)\(uid=(?P<_2>.+)\) HIT (?P<_3>.+)\(uid=(?P<_4>.+)\) by (?P<_5>.+) into (?P<_6>.+)\.\"`)
	ShotRegexp = regexp.MustCompile(`(?P<_0>.+) \| \"(?P<_1>.+)\(uid=(?P<_2>.+)\) SHOT (?P<_3>.+)\(uid=(?P<_4>.+)\) by (?P<_5>.+) into (?P<_6>.+)\.\"`)

	log.Println("Connection to database...")
	db, err = gorm.Open(
		"postgres",
		fmt.Sprintf(
			"host=%s port=%d user=%s dbname=%s password=%s sslmode=%s",
			Config.DB.Host,
			Config.DB.Port,
			Config.DB.User,
			Config.DB.Name,
			Config.DB.Password,
			Config.DB.SSLMode,
		),
	)

	if err != nil {
		log.Panicln(err)
	}

	defer db.Close()

	//db.LogMode(true)

	log.Println("Applying migrations...")
	db.AutoMigrate(&Player{},  &Weapon{}, &BodyPart{}, &ServerEvent{}, &KillEvent{}, &DamageEvent{})

	log.Println("Ready for work!")
	fmt.Println("Ready for work!")

	if os.Args[1] == "--parse" {
		ParseLogFile(os.Args[2])
	} else if os.Args[1] == "--report" {
		//GenerateLogForDay(time.Date(2018, 4, 7, 0, 0, 0, 0, time.UTC))
		time, err := time.Parse("02.01.2006", os.Args[2])
		if err != nil {
			panic(err)
		}
		GenerateLogForDay(time)
	} else {
		fmt.Println("Invalid command. Expected \"--parse\" or \"--report\"")
		return
	}
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
			"Статистика сервера за %s. Сгенерировано %s.",
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

		if r % 2 != 0 {
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

		if r % 2 != 0 {
			xlsx.SetCellStyle("Sheet1", "H"+i, "K"+i, cellStyleEven)
		} else {
			xlsx.SetCellStyle("Sheet1", "H"+i, "K"+i, cellStyle)
		}

		r++
		n++
	}

	xlsx.SetSheetName("Sheet1", "Статистика")
	fileName := day.Format("02.01.2006")+".xlsx"
	err = xlsx.SaveAs(fileName)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Report saved to \"%s\"!\n", fileName)
}