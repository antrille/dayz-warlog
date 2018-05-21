package main

import (
	"fmt"
	"github.com/jinzhu/configor"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
	"os"
	"regexp"
	"time"
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
		Host     string `default:"localhost"`
		Name     string `default:"dayzone"`
		User     string `default:"postgres"`
		Password string `default:"postgres"`
		Port     uint   `default:"5432"`
		SSLMode  string `default:"disable"`
	}

	Servers []struct {
		Schema	 string
		Name     string `required:"true"`
		FullName string `required:"true"`
	} `required:"true"`
}{}

var CurrentServerIndex = -1

func main() {
	err := os.MkdirAll("reports", 0644)
	if err != nil {
		panic(err)
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage:\n" +
			os.Args[0]+" --parse <log_filename> <server_name>\n" +
			os.Args[0]+" --report <DD.MM.YYYY> <server_name>")
		return
	}

	command := os.Args[1]
	if command != "--parse" && command != "--report" {
		fmt.Println("Invalid command input. Expected \"--parse\" or \"--report\"")
		return
	}

	if len(os.Args) != 4 {
		fmt.Printf("Not enough arguments for \"%s\".\n", command)
		return
	}

	err = configor.Load(&Config, "config.yml")
	if err != nil {
		panic(err)
	}

	serverName := os.Args[3]
	for i, s := range Config.Servers {
		Config.Servers[i].Schema = "srv_" + s.Name

		if CurrentServerIndex == -1 && s.Name == serverName {
			CurrentServerIndex = i
		}
	}

	if CurrentServerIndex == -1 {
		fmt.Printf("Server \"%s\" is not found, check your config file.\n", serverName)
		return
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

	fmt.Print("\nDayZ Warlog Server \nVersion 1.1\n---------------------------\n\n")

	log.Println("Initializing...")
	fmt.Println("Initializing...")
	fmt.Println("Compiling regular expressions...")

	LogStartRegexp = regexp.MustCompile(`AdminLog started on (?P<_0>.+) at (?P<_1>.+)`)
	KilledRegexp = regexp.MustCompile(`(?P<_0>.+) \| Player "(?P<_1>.+)"\(id=(?P<_2>.+)\) has been killed by player "(?P<_3>.+)"\(id=(?P<_4>.+)\)`)
	HitRegexp = regexp.MustCompile(`(?P<_0>.+) \| "(?P<_1>.+)\(uid=(?P<_2>.+)\) HIT (?P<_3>.+)\(uid=(?P<_4>.+)\) by (?P<_5>.+) into (?P<_6>.+)\."`)
	ShotRegexp = regexp.MustCompile(`(?P<_0>.+) \| "(?P<_1>.+)\(uid=(?P<_2>.+)\) SHOT (?P<_3>.+)\(uid=(?P<_4>.+)\) by (?P<_5>.+) into (?P<_6>.+)\."`)

	fmt.Println("Connecting to database...")
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

	fmt.Println("Applying migrations...")

	//db.LogMode(true)

	// Creating schemas for servers and dictionaries, applying migrations
	schema := Config.Servers[CurrentServerIndex].Schema

	db.Exec("CREATE SCHEMA IF NOT EXISTS public")
	db.Exec("CREATE SCHEMA IF NOT EXISTS "+schema)

	db.Exec("SET search_path TO public")
	db.AutoMigrate(&BodyPart{}, &Weapon{})

	db.Exec("SET search_path TO "+schema)
	db.AutoMigrate(&Player{}, &ServerEvent{}, &KillEvent{}, &DamageEvent{})

	db.Exec("SET timezone TO 'UTC'")
	db.Exec("SET search_path TO "+schema+", public")

	log.Println("Server is initialized!")
	fmt.Println("Server is initialized!")

	switch os.Args[1] {
	case "--parse":
		ParseLogFile(os.Args[2])
		break
	case "--report":
		day, err := time.Parse("02.01.2006", os.Args[2])
		if err != nil {
			panic(err)
		}

		CreateDailyReport(CurrentServerIndex, day)
		break
	default:
		break
	}
}