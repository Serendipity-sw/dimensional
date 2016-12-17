package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/ddliu/go-httpclient"
	"github.com/smtc/glog"
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

// 获取自己的uid
func getMyUserIndex() (uid string, errRet error) {
	uid = ""
	userIndexPage, err := httpclient.WithCookie(cookies...).Get("http://bcy.net/home/user/index", nil)
	if err != nil {
		glog.Error("getMyUserIndex http get index page err! err: %s \n", err.Error())
		return uid, errors.New(fmt.Sprintf("getMyUserIndex http get index page err! err: %s \n", err.Error()))
	}
	defer userIndexPage.Body.Close()

	doc, err := goquery.NewDocumentFromReader(userIndexPage.Body)
	if err != nil {
		glog.Error("getMyUserIndex NewDocumentFromReader read error! err: %s \n", err.Error())
		return uid, errors.New(fmt.Sprintf("getMyUserIndex NewDocumentFromReader read error! err: %s \n", err.Error()))
	}

	uidUrl, bo := doc.Find(".posr._avatar--xl.l-left.mr15").Find("._avatar._avatar--xl._avatar--user").Attr("href")
	if bo {
		uidArray := strings.Split(uidUrl, "/u/")

		if len(uidArray) != 2 {
			glog.Error("getMyUserIndex err! uidUrl: %s \n", uidUrl)
			return uid, errors.New(fmt.Sprintf("getMyUserIndex err! uidUrl: %s \n", uidUrl))
		}

		uid = uidArray[1]
	}
	return uid, nil
}

// 获取提供页面 goquery.Document 中的uid
func getUserIndexByDetailPageDoc(detailPage *goquery.Document) (uid string, errRet error) {
	uid = ""

	if detailPage == nil {
		glog.Error("getUserIndexByDetailPageDoc detailPage error! err: detailPage == nil \n")
		return uid, errors.New(fmt.Sprintf("getUserIndexByDetailPageDoc detailPage error! err: detailPage == nil \n"))
	}

	uidUrl, bo := detailPage.Find(".posr._avatar--xl.center-block.mb10").Find("._avatar._avatar--xl._avatar--user").Attr("href")
	if bo {
		uidArray := strings.Split(uidUrl, "/u/")

		if len(uidArray) != 2 {
			glog.Error("getUserIndexByDetailPageDoc err! uidUrl: %s \n", uidUrl)
			return uid, errors.New(fmt.Sprintf("getUserIndexByDetailPageDoc err! uidUrl: %s \n", uidUrl))
		}

		uid = uidArray[1]
	}
	return uid, nil
}

/**
查询关注列表
创建人:邵炜
创建时间:2016年12月13日12:00:49
*/
func attentionPageProcess() {
	uid, err := getMyUserIndex()
	if err != nil {
		glog.Error("attentionPageProcess http get getMyUserIndex err! err: %s \n", err.Error())
		return
	}

	glog.Info("My Uid: %s\n", uid)

	attentionPage, err := httpclient.WithCookie(cookies...).Get(fmt.Sprintf("http://bcy.net/u/%s/following", uid), nil)
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
		analysisAllFollowUser(uid, lastPageNumber)
	}
}

/**
解析所有的关注用户
创建人:邵炜
创建时间:2016年12月13日16:32:27
输入参数:总页数
*/
func analysisAllFollowUser(uid string, pagerNumber int) {
	httpUrl := ""
	for {
		if pagerNumber <= 0 {
			break
		}
		httpUrl = fmt.Sprintf(fmt.Sprintf("http://bcy.net/u/%s/following?&p=%d", uid), pagerNumber)
		attentionPage, err := httpclient.WithCookie(cookies...).Get(httpUrl, nil)
		if err != nil {
			glog.Error("analysisFollowUser send http error! pagerNumber: %d err: %s \n", pagerNumber, err.Error())
			continue
		}
		doc, err := goquery.NewDocumentFromReader(attentionPage.Body)
		attentionPage.Body.Close()
		if err != nil {
			glog.Error("analysisFollowUser anaysis followUser Page err! err: %s \n", err.Error())
			continue
		}
		doc.Find(".l-newFanBoxList.l-clearfix").Find("li").Each(func(indexNumber int, nodeObj *goquery.Selection) {
			hrefStr, bo := nodeObj.Find("a").First().Attr("href")
			if bo {
				analysisFollowUser(hrefStr)
			}
		})
		pagerNumber--
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
	glog.Info("userPageHomeUrl %s\n", userPageHomeUrl)
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
		if len(pageNumberArray) != 2 {
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
		glog.Info("httpUrl %s\n", httpUrl)
		attentionPage, err := httpclient.WithCookie(cookies...).Get(httpUrl, nil)
		if err != nil {
			glog.Error("analysisFollowUserCOSEveryPage send http err! httpUrl: %s err: %s \n", httpUrl, err.Error())
			return
		}
		doc, err := goquery.NewDocumentFromReader(attentionPage.Body)
		attentionPage.Body.Close()
		if err != nil {
			glog.Error("analysisFollowUserCOSEveryPage analysis documenterr! sendHttp: %s  err: %s \n", httpUrl, err.Error())
			return
		}
		doc.Find(".l-grid__inner li").Each(func(indexNumber int, nodeObj *goquery.Selection) {
			hrefUrlPath, bo := nodeObj.Find("a").First().Attr("href")
			if bo {
				userSendPostsProcess(fmt.Sprintf("http://bcy.net%s", hrefUrlPath))
			}
		})
		pageNumber--
	}
}

func getCoserDir(uid string, userName string) string {
	userName = trimInvalidChar(userName)

	return fmt.Sprintf("%s-%s", uid, userName)
}

/**
用户发的每一页cos页面图片下载
创建人:邵炜
创建时间:2016年12月14日11:18:10
输入参数:用户COS帖子页面地址
*/
func userSendPostsProcess(userCOSPostsUrlPath string) {
	attentionPage, err := httpclient.WithCookie(cookies...).Get(userCOSPostsUrlPath, nil)
	if err != nil {
		glog.Error("userSendPostsProcess send http err! httpUrl: %s err: %s \n", userCOSPostsUrlPath, err.Error())
		return
	}
	doc, err := goquery.NewDocumentFromReader(attentionPage.Body)
	attentionPage.Body.Close()
	if err != nil {
		glog.Error("userSendPostsProcess read body err! userCOSPostsUrlPath: %s err: %s \n", userCOSPostsUrlPath, err.Error())
		return
	}

	userName := doc.Find(".js-userTpl").Find(".fz14.blue1").First().Text()

	uid, err := getUserIndexByDetailPageDoc(doc)
	if err != nil {
		glog.Error("userSendPostsProcess getUserIndexByDetailPageDoc! err: %s \n", err.Error())
		return
	}

	//coserDirName := getCoserDir(uid, userName)
	//mkdirPath := fmt.Sprintf("./cos/%s", coserDirName)

	mkdirPath := getCoserExistDirPath(uid, userName, true)

	err = os.MkdirAll(mkdirPath, 0777)
	if err != nil {
		glog.Error("userSendPostsProcess create file err! mkdirPath: %s err: %s \n", mkdirPath, err.Error())
		return
	}
	mkdirPathFileName := doc.Find(".js-post-title").First().Text()
	mkdirPathFileName = strings.TrimSpace(mkdirPathFileName)
	mkdirPathFileNamePath := fmt.Sprintf("%s/%s", mkdirPath, trimInvalidChar(mkdirPathFileName))
	bo, _ := pathExists(mkdirPathFileNamePath)
	if bo {
		return
	}
	err = os.MkdirAll(mkdirPathFileNamePath, 0777)
	if err != nil {
		glog.Error("userSendPostsProcess userCOSPosts exsis! mkdirPathFileNamePath: %s err: %s \n", mkdirPathFileNamePath, err.Error())
		return
	}
	doc.Find(".detail_std.detail_clickable").Each(func(indexNumber int, nodeObj *goquery.Selection) {
		COSPictureUrlStr, bo := nodeObj.Attr("src")
		if bo {
			COSPictureUrlStr = COSPictureUrlStr[:strings.LastIndex(COSPictureUrlStr, "/")]
			pictureDown(COSPictureUrlStr, mkdirPathFileNamePath)
		}
	})
}

/**
图片下载
创建人:邵炜
创建时间:2016年12月14日11:46:36
输入参数:图片地址 初始目录地址
*/
func pictureDown(urlPathStr, mkdirPath string) {
	res, err := http.Get(urlPathStr)
	if err != nil {
		glog.Error("pictureDown send http err! urlPathStr: %s err: %s \n", urlPathStr, err.Error())
		return
	}
	urlPathArray := strings.Split(urlPathStr, "/")
	pictureFileName := urlPathArray[len(urlPathArray)-1]
	picuteCreatePathStr := fmt.Sprintf("%s/%s", mkdirPath, pictureFileName)
	file, err := os.Create(picuteCreatePathStr)
	if err != nil {
		glog.Error("pictureDown picute create err! filePath: %s err: %s \n", picuteCreatePathStr, err.Error())
		return
	}
	io.Copy(file, res.Body)
	defer res.Body.Close()
	file.Close()
}

/**
判断当前文件夹是否存在
创建人:邵炜
创建时间:2016年12月14日11:23:01
*/
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

/**
屏蔽非法字符
创建人:邵炜
创建时间:2016年12月14日14:24:05
*/
func trimInvalidChar(name string) string {
	var invalid []string = []string{"\\", "/", ":", "?", "*", "\"", "<", "|", ">"}
	for _, value := range invalid {
		name = strings.Replace(name, value, "", -1)
	}
	return name
}
