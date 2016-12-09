package main

import (
	"fmt"
	"github.com/ddliu/go-httpclient"
)

func main() {
	res, _ := httpclient.WithHeaders(nil).Post("http://bcy.net/public/dologin", map[string]string{
		"email":    "",
		"password": "",
		"remember": "1",
	})
	for _, value := range res.Request.Cookies() {
		fmt.Println(*value)
	}

}
