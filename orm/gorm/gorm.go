package main

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"time"
)

type Like struct {
	ID        int    `gorm:"primary_key"`
	Ip        string `gorm:"type:varchar(20);not null;index:ip_idx"`
	Ua        string `gorm:"type:varchar(256);not null;"`
	Title     string `gorm:"type:varchar(128);not null;index:title_idx"`
	Uid       uint32 `gorm:"type:int"`
	Gid       uint32 `gorm:"type:bigint"`
	Hash      uint64 `gorm:"unique_index:hash_idx;"`
	CreatedAt time.Time
}

var db *gorm.DB

func add(id int, Ip, Ua, Title string) {
	user := &Like{ID: id, Ip: Ip, Ua: Ua, Title: Title, Hash: uint64(id)}
	fmt.Println("line", db.Create(user).RowsAffected)
}

func mode(id int) {
	user := &Like{ID: id}
	fmt.Println("update line", db.Model(user).Update("Title", "mode tssss").RowsAffected)
}

func queryAll() {
	var like []Like
	db.Find(&like)
	fmt.Println("query all:", like)
}

func querySome(some int) {
	user := new(Like)
	db.First(user, some)
	fmt.Println(user)
}
func del(id int) {
	user := &Like{ID: id}
	db.Delete(user)
}

func init() {
	var err error
	db, err = gorm.Open("mysql", "root:123456@(127.0.0.1:3306)/test?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		panic(err)
	}

	if !db.HasTable(&Like{}) {
		if err := db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").CreateTable(&Like{}).Error; err != nil {
			panic(err)
		}
	}
}

func main() {
	defer db.Close()
	add(123, "1.2.3.4", "ua", "t2")
	queryAll()
	add(222, "1.2.3.4", "ua", "t2")
	queryAll()
	add(333, "1.2.3.4", "ua", "t2")
	queryAll()
	querySome(2)
	mode(123)
	queryAll()
}
