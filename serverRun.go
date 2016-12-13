package main

import (
	"fmt"
	"github.com/ddliu/go-httpclient"
	"github.com/smtc/glog"
	"goquery"
	"net/http"
	"strconv"
	"strings"
)

var cookies []*http.Cookie

/**
服务运行
创建人:邵炜
创建时间:2016年12月13日12:00:17
*/
func serverRun() {
	res, err := httpclient.WithHeaders(nil).Post("http://bcy.net/public/dologin", map[string]string{
		"email":    email,
		"password": password,
		"remember": "1",
	})
	if err != nil {
		glog.Error("serverRun http login err! err: %s \n", err.Error())
		return
	}
	cookies = res.Cookies()
	attentionPageProcess()
}

/**
查询关注列表
创建人:邵炜
创建时间:2016年12月13日12:00:49
*/
func attentionPageProcess() {
	attentionPage, err := httpclient.WithCookie(cookies...).Get("http://bcy.net/u/1496212/following", nil)
	if err != nil {
		glog.Error("attentionPageProcess http get follow page err! err: %s \n", err.Error())
		return
	}
	defer attentionPage.Body.Close()
	doc, err := goquery.NewDocumentFromReader(attentionPage.Body)
	if err != nil {
		glog.Error("attentionPageProcess NewDocumentFromReader read error! err: %s \n", err.Error())
		return
	}
	lastPageNumber := 0
	lastPager, bo := doc.Find(".pager li").Last().Find("a").Attr("href")
	if bo {
		numberArray := strings.Split(lastPager, "p=")
		if len(numberArray) != 2 {
			glog.Error("attentionPageProcess pager number analysis err! numberPager: %s \n", lastPager)
			return
		}
		lastPageNumber, err = strconv.Atoi(numberArray[1])
		if err != nil {
			glog.Error("attentionPageProcess pager number convert string to int err! numberPager: %s \n", numberArray[1])
			return
		}
		analysisAllFollowUser(lastPageNumber)
	}
}

/**
解析所有的关注用户
创建人:邵炜
创建时间:2016年12月13日16:32:27
输入参数:总页数
*/
func analysisAllFollowUser(pagerNumber int) {
	httpUrl := ""
	for {
		if pagerNumber <= 0 {
			break
		}
		httpUrl = fmt.Sprintf("http://bcy.net/u/1496212/following?&p=%d", pagerNumber)
		attentionPage, err := httpclient.WithCookie(cookies...).Get(httpUrl, nil)
		if err != nil {
			glog.Error("analysisFollowUser send http error! pagerNumber: %d err: %s \n", pagerNumber, err.Error())
			return
		}
		doc, err := goquery.NewDocumentFromReader(attentionPage.Body)
		attentionPage.Body.Close()
		if err != nil {
			glog.Error("analysisFollowUser anaysis followUser Page err! err: %s \n", err.Error())
			return
		}
		doc.Find(".l-newFanBoxList.l-clearfix").Find("li").Each(func(indexNumber int, nodeObj *goquery.Selection) {
			hrefStr, bo := nodeObj.Find("a").First().Attr("href")
			if bo {
				analysisFollowUser(hrefStr)
			}
		})
	}
}

/**
解析关注用户
创建人:邵炜
创建时间:2016年12月13日16:48:32
输入参数:关注用户url截断地址
*/
func analysisFollowUser(urlPathStr string) {
	userPageHomeUrl := fmt.Sprintf("http://bcy.net%s/post/Cos", urlPathStr)
	attentionPage, err := httpclient.WithCookie(cookies...).Get(userPageHomeUrl, nil)
	if err != nil {
		glog.Error("analysisFollowUser send http err! url: %s err: %s \n", userPageHomeUrl, err.Error())
		return
	}
	doc, err := goquery.NewDocumentFromReader(attentionPage.Body)
	attentionPage.Body.Close()
	if err != nil {
		glog.Error("analysisFollowUser page analysis err! err: %s \n", err.Error())
		return
	}
	pageNumberANode, bo := doc.Find(".l-home-follow-pager li").Last().Find("a").Attr("href")
	if bo {
		pageNumberArray := strings.Split(pageNumberANode, "p=")
		if pageNumberArray != 2 {
			glog.Error("analysisFollowUser pageNumber analysis err! pageNumberANode: %s \n", pageNumberANode)
		}
		pageNumber, err := strconv.Atoi(pageNumberArray[1])
		if err != nil {
			glog.Error("analysisFollowUser pageNumber can't convert string to int! pageNumberArraye: %v err: %s \n", pageNumberArray, err.Error())
			return
		}
		analysisFollowUserCOSEveryPage(userPageHomeUrl, pageNumber)
	}
}

/**
解析关注用户的每一页COS
创建人:邵炜
创建时间:2016年12月13日17:02:28
输入参数:关注用户COS页面地址,总页数
*/
func analysisFollowUserCOSEveryPage(followUserCOSPageUrl string, pageNumber int) {
	for {
		if pageNumber <= 0 {
			break
		}
		httpUrl := fmt.Sprintf("%s?p=%d", followUserCOSPageUrl, pageNumber)
		attentionPage, err := httpclient.WithCookie(cookies...).Get(httpUrl, nil)
		if err != nil {
			glog.Error("analysisFollowUserCOSEveryPage send http err! httpUrl: %s err: %s \n", httpUrl, err.Error())
			return
		}
		doc, err := goquery.NewDocumentFromReader(attentionPage)
		if err != nil {
			glog.Error("analysisFollowUserCOSEveryPage analysis documenterr! sendHttp: %s  err: %s \n", httpUrl, err.Error())
			return
		}
		doc.Find(".l-grid__inner li").Each(func() {

		})
	}
}
