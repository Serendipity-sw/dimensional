package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/ddliu/go-httpclient"
	"github.com/smtc/glog"
)

// 存储用户信息的结构，用于coser
type UserInfo struct {
	Id           string `json:"Id"`           // 用户ID
	UserName     string `json:"UserName"`     // 处理后的用户名 eg. 去掉非法字符，"-" 替换为 "="
	UserNamePure string `json:"UserNamePure"` // 原始用户名

	cookies []*http.Cookie // 存储登陆cookies

	UrlUserDetail string `json:"UrlUserDetail"` // 用户主页

	PathStorage string `json:"PathStorage"` // 用户资源存储路径

	PostCos   []*PostInfo `json:"PostCos"`
	PostDaily []*PostInfo `json:"PostDaily"`

	FollowingUid []string `json:"FollowingUid"` // 本用户关注的用户ID
}

// 作品Detail信息
type PostInfo struct {
	Url       string `json:"Url"`       // 作品地址
	Title     string `json:"Title"`     // 原始标题
	TitlePure string `json:"TitlePure"` // 原始标题
	Id        string `json:"Id"`        // 作品ID

	PathStorage string `json:"PathStorage"` // 资源存储路径

	Image []string `json:"Image"` // 作品页面图片列表
}

// 作品图片信息
//type ImageInfo struct {
//	Url string `json:"Url"` // 地址
//}

func (this *UserInfo) Login(loginUserName string, password string) error {
	//res, err := httpclient.WithHeaders(nil).Post("http://bcy.net/public/dologin", map[string]string{
	res, err := httpclient.WithHeaders(appConfig.HttpHeaderForLogin).Post(appConfig.LoginUrl, map[string]string{
		"email":    loginUserName,
		"password": password,
		"remember": "1",
	})
	if err != nil {
		glog.Error("serverRun http login err! err: %s \n", err.Error())
		return err
	}
	glog.Info("Login cookies %v\n", res.Cookies())
	this.cookies = res.Cookies()
	return err
}

// 根据id初始化用户结构
// 1. 获取用户名
// 2. 获取用户详细页面url
// 3. 获取用户资源存储路径（用户改名会自动修正目录名，但是如果目录不存在不会创建文件夹）
func (this *UserInfo) Init(id string) error {
	userCOSPostsUrlPath := fmt.Sprintf("http://bcy.net/u/%s", id)

	attentionPage, err := httpclient.WithCookie(cookies...).Get(userCOSPostsUrlPath, nil)
	if err != nil {
		glog.Error("UserInfo send http err! httpUrl: %s err: %s \n", userCOSPostsUrlPath, err.Error())
		return err
	}
	doc, err := goquery.NewDocumentFromReader(attentionPage.Body)
	attentionPage.Body.Close()
	if err != nil {
		glog.Error("UserInfo read body err! userCOSPostsUrlPath: %s err: %s \n", userCOSPostsUrlPath, err.Error())
		return err
	}

	userNamePure := doc.Find(".l-left.mr5.fz22.text-shadow.lh28._white.text-shadow").Text()

	userName := getVaildName(userNamePure)

	//uid, _ := doc.Find("._btn-group._btn-group--gray.js-btn.l-right.mr10").Attr("data-uid")

	pathStorage := getCoserExistDirPath(id, userName, true)

	// ========

	this.Id = id
	this.UserName = userName
	this.UserNamePure = userNamePure

	this.UrlUserDetail = userCOSPostsUrlPath

	this.PathStorage = pathStorage

	this.PostCos = make([]*PostInfo, 0, 0)
	this.PostDaily = make([]*PostInfo, 0, 0)

	this.FollowingUid = make([]string, 0, 0)

	return nil
}

// 解析用户关注列表
// 获取本用户关注的用户uid列表 存储在 UserInfo.FollowingUid 数组
// 不做其他操作
func (this *UserInfo) AnalyseFollowingInfo() (err error) {
	if this.Id == "" {
		return errors.New(fmt.Sprintf("UserInfo Not Init"))
	}

	followingPage, err := httpclient.WithCookie(this.cookies...).Get(fmt.Sprintf("http://bcy.net/u/%s/following", this.Id), nil)
	if err != nil {
		glog.Error("AnalyseFollowingInfo http get follow page err! err: %s \n", err.Error())
		return err
	}
	defer followingPage.Body.Close()

	doc, err := goquery.NewDocumentFromReader(followingPage.Body)
	if err != nil {
		glog.Error("AnalyseFollowingInfo NewDocumentFromReader read error! err: %s \n", err.Error())
		return err
	}

	lastPageNumber := 0
	lastPager, bo := doc.Find(".pager li").Last().Find("a").Attr("href")
	if bo {
		numberArray := strings.Split(lastPager, "p=")
		if len(numberArray) != 2 {
			glog.Error("AnalyseFollowingInfo pager number analysis err! numberPager: %s \n", lastPager)
			return errors.New(fmt.Sprintf("AnalyseFollowingInfo pager number analysis err! numberPager: %s \n", lastPager))
		}
		lastPageNumber, err = strconv.Atoi(numberArray[1])
		if err != nil {
			glog.Error("AnalyseFollowingInfo pager number convert string to int err! numberPager: %s \n", numberArray[1])
			return err
		}

		glog.Info("following pageNumber %v\n", lastPageNumber)

		httpUrl := ""
		for pageNumber := 1; pageNumber <= lastPageNumber; pageNumber++ {
			httpUrl = fmt.Sprintf("http://bcy.net/u/%s/following?&p=%d", this.Id, pageNumber)
			glog.Info("%v\n", httpUrl)
			attentionPage, err := httpclient.WithCookie(this.cookies...).Get(httpUrl, nil)
			if err != nil {
				glog.Error("AnalyseFollowingInfo send http error! pageNumber: %d err: %s \n", pageNumber, err.Error())
				continue
			}
			doc, err := goquery.NewDocumentFromReader(attentionPage.Body)
			attentionPage.Body.Close()
			if err != nil {
				glog.Error("AnalyseFollowingInfo anaysis followUser Page err! err: %s \n", err.Error())
				continue
			}
			doc.Find(".l-newFanBoxList.l-clearfix").Find("li").Each(func(indexNumber int, nodeObj *goquery.Selection) {
				hrefStr, bo := nodeObj.Find("a").First().Attr("href")
				if bo {
					uidArray := strings.Split(hrefStr, "/u/")

					if len(uidArray) != 2 {
						glog.Error("get uid err! uidUrl: %s \n", hrefStr)
						//err = errors.New(fmt.Sprintf("get uid err! uidUrl: %s \n", hrefStr))
						return
					}

					uid := uidArray[1]

					///analysisFollowUser(hrefStr) /////////////////
					this.FollowingUid = append(this.FollowingUid, uid)
				}
			})
		}
	}

	return nil
}

// 解析用户的cos作品列表
// 存储在 UserInfo.PostCos 数组
// 不做其他操作
func (this *UserInfo) AnalysePostCosInfo() (err error) {
	userPageHomeUrl := fmt.Sprintf("http://bcy.net/u/%s/post/Cos", this.Id)
	glog.Info("userPageHomeUrl %s\n", userPageHomeUrl)
	attentionPage, err := httpclient.WithCookie(this.cookies...).Get(userPageHomeUrl, nil)
	if err != nil {
		glog.Error("AnalysePostCosInfo send http err! url: %s err: %s \n", userPageHomeUrl, err.Error())
		return
	}
	doc, err := goquery.NewDocumentFromReader(attentionPage.Body)
	attentionPage.Body.Close()
	if err != nil {
		glog.Error("AnalysePostCosInfo page analysis err! err: %s \n", err.Error())
		return
	}
	pageNumberANode, bo := doc.Find(".l-home-follow-pager li").Last().Find("a").Attr("href")
	if bo {
		pageNumberArray := strings.Split(pageNumberANode, "p=")
		if len(pageNumberArray) != 2 {
			glog.Error("AnalysePostCosInfo pageNumber analysis err! pageNumberANode: %s \n", pageNumberANode)
		}
		lastPageNumber, err := strconv.Atoi(pageNumberArray[1])
		if err != nil {
			glog.Error("AnalysePostCosInfo pageNumber can't convert string to int! pageNumberArraye: %v err: %s \n", pageNumberArray, err.Error())
			return err
		}
		//analysisFollowUserCOSEveryPage(userPageHomeUrl, pageNumber)

		glog.Info("AnalysePostCosInfo pageNumber %d\n", lastPageNumber)

		for pageNumber := 1; pageNumber <= lastPageNumber; pageNumber++ {
			httpUrl := fmt.Sprintf("%s?p=%d", userPageHomeUrl, pageNumber)
			glog.Info("httpUrl %s\n", httpUrl)
			attentionPage, err := httpclient.WithCookie(this.cookies...).Get(httpUrl, nil)
			if err != nil {
				glog.Error("AnalysePostCosInfo send http err! httpUrl: %s err: %s \n", httpUrl, err.Error())
				return err
			}
			doc, err := goquery.NewDocumentFromReader(attentionPage.Body)
			attentionPage.Body.Close()
			if err != nil {
				glog.Error("AnalysePostCosInfo analysis documenterr! sendHttp: %s  err: %s \n", httpUrl, err.Error())
				return err
			}
			doc.Find(".l-grid__inner li").Each(func(indexNumber int, nodeObj *goquery.Selection) {
				post := &PostInfo{}

				// 解析作品地址
				hrefUrlPath, bo := nodeObj.Find("a").First().Attr("href")
				if !bo {
					return
				}
				post.Url = fmt.Sprintf("http://bcy.net%s", hrefUrlPath)

				//userSendPostsProcess(fmt.Sprintf("http://bcy.net%s", hrefUrlPath))

				// 解析作品ID
				idArray := strings.Split(hrefUrlPath, "/")
				if len(idArray) == 0 {
					return
				}
				post.Id = idArray[len(idArray)-1]

				// 解析作品标题
				titlePure := nodeObj.Find("footer").First().Text()
				title := getVaildName(titlePure)

				post.TitlePure = titlePure
				post.Title = title

				post.PathStorage = this.PathStorage + "/cos/" + post.Id + "-" + post.Title

				post.Image = make([]string, 0, 0)

				this.PostCos = append(this.PostCos, post)
			})
		}
	}

	return
}

//////////////////////////////////////////////////////////////////////

// 解析用户作品页面的图片 适配于cos作品
// 存储在 PostInfo.Image 数组
// 不做其他操作
func (this *PostInfo) AnalysePostCosImageInfo(cookies []*http.Cookie) (err error) {
	//func userSendPostsProcess(userCOSPostsUrlPath string) {
	this.Image = make([]string, 0, 0)

	attentionPage, err := httpclient.WithCookie(cookies...).Get(this.Url, nil)
	//attentionPage, err := httpclient.WithHeaders(appConfig.HttpHeaderForNormal).WithCookie(cookies...).Get(this.Url, nil)
	if err != nil {
		glog.Error("userSendPostsProcess send http err! httpUrl: %s err: %s \n", this.Url, err.Error())
		return
	}

	doc, err := goquery.NewDocumentFromReader(attentionPage.Body)
	attentionPage.Body.Close()
	if err != nil {
		glog.Error("userSendPostsProcess read body err! userCOSPostsUrlPath: %s err: %s \n", this.Url, err.Error())
		return
	}

	//coserDirName := getCoserDir(uid, userName)
	//mkdirPath := fmt.Sprintf("./cos/%s", coserDirName)

	mkdirPath := this.PathStorage

	err = os.MkdirAll(mkdirPath, 0777)
	if err != nil {
		glog.Error("userSendPostsProcess create file err! mkdirPath: %s err: %s \n", mkdirPath, err.Error())
		return
	}
	mkdirPathFileName := doc.Find(".js-post-title").First().Text()
	mkdirPathFileName = strings.TrimSpace(mkdirPathFileName)
	mkdirPathFileNamePath := fmt.Sprintf("%s/%s", mkdirPath, getVaildName(mkdirPathFileName))
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
			//pictureDown(COSPictureUrlStr, mkdirPathFileNamePath)
			this.Image = append(this.Image, COSPictureUrlStr)
		}
	})
	return
}

// json 反序列化
func (this *UserInfo) Unmarshal(buf string) (user *UserInfo, err error) {
	err = json.Unmarshal([]byte(buf), this)
	if err != nil {
		return nil, err
	}
	return this, nil
}

// json 序列化
func (this *UserInfo) Marshal() (buf string, err error) {
	bufByte, err := json.Marshal(this)
	return string(bufByte), err
}

/////////////////////////////////////////////////////////////////////

func getVaildName(name string) string {
	name = trimInvalidChar(name)
	// 用户名中的 "-" 会被替换为 "="
	name = strings.Replace(name, "-", "=", -1)
	return name
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
