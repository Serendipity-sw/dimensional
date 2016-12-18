package main

type AppConfig struct {
	LoginUrl           string
	HttpHeaderForLogin map[string]string
	HttpParamsForLogin map[string]string

	HttpHeaderForNormal map[string]string
}

var (
	appConfig *AppConfig = nil
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

	appConfig.HttpParamsForLogin = map[string]string{
		"email":    "",
		"password": "",
		"remember": "1",
	}
}
