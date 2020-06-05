package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	geoURL    = "http://ip-api.com/batch"
	cacheFile = "cache.json"
)

var (
	cache map[string]GEOResponse

	rotatelogNum = flag.Int("rotatelogNum", 3, "save rotatelog num, default:3")
	loginFile    = flag.String("loginFile", "login.ip", "daily login ip, default:login.ip")
	postSlice    = flag.Int("batchSlice", 100, "post slice num, default:100")
)

func init() {
	cache = make(map[string]GEOResponse)
	loadFromCache(cacheFile, cache)
}

func loadFromCache(file string, conf map[string]GEOResponse) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Error(err)
		return
	}
	json.Unmarshal(content, &conf)

	for k, v := range conf {
		if v.Status != "success" {
			fmt.Println("err", k)
		}
	}
}

func saveCache(file string, conf map[string]GEOResponse) {
	body, _ := json.Marshal(&conf)
	err := ioutil.WriteFile(file, body, 0777)
	if err != nil {
		log.Error(err)
		return
	}
}

type GEOResponse struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	Isp         string  `json:"isp"`
	Org         string  `json:"org"`
	As          string  `json:"as"`
	Query       string  `json:"query"`
}

func Append(str []string, s ...string) string {
	str = append(str, s...)
	body, _ := json.Marshal(&str)
	return string(body)
}

func buildRequestStr(index int, s []string) []GEOResponse {
	body, _ := json.Marshal(&s)

	log.Infof("index:%d, body:%s", index, string(body))

	return buildRequestJson(string(body))
}

func buildRequest(s ...string) []GEOResponse {
	var b []string
	body := Append(b, s...)

	return buildRequestJson(body)
}

func buildRequestJson(body string) []GEOResponse {
	contentType := "application/json;charset=utf-8"

	log.Info("xxxx body:", string(body))

	resp, err := http.Post(geoURL, contentType, bytes.NewBuffer([]byte(body)))
	if err != nil {
		log.Println("Post failed:", err)
		return nil
	}

	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Read failed:", err)
		return nil
	}

	var res []GEOResponse
	json.Unmarshal(content, &res)
	//log.Println("content:", string(content))
	log.Println("content:", len(res))
	return res
}

func saveGEOToCache(res []GEOResponse, cache map[string]GEOResponse) int {
	save := 0
	for i, _ := range res {
		if _, ok := cache[res[i].Query]; !ok && res[i].Status == "success" {
			cache[res[i].Query] = res[i]
			save++
		}
	}
	saveCache(cacheFile, cache)
	return save
}

func readLines(file string) []string {
	fi, err := os.Open(file)
	if err != nil {
		fmt.Errorf("Error: %s\n", err)
		return nil
	}
	defer fi.Close()

	var lines []string

	br := bufio.NewReader(fi)
	for {
		a, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		lines = append(lines, string(a))
	}
	return lines
}

func getNoCacheIp(lines []string, cache map[string]GEOResponse) []string {
	log.Infof("cache size:%d", len(cache))

	var res []string

	for i, _ := range lines {
		if _, ok := cache[lines[i]]; !ok {
			res = append(res, lines[i])
		}
	}
	log.Infof("new size:%d, need post size:%d, cache size:%d", len(lines), len(cache), len(res))
	return res
}

func run(file string) {
	lines := getNoCacheIp(readLines(file), cache)

	t := time.NewTicker(2 * time.Second)
	defer t.Stop()

	batchSlice := *postSlice
	sz := len(lines)
	loop := (len(lines) + batchSlice - 1) / batchSlice

	end := 0

	for i := 0; i < loop; i++ {
		if (i+1)*batchSlice < sz {
			end = (i + 1) * batchSlice
		} else {
			end = sz
		}
		res := buildRequestStr(i, lines[i*batchSlice:end])
		if res == nil {
			i--
			time.Sleep(time.Second * 10)
			log.Warn("fail")
			continue
		}
		ret := saveGEOToCache(res, cache)
		log.Infof("end:%d，save:%d, res:%v", end, ret, res)
	}
}

func main() {
	flag.Parse()

	hook := NewLfsHook("ip", time.Hour*1, uint(*rotatelogNum))
	log.AddHook(hook)

	loadFromCache(cacheFile, cache)

	run(*loginFile)
	log.Info("done finish")
}
