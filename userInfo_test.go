package main

import (
	"fmt"
	"testing"

	"github.com/ddliu/go-httpclient"
	"github.com/smtc/glog"
)

func TestUserInfo(t *testing.T) {
	InitConfig()

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

	coser := &UserInfo{}
	if len(myAccount.FollowingUid) > 1 {
		//coser.Init(myAccount.FollowingUid[1])
		coser.Init("34683")
		//err := coser.AnalysePostCosInfo()
		err := coser.AnalysePostDailyInfo()
		if err != nil {
			fmt.Printf("%v\n", err.Error())
		}
		//		for _, post := range coser.PostCos {
		//			err := post.AnalysePostCosImageInfo(true, true)
		//			if err != nil {
		//				glog.Info("err %s\n", err.Error())
		//			}
		//			err = post.DownloadPostCosImage()
		//			if err != nil {
		//				glog.Info("err %s\n", err.Error())
		//			}
		//		}
		buf, _ = coser.Marshal()
		glog.Info("coser %s\n", buf)
	}
}
