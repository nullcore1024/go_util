package main

import (
	"fmt"
	_ "github.com/Go-SQL-Driver/MySQL"
	"github.com/astaxie/beego/orm" //对应的beego/orm库 可以通过 go get来获取到本地GOPATH路径下
	"time"
)

//与数据库学生表映射的结构体
type studentinfo struct {
	Id          int    `pk:"auto"`
	Stuname     string `orm:"size(20)"`
	Stuidentify string `orm:"size(30)"`
	Stubirth    time.Time
	Stuclass    string `orm:"size(30)"`
	Stumajor    string `orm:"size(30)"`
}

//数据库连对象需要的信息
var (
	dbuser string = "root"
	dbpwd  string = "691214"
	dbname string = "gosql"
)

//初始化orm
func init() {
	conn := dbuser + ":" + dbpwd + "@/" + dbname + "?charset=utf8" //组合成连接串
	orm.RegisterModel(new(studentinfo))                            //注册表studentinfo 如果没有会自动创建
	orm.RegisterDriver("mysql", orm.DR_MySQL)                      //注册mysql驱动
	orm.RegisterDataBase("default", "mysql", conn)                 //设置conn中的数据库为默认使用数据库
	orm.RunSyncdb("default", false, false)                         //后一个使用true会带上很多打印信息，数据库操作和建表操作的；第二个为true代表强制创建表
}

func main() {
	orm.Debug = true
	dbObj := orm.NewOrm()

	var stuPtr *studentinfo = new(studentinfo)
	stuPtr.Stuname = "xiaom"
	stuPtr.Stubirth = time.Now()
	stuPtr.Stuclass = "一年级1班"
	stuPtr.Stuidentify = "1234"
	stuPtr.Stumajor = "计算机"

	tm := time.Now()
	var studentus = []studentinfo{
		{Stuname: "xd", Stuidentify: "1235", Stubirth: tm, Stuclass: "一年级2班", Stumajor: "数据库"},
		{Stuname: "xx", Stuidentify: "1236", Stubirth: tm, Stuclass: "一年级3班", Stumajor: "网络"},
		{Stuname: "xn", Stuidentify: "1237", Stubirth: tm, Stuclass: "一年级4班", Stumajor: "C语言"},
		{Stuname: "xb", Stuidentify: "1238", Stubirth: tm, Stuclass: "一年级5班", Stumajor: "JAVA"},
		{Stuname: "xq", Stuidentify: "1239", Stubirth: tm, Stuclass: "一年级6班", Stumajor: "C++"},
	}

	var err error
	_, err = dbObj.Insert(stuPtr) //单条记录插入
	if err != nil {
		fmt.Printf("插入学生:%s信息出错。\n", stuPtr.Stuname)
	} else {
		fmt.Printf("插入学生:%s信息成功。\n", stuPtr.Stuname)
	}

	var num int64
	num, err = dbObj.InsertMulti(5, studentus) //多条记录插入
	if err != nil {
		fmt.Printf("插入%d个学生信息错误,%d个学会信息成功。\n", 5-num, num)
	} else {
		fmt.Printf("成功插入%d学生信息。\n", num)
	}

	studentR := new(studentinfo) //记录读取，需要指定主键
	studentR.Id = 6
	err = dbObj.Read(studentR)
	if err != nil {
		fmt.Printf("读取ID:%d的学生信息失败", studentR.Id)
	} else {
		fmt.Printf("ID:%d的学生个人信息为：\n", studentR.Id)
		fmt.Println(studentR)
	}

	studentU := new(studentinfo)
	studentU.Id = 5
	studentU.Stumajor = "管理科学与工程"
	_, err = dbObj.Update(studentU, "Stumajor") //记录更新
	if err != nil {
		fmt.Printf("更新ID:%d的学生信息失败。", studentU.Id)
	} else {
		fmt.Printf("更新ID:%d的学生信息成功。", studentU.Id)
	}

	///删除 dbObj.Delete(studentU)//
}
