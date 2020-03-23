package main

import (
	proto "./platform_app_proto"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fvbock/endless"
	pb "github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
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

var addr = flag.String("addr", ":8080", "http service address")
var configFile = flag.String("conf", "configFile.json", "app uri rate config")

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
	Report_interval_time time.Duration `json:"report_interval_minute"`
	Rate                 int32         `json: "uid_rate"`
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
		Report_interval_time: time.Minute * 5,
	}
	SampleConfig.Load(*configFile)
	/*
		rand.Seed(time.Now().UnixNano())
		setUri := []int32{19, 23, 1001, 4001, 5005, 5007}
		for i, _ := range setUri {
			rate := int32(rand.Intn(100))
			sample := Sample_rate{
				Uri:  &setUri[i],
				Rate: &rate,
			}
			SampleConfig.UriRate[*sample.Uri] = sample
		}
	*/

	//SampleConfig.Dump(*configFile)
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
	*res.ReportIntervalTime = int32(thiz.Report_interval_time.Minutes())
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
	*res.ReportIntervalTime = int32(thiz.Report_interval_time.Minutes())
	return true
}

func (thiz *Uri_report_sample_config) save_sample_rtt_stat(req *App_report_rtt_stat_req) error {
	log.WithFields(log.Fields{
		"Seqid": req.Seqid,
		"Uid":   req.Uid,
	}).Debug("save_sample_rtt_stat")

	for _, v := range req.Sample {
		log.WithFields(log.Fields{
			"Seqid":     req.Seqid,
			"Uid":       req.Uid,
			"Uri":       v.Uri,
			"avgRttMs":  v.AvgRttMs,
			"sendTimes": v.ProtoSendTimes,
		}).Debug("save_sample_rtt_stat")
	}
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

		var rr App_rtt_rate_res
		err := pb.Unmarshal(wdata, &rr)
		if err != nil {
			log.Error("pb xxxx unmarshal fail", err)
		} else {
			log.Info("pb xxxx unmarshal ok")
		}

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
		Seqid: &Seqid,
		Uid:   &Uid,
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

	log.WithFields(log.Fields{
		"Seqid": req.Seqid,
		"Uid":   req.Uid,
	}).Debug("report_uri_stat")

	var code int32 = 200
	res := App_report_rtt_stat_res{
		Seqid:   req.Seqid,
		Uid:     req.Uid,
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

	log.WithFields(log.Fields{
		"Seqid": req.Seqid,
		"Uid":   req.Uid,
	}).Debug("report_uri_stat")

	var code int32 = 200
	res := App_report_rtt_stat_res{
		Seqid:   req.Seqid,
		Uid:     req.Uid,
		ResCode: &code,
	}

	if wdata, err := pb.Marshal(&res); err == nil {
		w.Write(wdata)
	}
}

func main() {
	flag.Parse()

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
