package main

import (
	"fmt"
	"log"
	"os"
	"time"
	"flag"

	"github.com/jinzhu/configor"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"gopkg.in/natefinch/lumberjack.v2"
	"strings"
)

var (
	db               *gorm.DB
	currentServerIdx = -1

	lang           	 = struct {
		Header string `required:"true"`
		Time string `required:"true"`
		Killed string `required:"true"`
		Killer string `required:"true"`
		Weapon string `required:"true"`
		BodyPart string `required:"true"`
		Player string `required:"true"`
		Kills string `required:"true"`
		Deaths string `required:"true"`
		LeadersList string `required:"true"`
		Unknown string `required:"true"`
		DailyReport string `required:"true"`
	}{}

	config           = struct {
		LogFile string `default:"server.log"`
		Lang string `default:"en"`

		DB struct {
			Host     string `default:"localhost"`
			Name     string `default:"dayzone"`
			User     string `default:"postgres"`
			Password string `default:"postgres"`
			Port     uint   `default:"5432"`
			SSLMode  string `default:"disable"`
		}

		Servers []struct {
			Name     string `required:"true"`
			FullName string `required:"true"`
		} `required:"true"`
	}{}
)

func PrintUsage() {
	fmt.Println("Usage: dayz-warlog <command> [<args>]")
	fmt.Println("Commands: ")
	fmt.Println(" parse   Parse DayZ log file data in Warlog database")
	fmt.Println(" report  Generate frag report in XLSX format for specific day")
}

func main() {
	err := os.MkdirAll("reports", 0644)
	if err != nil {
		panic(err)
	}

	var serverName string

	parseCommand := flag.NewFlagSet("parse", flag.ExitOnError)
	parseFile := parseCommand.String("file", "", "Log filename to parse")
	parseServerName := parseCommand.String("server", "", "Server name from config file")

	reportCommand := flag.NewFlagSet("report", flag.ExitOnError)
	reportDate := reportCommand.String("date", "", "Date in DD.MM.YYYY format")
	reportServerName := reportCommand.String("server", "", "Server name from config file")

	if len(os.Args) == 1 {
		PrintUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "parse":
		parseCommand.Parse(os.Args[2:])
	case "report":
		reportCommand.Parse(os.Args[2:])
	default:
		PrintUsage()
		os.Exit(2)
	}

	if parseCommand.Parsed() {
		if *parseFile == "" {
			fmt.Println("You must provide a filename to parse in -file option.")
			return
		}

		if *parseServerName == "" {
			fmt.Println("You must provide a server name in -server option.")
			return
		}

		if _, err := os.Stat(*parseFile); os.IsNotExist(err) {
			fmt.Printf("File \"%s\" doesn't exist.\n", *parseFile)
			return
		}

		serverName = *parseServerName
	}

	if reportCommand.Parsed() {
		if *reportDate == "" {
			fmt.Println("You must provide a date in DD.MM.YYYY format in -date option.")
			return
		}

		if *reportServerName == "" {
			fmt.Println("You must provide a server name in -server option.")
			return
		}

		serverName = *reportServerName
	}

	err = configor.Load(&config, "config.yml")
	if err != nil {
		panic(err)
	}

	for i, s := range config.Servers {
		if  s.Name == serverName {
			currentServerIdx = i
			break
		}
	}

	if currentServerIdx == -1 {
		fmt.Printf("Server \"%s\" is not found, check your config file.\n", serverName)
		return
	}

	log.SetOutput(&lumberjack.Logger{
		Filename:   config.LogFile,
		MaxSize:    20, // megabytes
		MaxBackups: 10,
		MaxAge:     7, // days
		Compress:   true,
	})

	log.Println("------------------------------------------------------")
	log.Println("Log started.")
	log.Println("------------------------------------------------------")

	fmt.Print("\nDayZ Warlog\nVersion 1.2\n---------------------------\n\n")

	fmt.Println("Connecting to database...")
	db, err = gorm.Open(
		"postgres",
		fmt.Sprintf(
			"host=%s port=%d user=%s dbname=%s password=%s sslmode=%s",
			config.DB.Host,
			config.DB.Port,
			config.DB.User,
			config.DB.Name,
			config.DB.Password,
			config.DB.SSLMode,
		),
	)

	if err != nil {
		log.Panicln(err)
	}

	defer db.Close()

	fmt.Println("Applying database migrations...")

	//db.LogMode(true)

	schema := "srv_"+config.Servers[currentServerIdx].Name

	db.Exec("CREATE SCHEMA IF NOT EXISTS public")
	db.Exec("CREATE SCHEMA IF NOT EXISTS "+schema)

	db.Exec("SET search_path TO public")
	db.AutoMigrate(&BodyPart{}, &Weapon{})

	db.Exec("SET search_path TO "+schema)
	db.AutoMigrate(&Player{}, &ServerEvent{}, &KillEvent{}, &DamageEvent{})

	db.Exec("SET timezone TO 'UTC'")
	db.Exec("SET search_path TO "+schema+", public")

	// Load translation
	langFile := fmt.Sprintf("lang/%s.yml", config.Lang)
	err = configor.Load(&lang, langFile)
	if err != nil {
		panic(err)
	}

	// Check table header string for two string verbs
	if strings.Count(lang.Header, "%s") != 2 {
		msg := fmt.Sprintf("Invalid string format for \"%s\": 2 string (%%s) verbs expected.", langFile)
		fmt.Println(msg)
		log.Println(msg)

		return
	}

	switch os.Args[1] {
	case "parse":
		if ParseLogFile(*parseFile) {
			fmt.Println("Log parsing complete.")
		}

		break
	case "report":
		fmt.Printf("Report language: %s\n", config.Lang)

		day, err := time.Parse("02.01.2006", *reportDate)
		if err != nil {
			panic(err)
		}

		xlsx, err := GenerateDailyReport(currentServerIdx, day)
		if err != nil {
			msg := fmt.Sprintf("Report generation failed: %s", err.Error())
			log.Println(msg)
			fmt.Println(msg)

			return
		}

		fileName := fmt.Sprintf(
			"reports/%s_%s.xlsx",
			config.Servers[currentServerIdx].Name,
			day.Format("02.01.2006"),
		)

		err = xlsx.SaveAs(fileName)
		if err != nil {
			msg := fmt.Sprintf("Unable to save generated report: %s\n", err.Error())
			log.Println(msg)
			fmt.Println(msg)

			return
		}

		fmt.Printf("Daily frag report saved to \"%s\".\n", fileName)

		break
	default:
		break
	}
}