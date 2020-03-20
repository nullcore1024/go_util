package main

import (
	proto "./platform_app_proto"
	"encoding/json"
	"flag"
	"github.com/fvbock/endless"
	pb "github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

var addr = flag.String("addr", "127.0.0.1:8080", "http service address")

type (
	App_rtt_rate_req        = proto.AppRttRateReq
	Sample_rate             = proto.SampleRate
	App_rtt_rate_res        = proto.AppRttRateRes
	Sample_uri_rtt          = proto.SampleUriRtt
	App_report_rtt_stat_req = proto.AppReportRttStatReq
	App_report_rtt_stat_res = proto.AppReportRttStatRes
)

//rand.Intn(1000000)
type Uri_report_sample_config struct {
	UriRate              map[int32]Sample_rate
	CountryCodeUriRate   map[string][]Sample_rate
	Expire               time.Duration `json:"expire"`
	Report_interval_time time.Duration `json:"report_interval"`
	Rate                 int32
}

var SampleConfig Uri_report_sample_config

func init() {
	hook := NewLfsHook("app", time.Hour*1, 48)
	log.AddHook(hook)

	SampleConfig = Uri_report_sample_config{
		UriRate:              make(map[int32]Sample_rate),
		CountryCodeUriRate:   make(map[string][]Sample_rate),
		Rate:                 50,
		Expire:               time.Hour * 8,
		Report_interval_time: time.Minute * 5,
	}
	max := 10
	for i := 0; i < max; i++ {
		uri := int32(rand.Intn(50))
		rate := int32(rand.Intn(100))
		sample := Sample_rate{
			Uri:  &uri,
			Rate: &rate,
		}
		SampleConfig.UriRate[*sample.Uri] = sample
	}
}

func get_sample_rate_by_country(seqId, uid int64, appId int32, mnc, mcc, country_code *string, res *App_rtt_rate_res) bool {
	if country_code == nil {
		return false
	}
	v, ok := SampleConfig.CountryCodeUriRate[*country_code]
	if ok {
		for _, node := range v {
			res.Rates = append(res.Rates, &node)
		}
		return true
	}
	var expire int32 = int32(SampleConfig.Expire.Seconds())
	var report int32 = int32(SampleConfig.Report_interval_time.Minutes())
	res.GlobalExpireSecond = &expire
	res.ReportIntervalTime = &report
	return false
}

func get_sample_rate_by_uid(seqId, uid int64, appId int32, mnc, mcc, country_code *string, res *App_rtt_rate_res) bool {
	rate := rand.Intn(100)
	if int32(rate) < SampleConfig.Rate {
		return false
	}
	if hit := get_sample_rate_by_country(seqId, uid, appId, mnc, mcc, country_code, res); hit {
		return true
	}
	for _, v := range SampleConfig.UriRate {
		res.Rates = append(res.Rates, &v)
	}

	var expire int32 = int32(SampleConfig.Expire.Seconds())
	var report int32 = int32(SampleConfig.Report_interval_time.Minutes())
	res.GlobalExpireSecond = &expire
	res.ReportIntervalTime = &report
	return true
}

func pull_rtt_rate_json(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req App_rtt_rate_req
	body, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal(body, &req)
	if err != nil {
		log.Error("unmarshal fail", err)
		w.Write([]byte("format err"))
		return
	}

	log.WithFields(log.Fields{
		"body": string(body),
	}).Debug("pull_rtt_rate body")

	log.WithFields(log.Fields{
		"Seqid": *req.Seqid,
		"Uid":   *req.Uid,
	}).Debug("pull_rtt_rate")

	res := App_rtt_rate_res{
		Seqid: req.Seqid,
		Uid:   req.Uid,
	}
	get_sample_rate_by_uid(*req.Seqid, *req.Uid, *req.Appid, req.Mnc, req.Mcc, req.CountryCode, &res)
	log.Debug("res rates size", len(res.Rates))
	json.NewEncoder(w).Encode(&res)
}

func pull_rtt_rate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/binary")

	var req App_rtt_rate_req
	body, _ := ioutil.ReadAll(r.Body)
	err := pb.Unmarshal(body, &req)
	if err != nil {
		log.Error("unmarshal fail", err)
		w.Write([]byte("format err"))
		return
	}

	log.WithFields(log.Fields{
		"body": string(body),
	}).Debug("pull_rtt_rate body")

	log.WithFields(log.Fields{
		"Seqid": *req.Seqid,
		"Uid":   *req.Uid,
	}).Debug("pull_rtt_rate")

	res := App_rtt_rate_res{
		Seqid: req.Seqid,
		Uid:   req.Uid,
	}
	get_sample_rate_by_uid(*req.Seqid, *req.Uid, *req.Appid, req.Mnc, req.Mcc, req.CountryCode, &res)
	log.Debug("res rates size", len(res.Rates))
	if wdata, err := pb.Marshal(&res); err == nil {
		w.Write(wdata)
	}
}

func save_sample_rtt_stat(req *App_report_rtt_stat_req) error {
	log.WithFields(log.Fields{
		"Seqid": req.Seqid,
		"Uid":   req.Uid,
	}).Debug("save_sample_rtt_stat")
	return nil
}

func report_uri_stat_json(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req App_report_rtt_stat_req

	body, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal(body, &req)
	if err != nil {
		log.Error("unmarshal fail", err)
		w.Write([]byte("format err"))
		return
	}

	save_sample_rtt_stat(&req)

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
		w.Write([]byte("format err"))
		return
	}

	save_sample_rtt_stat(&req)

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
		PrettyPrint:     true,
	})

	return lfsHook
}
