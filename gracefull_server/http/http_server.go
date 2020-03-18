package main

import (
	"encoding/json"
	"github.com/fvbock/endless"
	"github.com/gorilla/mux"
	"math/rand"
	"net/http"
	"time"
)

type App_rtt_rate_req struct {
	Seqid        int64  `json:"seqid"`
	Uid          int64  `json:"uid"`
	Appid        int32  `json:"appid"`
	Mnc          string `json:"mnc"`
	Mcc          string `json:"mcc"`
	Country_code string `json:"country_code"`
}

type Sample_rate struct {
	Uri    int32         `json:"uri"`
	Rate   int32         `json:"rate"`
	Expire time.Duration `json:"expire"`
}

type App_rtt_rate_res struct {
	Seqid                int64         `json:"seqid"`
	Uid                  int64         `json:"uid"`
	Rates                []Sample_rate `json:"rates"`
	Expire_time          int32         `json:"expire_time"`
	Report_interval_time int32         `json:"report_interval_time"`
}

type Sample_uri_rtt struct {
	Uri        int32 `json:"uri"`
	Avg_rtt_ms int32 `json:"avt_rtt_ms"`
	Rtt_total  int32 `json:"rtt_total"`
}

type App_report_rtt_stat_req struct {
	Seqid int64          `json:"seqid"`
	Uid   int64          `json:"uid"`
	Stat  Sample_uri_rtt `json:"stat"`
}

type App_report_rtt_stat_res struct {
	Seqid    int64 `json:"seqid"`
	Uid      int64 `json:"uid"`
	Res_code int32 `json:"res_code"`
}

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
	SampleConfig = Uri_report_sample_config{
		UriRate:            make(map[int32]Sample_rate),
		CountryCodeUriRate: make(map[string][]Sample_rate),
		Rate:               50,
	}
	max := 10
	for i := 0; i < max; i++ {
		sample := Sample_rate{
			Uri:  int32(rand.Intn(50)),
			Rate: int32(rand.Intn(100)),
		}
		SampleConfig.UriRate[sample.Uri] = sample
	}
}

func get_sample_rate_by_country(seqId, uid int64, appId int32, mnc, mcc, country_code string, res *App_rtt_rate_res) bool {
	v, ok := SampleConfig.CountryCodeUriRate[country_code]
	if ok {
		res.Rates = v
		return true
	}
	return false
}

func get_sample_rate_by_uid(seqId, uid int64, appId int32, mnc, mcc, country_code string, res *App_rtt_rate_res) bool {
	rate := rand.Intn(100)
	if int32(rate) < SampleConfig.Rate {
		return false
	}
	if hit := get_sample_rate_by_country(seqId, uid, appId, mnc, mcc, country_code, res); hit {
		return true
	}
	for _, v := range SampleConfig.UriRate {
		res.Rates = append(res.Rates, v)
	}
	return true
}

func pull_rtt_rate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req App_rtt_rate_req
	_ = json.NewDecoder(r.Body).Decode(&req)

	res := App_rtt_rate_res{
		Seqid: req.Seqid,
		Uid:   req.Uid,
	}
	get_sample_rate_by_uid(req.Seqid, req.Uid, req.Appid, req.Mnc, req.Mcc, req.Country_code, &res)
	json.NewEncoder(w).Encode(&res)
}

func save_sample_rtt_stat(req *App_report_rtt_stat_req) error {
	return nil
}

func report_uri_stat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req App_report_rtt_stat_req
	_ = json.NewDecoder(r.Body).Decode(&req)
	save_sample_rtt_stat(&req)
	res := App_report_rtt_stat_res{
		Seqid:    req.Seqid,
		Uid:      req.Uid,
		Res_code: 200,
	}
	json.NewEncoder(w).Encode(&res)
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/pull_rtt_rate", pull_rtt_rate).Methods("POST")
	router.HandleFunc("/report_uri_stat", report_uri_stat).Methods("POST")
	endless.ListenAndServe("localhost:8900", router)
}
