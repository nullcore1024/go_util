package main

import (
	proto "./platform_app_proto"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/fvbock/endless"
	pb "github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/influxdata/influxdb1-client/v2"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	MyDB          = "app_stats"
	username      = "influx"
	password      = "123456"
	MyMeasurement = "app_uri_rtt_stats"
)

var (
	dbname        = flag.String("database", "appstats", "db database")
	dbuser        = flag.String("dbuser", "root", "db user")
	dbpwd         = flag.String("pwd", "sql123", "user passwd")
	dbAddr        = flag.String("db", "127.0.0.1:3306", "db address and port")
	addr          = flag.String("addr", ":8090", "http service address")
	configFile    = flag.String("conf", "configFile.json", "app uri rate config")
	influxAddr    = flag.String("influxAddr", "http://127.0.0.1:8086", "influx server address")
	DbNotInit     = errors.New("db not init")
	DbErr         = errors.New("db insert fail")
	InfluxNotInit = errors.New("influx not init")
)

type (
	App_rtt_rate_req        = proto.AppRttRateReq
	Sample_rate             = proto.SampleRate
	App_rtt_rate_res        = proto.AppRttRateRes
	Sample_uri_rtt          = proto.SampleUriRtt
	App_report_rtt_stat_req = proto.AppReportRttStatReq
	App_report_rtt_stat_res = proto.AppReportRttStatRes
)

type Uri_report_sample_config struct {
	rwlock               *sync.RWMutex
	UriRate              map[int32]Sample_rate
	CountryCodeUriRate   map[string][]Sample_rate
	Expire               time.Duration `json:"expire_second"`
	Report_interval_time time.Duration `json:"report_interval_second"`
	Rate                 int32         `json: "uid_rate"`
	db                   *gorm.DB
	influxClient         client.Client
}

var SampleConfig Uri_report_sample_config

func init() {
	hook := NewLfsHook("app_stats", time.Hour*1, 48)
	log.AddHook(hook)

	SampleConfig = Uri_report_sample_config{
		rwlock:               new(sync.RWMutex),
		UriRate:              make(map[int32]Sample_rate),
		CountryCodeUriRate:   make(map[string][]Sample_rate),
		Rate:                 50,
		Expire:               time.Hour * 1,
		Report_interval_time: time.Second * 60,
	}
	SampleConfig.Load(*configFile)
}

func (thiz *Uri_report_sample_config) InitInflux() error {
	client, err := connInflux()
	if err != nil {
		return err
	}
	thiz.influxClient = client
	return nil
}

func connInflux() (client.Client, error) {
	cli, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     *influxAddr,
		Username: username,
		Password: password,
	})
	return cli, err
}

func (thiz *Uri_report_sample_config) InitDB() error {
	var err error
	//thiz.db, err = gorm.Open("mysql", fmt.Sprintf("%s:%s@(127.0.0.1:3306)/%s?charset=utf8&parseTime=True&loc=Local", *dbuser, *dbpwd, *dbname))
	thiz.db, err = gorm.Open("mysql", fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local", *dbuser, *dbpwd, *dbAddr, *dbname))
	if err != nil {
		log.Error(err)
		return err
	}

	if !thiz.db.HasTable(&AppStatsItem{}) {
		if err := thiz.db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").CreateTable(&AppStatsItem{}).Error; err != nil {
			log.Error(err)
			return err
		}
	}
	thiz.db.LogMode(true)
	thiz.db.DB().SetMaxIdleConns(10)
	thiz.db.DB().SetMaxOpenConns(100)
	return nil
}

type AppStatsItem struct {
	//Id             int32  `gorm:"type:bigint;primary_key;AUTO_INCREMENT"`
	Seqid          int64  `gorm: "type:bigint"`
	Appid          int32  `gorm: "type:smallint; index:appid_idx"`
	Uid            int64  `gorm: "type:bigint"`
	ClientType     string `gorm: "type:varchar(32); index:client_type_idx"`
	Uri            int32  `gorm: "type:int; index: uri_idx"`
	Version        string `gorm: "type:varcha(32); index: version_idx"`
	AvgRttMs       int32  `gorm: "type:int"`
	ProtoSendTimes int32  `gorm: "type:smallint"`
}

func (thiz *Uri_report_sample_config) Load(file string) error {
	if thiz.rwlock == nil {
		thiz.rwlock = new(sync.RWMutex)
	}
	b, err := ioutil.ReadFile(file)
	if err != nil {
		log.Error("error:", err)
		return err
	}
	json.Unmarshal(b, thiz)
	return nil
}

func (thiz *Uri_report_sample_config) Dump(file string) error {
	str, err := json.Marshal(thiz)
	if err != nil {
		log.Error("marshal failure, err=[%v]", err)
		return err
	}
	err = ioutil.WriteFile(file, []byte(str), 0777)
	if err != nil {
		log.Error("WriteFile failure, err=[%v]", err)
		return err
	}
	return nil
}

func (thiz *Uri_report_sample_config) get_sample_rate_by_country(seqId, uid int64, appId int32, mnc, mcc, country_code string, res *App_rtt_rate_res) bool {
	thiz.rwlock.RLock()
	defer thiz.rwlock.RUnlock()

	v, ok := thiz.CountryCodeUriRate[country_code]
	if ok {
		for _, node := range v {
			res.Rates = append(res.Rates, &node)
		}
		return true
	}
	*res.GlobalExpireSecond = int32(thiz.Expire.Seconds())
	*res.ReportIntervalTime = int32(thiz.Report_interval_time.Seconds())
	return false
}

func (thiz *Uri_report_sample_config) get_sample_rate_by_uid(seqId, uid int64, appId int32, mnc, mcc, country_code string, res *App_rtt_rate_res) bool {
	rate := rand.Intn(100)
	if int32(rate) < thiz.Rate {
		return false
	}

	if hit := thiz.get_sample_rate_by_country(seqId, uid, appId, mnc, mcc, country_code, res); hit {
		return true
	}
	return thiz.get_sample_all_rate(seqId, uid, appId, mnc, mcc, country_code, res)
}

func (thiz *Uri_report_sample_config) get_sample_all_rate(seqId, uid int64, appId int32, mnc, mcc, country_code string, res *App_rtt_rate_res) bool {
	thiz.rwlock.RLock()
	defer thiz.rwlock.RUnlock()

	for _, v := range thiz.UriRate {
		rate := new(Sample_rate)
		*rate = v
		if v.ExpireSecond == nil || *v.ExpireSecond == 0 {
			expire := new(int32)
			*expire = int32(thiz.Expire.Seconds())
			rate.ExpireSecond = expire
		}
		res.Rates = append(res.Rates, rate)
		log.WithFields(log.Fields{
			"uri":    v.GetUri(),
			"rate":   v.GetRate(),
			"expire": rate.GetExpireSecond(),
		}).Info("pull_rtt_rate uriRate")
	}

	*res.GlobalExpireSecond = int32(thiz.Expire.Seconds())
	*res.ReportIntervalTime = int32(thiz.Report_interval_time.Seconds())
	return true
}

//Insert
func WritesPoints(cli client.Client, table string, tags map[string]string, fields map[string]interface{}) error {
	if cli == nil {
		return InfluxNotInit
	}
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  MyDB,
		Precision: "s",
	})
	if err != nil {
		log.Error(err)
		return err
	}

	pt, err := client.NewPoint(
		table,
		tags,
		fields,
		time.Unix(time.Now().Unix(), 0),
	)
	log.Debug("")
	if err != nil {
		log.Error(err)
		return err
	}
	pt.PrecisionString("ms")
	bp.AddPoint(pt)

	if err := cli.Write(bp); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (thiz *Uri_report_sample_config) WritesPoints(table string, req *App_report_rtt_stat_req) error {
	var err error
	for _, v := range req.Sample {
		log.WithFields(log.Fields{
			"Appid":      req.GetAppid(),
			"Version":    req.GetVersion(),
			"clientType": req.GetClientType(),
			"Uri":        v.GetUri(),
			"avgRttMs":   v.GetAvgRttMs(),
			"sendTimes":  v.GetProtoSendTimes(),
		}).Info("save_sample_rtt_stat")

		tags := map[string]string{
			"Appid":      fmt.Sprintf("%d", req.GetAppid()),
			"Version":    req.GetVersion(),
			"clientType": req.GetClientType(),
			"Uri":        fmt.Sprintf("%d", v.GetUri()),
		}
		fields := map[string]interface{}{
			"avgRttMs":  v.GetAvgRttMs(),
			"sendTimes": v.GetProtoSendTimes(),
		}
		log.Debug("dump tags", tags)

		log.WithFields(log.Fields(fields)).Debug("dump fields")

		err = WritesPoints(thiz.influxClient, table, tags, fields)
		if err != nil {
			log.Error("write fail", err)
		}
	}
	return err
}

func (thiz *Uri_report_sample_config) save_db(req *App_report_rtt_stat_req) error {
	if thiz.db == nil {
		return DbNotInit
	}
	var err error
	for _, v := range req.Sample {
		log.WithFields(log.Fields{
			"Seqid":      req.GetSeqid(),
			"Uid":        req.GetUid(),
			"ClientType": req.GetClientType(),
			"Version":    req.GetVersion(),
			"Uri":        v.GetUri(),
			"avgRttMs":   v.GetAvgRttMs(),
			"sendTimes":  v.GetProtoSendTimes(),
		}).Info("save_sample_rtt_stat to db")

		item := &AppStatsItem{
			Seqid:          req.GetSeqid(),
			Appid:          req.GetAppid(),
			Uid:            req.GetUid(),
			ClientType:     req.GetClientType(),
			Version:        req.GetVersion(),
			Uri:            v.GetUri(),
			AvgRttMs:       v.GetAvgRttMs(),
			ProtoSendTimes: v.GetProtoSendTimes(),
		}
		err := thiz.db.Create(item)
		if err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Error("save_sample_rtt_stat to db error")
		}
	}
	return err
}

func (thiz *Uri_report_sample_config) save_sample_rtt_stat(req *App_report_rtt_stat_req) error {
	log.WithFields(log.Fields{
		"Seqid":      req.GetSeqid(),
		"Uid":        req.GetUid(),
		"Appid":      req.GetAppid(),
		"Version":    req.GetVersion(),
		"clientType": req.GetClientType(),
		"size":       len(req.Sample),
	}).Info("save_sample_rtt_stat")

	for _, v := range req.Sample {
		log.WithFields(log.Fields{
			"Seqid":      req.GetSeqid(),
			"Appid":      req.GetAppid(),
			"Uid":        req.GetUid(),
			"Version":    req.GetVersion(),
			"clientType": req.GetClientType(),
			"Uri":        v.GetUri(),
			"avgRttMs":   v.GetAvgRttMs(),
			"sendTimes":  v.GetProtoSendTimes(),
		}).Info("save_sample_rtt_stat")
	}
	thiz.save_db(req)
	thiz.WritesPoints(MyMeasurement, req)
	return nil
}

func pull_rtt_rate_json(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req App_rtt_rate_req
	body, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal(body, &req)
	if err != nil {
		log.Error("unmarshal fail", err)
		w.Write([]byte("format err" + err.Error()))
		return
	}

	log.WithFields(log.Fields{
		"Seqid": req.GetSeqid(),
		"Uid":   req.GetUid(),
	}).Debug("pull_rtt_rate_json")

	res := App_rtt_rate_res{
		Seqid:              req.Seqid,
		Uid:                req.Uid,
		GlobalExpireSecond: new(int32),
		ReportIntervalTime: new(int32),
	}
	SampleConfig.get_sample_rate_by_uid(req.GetSeqid(), req.GetUid(), req.GetAppid(), req.GetMnc(), req.GetMcc(), req.GetCountryCode(), &res)
	log.Debug("res rates size ", len(res.Rates))
	json.NewEncoder(w).Encode(&res)
}

func pull_rtt_rate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/binary")

	var req App_rtt_rate_req
	body, _ := ioutil.ReadAll(r.Body)
	err := pb.Unmarshal(body, &req)
	if err != nil {
		log.Error("pb unmarshal fail", err)
		w.Write([]byte("format err" + err.Error()))
		return
	}

	log.WithFields(log.Fields{
		"Seqid": req.GetSeqid(),
		"Uid":   req.GetUid(),
	}).Debug("pull_rtt_rate")

	res := App_rtt_rate_res{
		Seqid:              req.Seqid,
		Uid:                req.Uid,
		GlobalExpireSecond: new(int32),
		ReportIntervalTime: new(int32),
	}
	SampleConfig.get_sample_rate_by_uid(req.GetSeqid(), req.GetUid(), req.GetAppid(), req.GetMnc(), req.GetMcc(), req.GetCountryCode(), &res)
	log.Debug("res rates size ", len(res.Rates))
	if wdata, err := pb.Marshal(&res); err == nil {
		w.Write([]byte(wdata))
		log.Debug("res data size:", len(wdata))
	} else {
		w.Write([]byte(err.Error()))
		log.Debug("pull_rtt_rate pb mrashal error:", err)
	}
}

func del_uri_rate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vals := r.URL.Query()
	uri, ok := vals["uri"]
	if ok {
		uri_n64, _ := strconv.ParseInt(uri[0], 10, 32)
		var uri_n int32 = int32(uri_n64)

		if _, ok := SampleConfig.UriRate[uri_n]; ok {
			SampleConfig.rwlock.Lock()
			delete(SampleConfig.UriRate, uri_n)
			SampleConfig.rwlock.Unlock()
		}

		SampleConfig.Dump(*configFile)
		log.Debug("del uri:%s rate", uri)
	} else {
		w.Write([]byte("url format err, not find uri"))
		log.Error("url format err, not find uri")
	}
	w.Write([]byte("add uri rate ok"))
}

func mode_uri_rate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vals := r.URL.Query()
	uri, ok := vals["uri"]
	if ok {
		rate, ok := vals["rate"]
		if ok {
			log.Info("add_uri_rate, find uri:%s rate:%s", uri, rate)
		} else {
			log.Error("url format err, find uri:%s but not find rate", uri)
			w.Write([]byte("url format err, find uri but not find rate"))
		}
	} else {
		w.Write([]byte("url format err, not find uri"))
		log.Error("url format err, not find uri")
	}
	w.Write([]byte("add uri rate ok"))
}

func mode_rate_expire(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vals := r.URL.Query()
	str, ok := vals["expire_sec"]
	if ok {
		expire, _ := strconv.ParseInt(str[0], 10, 32)
		expire_second, _ := time.ParseDuration(fmt.Sprintf("%ds", expire))
		SampleConfig.Expire = expire_second
		SampleConfig.Dump(*configFile)
	} else {
		w.Write([]byte("mode_rate_expire url format err, not find expire"))
		log.Error("mode_rate_expire url format err, not find expire")
	}
	w.Write([]byte("mode rate expire ok"))
}

func add_uri_rate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vals := r.URL.Query()
	uri, ok := vals["uri"]
	if ok {
		rate, ok := vals["rate"]
		if ok {
			uri_n64, _ := strconv.ParseInt(uri[0], 10, 32)
			rate_n64, _ := strconv.ParseInt(rate[0], 10, 32)
			var uri_n int32 = int32(uri_n64)
			var rate_n int32 = int32(rate_n64)
			sample := Sample_rate{
				Uri:  &uri_n,
				Rate: &rate_n,
			}
			SampleConfig.rwlock.Lock()
			SampleConfig.UriRate[*sample.Uri] = sample
			SampleConfig.rwlock.Unlock()
			SampleConfig.Dump(*configFile)

			log.Infof("add_uri_rate, find uri:%s rate:%s", uri, rate)
		} else {
			log.Errorf("add_uri_rate url format err, find uri:%s but not find rate", uri)
			w.Write([]byte("add_uri_rate url format err, find uri but not find rate"))
		}
	} else {
		w.Write([]byte("add_uri_rate url format err, not find uri"))
		log.Error("add_uri_rate url format err, not find uri")
	}
	w.Write([]byte("add uri rate ok"))
}

func dump_uri_rate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var (
		Seqid int64 = 0
		Uid   int64 = 0
	)

	res := App_rtt_rate_res{
		Seqid:              &Seqid,
		Uid:                &Uid,
		GlobalExpireSecond: new(int32),
		ReportIntervalTime: new(int32),
	}

	SampleConfig.get_sample_all_rate(res.GetSeqid(), res.GetUid(), 0, "", "", "", &res)
	log.Debug("res rates size ", len(res.Rates))
	json.NewEncoder(w).Encode(&res)
}

func report_uri_stat_json(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req App_report_rtt_stat_req

	body, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal(body, &req)
	if err != nil {
		log.Error("unmarshal fail", err)
		w.Write([]byte("format err" + err.Error()))
		return
	}

	SampleConfig.save_sample_rtt_stat(&req)

	var code int32 = 200
	res := App_report_rtt_stat_res{
		Seqid:   req.Seqid,
		Uid:     req.Uid,
		Appid:   req.Appid,
		Version: req.Version,
		ResCode: &code,
	}
	json.NewEncoder(w).Encode(&res)
}

func report_uri_stat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/binary")
	var req App_report_rtt_stat_req

	body, _ := ioutil.ReadAll(r.Body)
	err := pb.Unmarshal(body, &req)
	if err != nil {
		log.Error("unmarshal fail", err)
		w.Write([]byte("format err" + err.Error()))
		return
	}

	SampleConfig.save_sample_rtt_stat(&req)

	var code int32 = 200
	res := App_report_rtt_stat_res{
		Seqid:   req.Seqid,
		Uid:     req.Uid,
		Appid:   req.Appid,
		Version: req.Version,
		ResCode: &code,
	}

	if wdata, err := pb.Marshal(&res); err == nil {
		w.Write(wdata)
	} else {
		w.Write([]byte(err.Error()))
		log.Error("App_report_rtt_stat_res marshal fail", err)
	}
}

func main() {
	flag.Parse()
	if err := SampleConfig.InitDB(); err != nil {
		panic(err)
	}
	if err := SampleConfig.InitInflux(); err != nil {
		log.Error("init influx failed, err:", err.Error())
	}

	router := mux.NewRouter()
	router.HandleFunc("/add_uri_rate", add_uri_rate).Methods("GET")
	router.HandleFunc("/mode_expire", mode_rate_expire).Methods("GET")
	router.HandleFunc("/del_uri_rate", del_uri_rate).Methods("GET")
	router.HandleFunc("/dump_uri_rate", dump_uri_rate).Methods("GET")
	router.HandleFunc("/mode_uri_rate", mode_uri_rate).Methods("GET")
	router.HandleFunc("/pull_rtt_rate", pull_rtt_rate).Methods("POST")
	router.HandleFunc("/pull_rtt_rate_json", pull_rtt_rate_json).Methods("POST")
	router.HandleFunc("/report_uri_stat", report_uri_stat).Methods("POST")
	router.HandleFunc("/report_uri_stat_json", report_uri_stat_json).Methods("POST")
	endless.ListenAndServe(*addr, router)
}

// 日志钩子(日志拦截，并重定向)
func NewLfsHook(logName string, rotationTime time.Duration, leastDay uint) log.Hook {
	writer, err := rotatelogs.New(
		// 日志文件
		logName+".%Y_%m_%d_%H-%M-%S",

		// 日志周期(默认每86400秒/一天旋转一次)
		rotatelogs.WithRotationTime(rotationTime),

		// 清除历史 (WithMaxAge和WithRotationCount只能选其一)
		//rotatelogs.WithMaxAge(time.Hour*24*7), //默认每7天清除下日志文件
		rotatelogs.WithRotationCount(leastDay), //只保留最近的N个日志文件
	)
	if err != nil {
		panic(err)
	}
	log.SetLevel(log.DebugLevel)
	log.SetReportCaller(true)

	// 可设置按不同level创建不同的文件名
	lfsHook := lfshook.NewHook(lfshook.WriterMap{
		log.DebugLevel: writer,
		log.InfoLevel:  writer,
		log.WarnLevel:  writer,
		log.ErrorLevel: writer,
		log.FatalLevel: writer,
		log.PanicLevel: writer,
	}, &log.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		//PrettyPrint:     true,
	})

	return lfsHook
}
