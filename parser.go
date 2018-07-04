package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"regexp"

	"github.com/lib/pq"
)

var (
	logStartRegexp = regexp.MustCompile(`AdminLog started on (?P<_0>.+) at (?P<_1>.+)`)
	killedRegexp   = regexp.MustCompile(`(?P<_0>.+) \| Player "(?P<_1>.+)"\(id=(?P<_2>.+)\) has been killed by player "(?P<_3>.+)"\(id=(?P<_4>.+)\)`)
	hitRegexp      = regexp.MustCompile(`(?P<_0>.+) \| "(?P<_1>.+)\(uid=(?P<_2>.+)\) HIT (?P<_3>.+)\(uid=(?P<_4>.+)\) by (?P<_5>.+) into (?P<_6>.+)\."`)
	shotRegexp     = regexp.MustCompile(`(?P<_0>.+) \| "(?P<_1>.+)\(uid=(?P<_2>.+)\) SHOT (?P<_3>.+)\(uid=(?P<_4>.+)\) by (?P<_5>.+) into (?P<_6>.+)\."`)
)

func ParseLogFile(fn string) bool {
	f, err := os.Open(fn)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()

		matches := logStartRegexp.FindAllStringSubmatch(line, -1)
		if matches == nil {
			continue
		}

		t, _ := time.Parse("2006-01-02 15:04:05", matches[0][1]+" "+matches[0][2])
		e := ServerEvent{Type: "restart", CreatedAt: t}

		db.FirstOrCreate(&e, e)

		ParseLogPart(s, t)
	}

	if err := s.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
		return false
	}

	return true
}

func ParseLogPart(s *bufio.Scanner, t time.Time) {
	var initTime pq.NullTime

	for s.Scan() {
		line := s.Text()

		if len(line) == 0 {
			continue
		}

		if line[0] == 0 || line[0] == '*' {
			return
		}

		t2, _ := time.Parse("15:04:05", line[:strings.Index(line, " |")])

		eventTime := time.Date(t.Year(), t.Month(), t.Day(), t2.Hour(), t2.Minute(), t2.Second(), 0, time.UTC)

		// Day span fix
		if !initTime.Valid {
			initTime.Scan(eventTime)
		}

		if eventTime.Before(initTime.Time) {
			eventTime = eventTime.AddDate(0, 0, 1)
		}

		matches := killedRegexp.FindAllStringSubmatch(line, -1)
		if matches != nil {
			id, err := strconv.ParseInt(matches[0][5], 10, 64)
			if err != nil {
				fmt.Println(err)
				continue
			}

			killer := CreateOrUpdatePlayer(id, matches[0][4])

			id, err = strconv.ParseInt(matches[0][3], 10, 64)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}

			killed := CreateOrUpdatePlayer(id, matches[0][2])

			e := KillEvent{
				KillerPlayerId: killer.Id,
				KilledPlayerId: killed.Id,
				CreatedAt: eventTime,
			}

			db.FirstOrCreate(&e, e)

			continue
		}

		matches = hitRegexp.FindAllStringSubmatch(line, -1)

		if matches == nil {
			matches = shotRegexp.FindAllStringSubmatch(line, -1)
		}

		if matches != nil {
			id, err := strconv.ParseInt(matches[0][3], 10, 64)
			if err != nil {
				fmt.Println(err)
				continue
			}

			dealer := CreateOrUpdatePlayer(id, matches[0][2])

			id, err = strconv.ParseInt(matches[0][5], 10, 64)
			if err != nil {
				fmt.Println(err)
				continue
			}

			receiver := CreateOrUpdatePlayer(id, matches[0][4])

			weapon := Weapon{Name: strings.ToLower(matches[0][6])}
			db.FirstOrCreate(&weapon, weapon)

			bodyPart := BodyPart{Name: strings.ToLower(matches[0][7])}
			db.FirstOrCreate(&bodyPart, bodyPart)

			e := DamageEvent{
				DealtPlayerId:    dealer.Id,
				ReceivedPlayerId: receiver.Id,
				WeaponId:         weapon.Id,
				BodyPartId:       bodyPart.Id,
				CreatedAt: 		  eventTime,
			}

			db.FirstOrCreate(&e, e)

			continue
		}
	}
}