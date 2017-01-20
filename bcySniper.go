package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ddliu/go-httpclient"
	"github.com/smtc/glog"
)

type ConfigInFile struct {
	Email    string `json:Email`
	Password string `json:Password`
}

type AppConfig struct {
	LoginUrl           string
	HttpHeaderForLogin map[string]string
	HttpParamsForLogin map[string]string

	HttpHeaderForNormal map[string]string
}

var (
	appConfig      *AppConfig = nil
	debugFlag                 = flag.Bool("d", false, "debug mode")
	configFilePath            = flag.String("f", "config.json", "config file")
)

func InitConfig() {
	appConfig = &AppConfig{}

	appConfig.LoginUrl = "http://bcy.net/public/dologin"

	appConfig.HttpHeaderForLogin = map[string]string{
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Encoding":           "gzip, deflate",
		"Accept-Language":           "zh-CN,zh;q=0.8",
		"Cache-Control":             "max-age=0",
		"Connection":                "keep-alive",
		"Content-Type":              "application/x-www-form-urlencoded",
		"Host":                      "bcy.net",
		"Origin":                    "http://bcy.net",
		"Referer":                   "http://bcy.net/login",
		"Upgrade-Insecure-Requests": "1",
		"User-Agent":                "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/55.0.2883.75 Safari/537.36",
	}

	appConfig.HttpHeaderForNormal = map[string]string{
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Encoding":           "gzip, deflate",
		"Accept-Language":           "zh-CN,zh;q=0.8",
		"Cache-Control":             "max-age=0",
		"Connection":                "keep-alive",
		"Content-Type":              "application/x-www-form-urlencoded",
		"Host":                      "bcy.net",
		"Origin":                    "http://bcy.net",
		"Referer":                   "http://bcy.net",
		"Upgrade-Insecure-Requests": "1",
		"User-Agent":                "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/55.0.2883.75 Safari/537.36",
	}

	config, err := GetFileConfig(*configFilePath)
	if err != nil {
		glog.Error("read config file fail %s\n", configFilePath)
		panic(err)
	}

	appConfig.HttpParamsForLogin = map[string]string{
		"email":    config.Email,
		"password": config.Password,
		"remember": "1",
	}
}

func GetFileConfig(filePath string) (*ConfigInFile, error) {
	fi, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer fi.Close()
	fd, err := ioutil.ReadAll(fi)

	config := &ConfigInFile{}
	err = json.Unmarshal(fd, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func test() {
	res, err := httpclient.WithHeaders(appConfig.HttpHeaderForLogin).Post(appConfig.LoginUrl, appConfig.HttpParamsForLogin)
	//bodyString, err := res.ToString()

	//fmt.Println(res.StatusCode, bodyString, err)

	fmt.Printf("%v\n", res.Cookies())

	//res2, err := httpclient.WithHeaders(appConfig.HttpHeaderForNormal).WithCookie(res.Cookies()...).Get("http://bcy.net/coser/detail/8879/381112", nil)
	//res2, err := httpclient.WithHeaders(appConfig.HttpHeaderForNormal).WithCookie(res.Cookies()...).Get("http://bcy.net/coser/detail/62333/942591", nil)
	//res2, err := httpclient.WithHeaders(appConfig.HttpHeaderForNormal).WithCookie(res.Cookies()...).Get("http://bcy.net/home/user/index", nil)
	res2, err := httpclient.WithHeaders(appConfig.HttpHeaderForNormal).Get("http://bcy.net/home/user/index", nil)
	bodyString, err := res2.ToString()
	_ = res2
	_ = err
	_ = bodyString

	fmt.Printf("%v\n", httpclient.Cookies("http://bcy.net"))

	fmt.Println(res2.StatusCode, bodyString, err)
}

func mainProcess() {
	email := appConfig.HttpParamsForLogin["email"]
	password := appConfig.HttpParamsForLogin["password"]

	myAccount := &UserInfo{}
	myAccount.Login(email, password)

	glog.Info("cookies %v\n", httpclient.Cookies("http://bcy.net"))

	uid, err := getMyUserIndex()
	if err != nil {
		glog.Error("get getMyUserIndex err! err: %s \n", err.Error())
		return
	}

	myAccount.Init(uid)

	myAccount.AnalyseFollowingInfo()
	buf, _ := myAccount.Marshal()
	glog.Info("%s\n", buf)

	for _, followingUid := range myAccount.FollowingUid {
		glog.Info("mainProcess AnalysePostCosInfo followingUid %s\n", followingUid)
		coser := &UserInfo{}
		coser.Init(followingUid)

		err := coser.AnalysePostCosInfo()
		if err != nil {
			glog.Error("%s\n", err.Error())
			continue
		}

		err = coser.AnalysePostDailyInfo()
		if err != nil {
			glog.Error("%s\n", err.Error())
			continue
		}

		// 保存用户解析信息的 json 文件
		coser.SaveCosInfo()

		for _, post := range coser.PostCos {
			glog.Info("mainProcess AnalysePostCosImageInfo post %s - %s\n", post.Url, post.Title)
			err := post.AnalysePostCosImageInfo(true, true)
			if err != nil {
				glog.Error("err %s\n", err.Error())
				continue
			}
			err = post.DownloadPostCosImage()
			if err != nil {
				glog.Error("err %s\n", err.Error())
				continue
			}
			post.SavePostCosImageInfo()
			if err != nil {
				glog.Error("err %s\n", err.Error())
				continue
			}
			if post.IsIngore() {
				post.ClearPostCosImage()
			}
		}

		for _, post := range coser.PostDaily {
			glog.Info("mainProcess AnalysePostDailyImageInfo post %s - %s\n", post.Url, post.Title)
			err := post.AnalysePostCosImageInfo(true, true)
			if err != nil {
				glog.Error("err %s\n", err.Error())
				continue
			}
			err = post.DownloadPostCosImage()
			if err != nil {
				glog.Error("err %s\n", err.Error())
				continue
			}
			post.SavePostCosImageInfo()
			if err != nil {
				glog.Error("err %s\n", err.Error())
				continue
			}
			if post.IsIngore() {
				post.ClearPostCosImage()
			}
		}

		buf, _ = coser.Marshal()
		glog.Info("coser %s\n", buf)
	}
}

func main() {
	flag.Parse()
	logInit(*debugFlag)

	InitConfig()

	mainProcess()
}
