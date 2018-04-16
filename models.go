package main

import (
	"time"
	"strings"
	"io/ioutil"
	"golang.org/x/text/transform"
	"golang.org/x/text/encoding/charmap"
	"database/sql"
)

type Player struct {
	Id   int64 `gorm:"primary_key;unique;not null"`
	Name string
}

type Weapon struct {
	Id     uint `gorm:"primary_key;unique;not null"`
	Name   string
	NameRu sql.NullString
}

type BodyPart struct {
	Id     uint `gorm:"primary_key;unique;not null"`
	Name   string
	NameRu sql.NullString
}

type ServerEvent struct {
	Id        uint `gorm:"primary_key;unique;not null"`
	Type      string
	CreatedAt time.Time
}

type KillEvent struct {
	Id             uint   `gorm:"primary_key;unique;not null"`
	Killed         Player `gorm:"foreignkey:KilledPlayerId"`
	KilledPlayerId int64  `gorm:"type:bigint REFERENCES players(id) ON DELETE CASCADE"`
	Killer         Player `gorm:"foreignkey:KillerPlayerId"`
	KillerPlayerId int64  `gorm:"type:bigint REFERENCES players(id) ON DELETE CASCADE"`
	CreatedAt      time.Time
}

type DamageEvent struct {
	Id               uint     `gorm:"primary_key;unique;not null"`
	DealtPlayer      Player   `gorm:"foreignkey:DealtPlayerId"`
	DealtPlayerId    int64    `gorm:"type:bigint REFERENCES players(id) ON DELETE CASCADE"`
	ReceivedPlayer   Player   `gorm:"foreignkey:ReceivedPlayerId"`
	ReceivedPlayerId int64    `gorm:"type:bigint REFERENCES players(id) ON DELETE CASCADE"`
	Weapon           Weapon   `gorm:"foreignkey:WeaponId"`
	WeaponId         uint     `gorm:"REFERENCES weapons(id)"`
	BodyPart         BodyPart `gorm:"foreignkey:BodyPartId"`
	BodyPartId       uint     `gorm:"REFERENCES body_parts(id)"`
	CreatedAt        time.Time
}

func CreateOrUpdatePlayer(id int64, name string) *Player {
	// Convert Windows-1251 name charset to UTF-8
	sr := strings.NewReader(name)
	tr := transform.NewReader(sr, charmap.Windows1251.NewDecoder())
	buf, err := ioutil.ReadAll(tr)
	if err == nil {
		name = string(buf)
	}

	var p Player

	if db.Find(&p, "id = ?", id).RecordNotFound() {
		p.Id = id
		p.Name = name
		db.Create(&p)
	} else if p.Name != name {
		db.Model(&p).Update("name", name)
	}

	return &p
}
