package main

import (
	"fmt"
	"testing"

	"github.com/guotie/config"
	"github.com/smtc/glog"
)

func TestUserInfo(t *testing.T) {
	InitConfig()

	config.ReadCfg(*configFn)
	email = config.GetString("email")
	password = config.GetString("password")

	myAccount := &UserInfo{}
	myAccount.Login(email, password)

	uid, err := getMyUserIndex()
	if err != nil {
		glog.Error("get getMyUserIndex err! err: %s \n", err.Error())
		return
	}

	myAccount.Init(uid)

	glog.Info("cookies %v\n", myAccount.cookies)

	myAccount.AnalyseFollowingInfo()
	buf, _ := myAccount.Marshal()
	glog.Info("%s\n", buf)

	coser := &UserInfo{}
	if len(myAccount.FollowingUid) > 1 {
		coser.Init(myAccount.FollowingUid[1])
		coser.AnalysePostCosInfo()
		for _, post := range coser.PostCos {
			err := post.AnalysePostCosImageInfo(myAccount.cookies)
			if err != nil {
				fmt.Printf("%v\n", err.Error())
			}
			break
		}
		buf, _ = coser.Marshal()
		glog.Info("%s\n", buf)
	}
}
