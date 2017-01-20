package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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

	// cookies []*http.Cookie // 存储登陆cookies // 注：httpclient 会自动根据域名存储cookies，默认后面同域名的请求会自动携带cookies

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

	PathStorage       string `json:"PathStorage"`       // 资源存储路径
	IgnorePathStorage string `json:"IgnorePathStorage"` // 如果资源被忽略，则使用的存储路径

	Image []*ImageInfo `json:"Image"` // 作品页面图片列表
}

// 作品图片信息
type ImageInfo struct {
	Url    string `json:"Url"`    // 地址
	Length int64  `json:"Length"` // 地址
}

// 解析 日常 作品页面的 timeline json 结构体
type Timeline struct {
	Status int          `json:"status"`
	Info   int          `json:"info"`
	Data   TimelineData `json:"data"`
}

type TimelineData struct {
	List []TimelinePost `json:"list"`
}

type TimelinePost struct {
	Detail TimelinePostDetail `json:"detail"`
}

type TimelinePostDetail struct {
	Ud_id int    `json:"ud_id"` // 作品 detail ID
	Plain string `json:"plain"` // 作品名称
}

func (this *UserInfo) Login(loginUserName string, password string) error {
	//res, err := httpclient.WithHeaders(nil).Post("http://bcy.net/public/dologin", map[string]string{
	_, err := httpclient.WithHeaders(appConfig.HttpHeaderForLogin).Post(appConfig.LoginUrl, map[string]string{
		"email":    loginUserName,
		"password": password,
		"remember": "1",
	})
	if err != nil {
		glog.Error("serverRun http login err! err: %s \n", err.Error())
		return err
	}
	glog.Info("Login cookies %v\n", httpclient.Cookies("http://bcy.net"))
	//this.cookies = httpclient.Cookies("http://bcy.net")
	return err
}

// 根据id初始化用户结构
// 1. 获取用户名
// 2. 获取用户详细页面url
// 3. 获取用户资源存储路径（用户改名会自动修正目录名，但是如果目录不存在不会创建文件夹）
func (this *UserInfo) Init(id string) error {
	userCOSPostsUrlPath := fmt.Sprintf("http://bcy.net/u/%s", id)

	attentionPage, err := httpclient.WithHeaders(appConfig.HttpHeaderForNormal).Get(userCOSPostsUrlPath, nil)
	if err != nil {
		glog.Error("UserInfo send http err! httpUrl: %s err: %s \n", userCOSPostsUrlPath, err.Error())
		return err
	}

	// 从 response 中读取 byte 数据，并生成新的 reader 供给 goquery 使用
	byteData, err := attentionPage.ReadAll()
	attentionPage.Body.Close()
	r := bytes.NewReader(byteData)

	doc, err := goquery.NewDocumentFromReader(r)

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

	followingPage, err := httpclient.WithHeaders(appConfig.HttpHeaderForNormal).Get(fmt.Sprintf("http://bcy.net/u/%s/following", this.Id), nil)
	if err != nil {
		glog.Error("AnalyseFollowingInfo http get follow page err! err: %s \n", err.Error())
		return err
	}

	// 从 response 中读取 byte 数据，并生成新的 reader 供给 goquery 使用
	byteData, err := followingPage.ReadAll()
	followingPage.Body.Close()
	r := bytes.NewReader(byteData)

	doc, err := goquery.NewDocumentFromReader(r)
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
			attentionPage, err := httpclient.WithHeaders(appConfig.HttpHeaderForNormal).Get(httpUrl, nil)
			if err != nil {
				glog.Error("AnalyseFollowingInfo send http error! pageNumber: %d err: %s \n", pageNumber, err.Error())
				continue
			}
			// 从 response 中读取 byte 数据，并生成新的 reader 供给 goquery 使用
			byteData, err := attentionPage.ReadAll()
			attentionPage.Body.Close()
			r := bytes.NewReader(byteData)

			doc, err := goquery.NewDocumentFromReader(r)
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
	glog.Info("AnalysePostCosInfo userPageHomeUrl %s\n", userPageHomeUrl)
	attentionPage, err := httpclient.WithHeaders(appConfig.HttpHeaderForNormal).Get(userPageHomeUrl, nil)
	if err != nil {
		glog.Error("AnalysePostCosInfo send http err! url: %s err: %s \n", userPageHomeUrl, err.Error())
		return err
	}

	// 从 response 中读取 byte 数据，并生成新的 reader 供给 goquery 使用
	byteData, err := attentionPage.ReadAll()
	attentionPage.Body.Close()
	r := bytes.NewReader(byteData)

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		glog.Error("AnalysePostCosInfo page analysis err! err: %s \n", err.Error())
		return err
	}

	lastPageNumber := 1
	pageNumberANode, bo := doc.Find(".l-home-follow-pager li").Last().Find("a").Attr("href")
	if bo {
		pageNumberArray := strings.Split(pageNumberANode, "p=")
		if len(pageNumberArray) != 2 {
			glog.Error("AnalysePostCosInfo pageNumber analysis err! pageNumberANode: %s \n", pageNumberANode)
		}
		lastPageNumber, err = strconv.Atoi(pageNumberArray[1])
		if err != nil {
			glog.Error("AnalysePostCosInfo pageNumber can't convert string to int! pageNumberArraye: %v err: %s \n", pageNumberArray, err.Error())
			return err
		}
	}

	glog.Info("AnalysePostCosInfo pageNumber %d\n", lastPageNumber)

	for pageNumber := 1; pageNumber <= lastPageNumber; pageNumber++ {
		httpUrl := fmt.Sprintf("%s?p=%d", userPageHomeUrl, pageNumber)
		glog.Info("httpUrl %s\n", httpUrl)
		attentionPage, err := httpclient.WithHeaders(appConfig.HttpHeaderForNormal).Get(httpUrl, nil)
		if err != nil {
			glog.Error("AnalysePostCosInfo send http err! httpUrl: %s err: %s \n", httpUrl, err.Error())
			return err
		}
		// 从 response 中读取 byte 数据，并生成新的 reader 供给 goquery 使用
		byteData, err := attentionPage.ReadAll()
		attentionPage.Body.Close()
		r := bytes.NewReader(byteData)

		doc, err := goquery.NewDocumentFromReader(r)
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
			post.IgnorePathStorage = this.PathStorage + "/cos/ignore/" + post.Id + "-" + post.Title

			post.Image = make([]*ImageInfo, 0, 0)

			this.PostCos = append(this.PostCos, post)
		})
	}

	return
}

// 解析用户的 日常(source=user) 作品列表
// 存储在 UserInfo.PostDaily 数组
// 不做其他操作
func (this *UserInfo) AnalysePostDailyInfo() (err error) {
	userPageHomeUrl := fmt.Sprintf("http://bcy.net/u/%s/timeline?&source=user&filter=origin", this.Id)

	glog.Info("AnalysePostDailyInfo userPageHomeUrl %s\n", userPageHomeUrl+"&p=1")
	attentionPage, err := httpclient.WithHeaders(appConfig.HttpHeaderForNormal).Get(userPageHomeUrl+"&p=1", nil)
	if err != nil {
		glog.Error("AnalysePostDailyInfo send http err! url: %s err: %s \n", userPageHomeUrl+"&p=1", err.Error())
		return err
	}

	// 从 response 中读取 byte 数据，并生成新的 reader 供给 goquery 使用
	byteData, err := attentionPage.ReadAll()
	attentionPage.Body.Close()
	r := bytes.NewReader(byteData)

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		glog.Error("AnalysePostDailyInfo page analysis err! err: %s \n", err.Error())
		return err
	}

	lastPageNumber := 1
	pageNumberANode, bo := doc.Find(".pager li").Last().Find("a").Attr("href")
	if bo {
		pageNumberArray := strings.Split(pageNumberANode, "p=")
		if len(pageNumberArray) != 2 {
			glog.Error("AnalysePostDailyInfo pageNumber analysis err! pageNumberANode: %s \n", pageNumberANode)
		}
		lastPageNumber, err = strconv.Atoi(pageNumberArray[1])
		if err != nil {
			glog.Error("AnalysePostDailyInfo pageNumber can't convert string to int! pageNumberArraye: %v err: %s \n", pageNumberArray, err.Error())
			return err
		}
	}

	glog.Info("AnalysePostDailyInfo pageNumber %d\n", lastPageNumber)

	for pageNumber := 1; pageNumber <= lastPageNumber; pageNumber++ {
		//  日常页面的获取方法不同，使用 post 请求获取 json 格式的日志时间线
		//  Request Headers
		//    Accept:*/*
		//    Accept-Encoding:gzip, deflate
		//    Accept-Language:en-US,en;q=0.8,zh-CN;q=0.6
		//    Connection:keep-alive
		//    Content-Length:63
		//    Content-Type:application/x-www-form-urlencoded; charset=UTF-8
		//    Cookie:acw_tc=AQAAAEPN7Sd8cAQAol5vtN0Bh9I8df6l; PHPSESSID=mfpioi81ad9t0jbrgm5fbh5800; LOGGED_USER=1BnyStMWv7TNA9GQFkYaug%3D%3D%3A3g3%2B34FO42ukWk2UN0dwCg%3D%3D; lang_set=zh; mobile_set=no; CNZZDATA1257708097=1593242468-1482150394-%7C1483029642; Hm_lvt_330d168f9714e3aa16c5661e62c00232=1482155542; Hm_lpvt_330d168f9714e3aa16c5661e62c00232=1483031792
		//    Host:bcy.net
		//    Origin:http://bcy.net
		//    Referer:http://bcy.net/u/34683/timeline?&source=user&filter=origin&p=16
		//    User-Agent:Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/55.0.2883.87 Safari/537.36
		//    X-Requested-With:XMLHttpRequest

		//  Form Data
		//    uid:34683
		//    sub:user
		//    since:300
		//    limit:20
		//    source:user
		//    filter:origin

		httpUrl := "http://bcy.net/home/user/loadtimeline"

		timelinePage, err := httpclient.WithHeaders(appConfig.HttpHeaderForNormal).Post(httpUrl, map[string]string{
			"uid":    this.Id,
			"sub":    "user",
			"since":  fmt.Sprintf("%d", (pageNumber-1)*20),
			"limit":  "20",
			"source": "user",
			"filter": "origin",
		})

		timelineJsonString, _ := timelinePage.ToString()
		//glog.Info("%s", timelineJsonString)

		timeline := Timeline{}

		err = json.Unmarshal([]byte(timelineJsonString), &timeline)
		if err != nil {
			return err
		}

		//glog.Info("%v", timeline)
		for _, value := range timeline.Data.List {
			timelinePostDetail := value.Detail

			post := &PostInfo{}

			// 解析作品地址
			post.Url = fmt.Sprintf("http://bcy.net/daily/detail/%d", timelinePostDetail.Ud_id)

			post.Id = fmt.Sprintf("%d", timelinePostDetail.Ud_id)

			// 解析作品标题
			titlePure := timelinePostDetail.Plain
			title := getVaildName(titlePure)

			post.TitlePure = titlePure
			post.Title = title

			post.PathStorage = this.PathStorage + "/daily/" + post.Id + "-" + post.Title
			post.IgnorePathStorage = this.PathStorage + "/daily/ignore/" + post.Id + "-" + post.Title

			post.Image = make([]*ImageInfo, 0, 0)

			this.PostDaily = append(this.PostDaily, post)
			str, _ := post.Marshal()
			glog.Info("%s\n", str)
		}
	}

	return
}

// 将 UserInfo 序列化，保存为用户解析的 json 文件
func (this *UserInfo) SaveCosInfo() (err error) {
	cacheFilePath := this.PathStorage + "/" + "user.json"

	json, err := this.Marshal()
	if err != nil {
		return err
	}

	var d1 = []byte(json)
	err = ioutil.WriteFile(cacheFilePath, d1, 0666) //写入文件(字节数组)
	return err
}

//////////////////////////////////////////////////////////////////////

// 解析用户作品页面的图片 适配于cos作品
// 存储在 PostInfo.Image 数组
// 不做其他操作
func (this *PostInfo) AnalysePostCosImageInfo(useCache bool, mkdir bool) (err error) {
	this.Image = make([]*ImageInfo, 0, 0)

	var doc *goquery.Document
	cacheFilePath := this.GetCacheFileDirPath() + "/" + "post.json"
	cacheHtmlFilePath := this.GetCacheFileDirPath() + "/" + "post.html"

	isIgnore := this.IsIngore()

	if isIgnore {
		glog.Info("AnalysePostCosImageInfo ignorePath judge %s [ignore]\n", this.IgnorePathStorage)
	} else {
		glog.Info("AnalysePostCosImageInfo ignorePath judge %s [normal]\n", this.IgnorePathStorage)
	}

	if mkdir && !isIgnore {
		err = os.MkdirAll(this.PathStorage, 0777)
		if err != nil {
			glog.Error("AnalysePostCosImageInfo create file err! mkdirPath: %s err: %s \n", this.PathStorage, err.Error())
			return err
		}
	}

	if useCache && fileExist(cacheFilePath) {
		fi, err := os.Open(cacheFilePath)
		if err != nil {
			panic(err)
		}

		fd, err := ioutil.ReadAll(fi)
		if err != nil {
			panic(err)
		}

		content := string(fd)
		//doc, err = goquery.NewDocumentFromReader(fi)
		fi.Close()

		_, err = this.Unmarshal(content)
		if err != nil {
			panic(err)
		}

	} else {
		attentionPage, err := httpclient.WithHeaders(appConfig.HttpHeaderForNormal).Get(this.Url, nil)
		if err != nil {
			glog.Error("AnalysePostCosImageInfo send http err! httpUrl: %s err: %s \n", this.Url, err.Error())
			return err
		}

		os.Remove(cacheHtmlFilePath)
		dstFile, err := os.Create(cacheHtmlFilePath)
		if err != nil {
			glog.Error("Create html cache file fail %s, err: %s\n", cacheHtmlFilePath, err.Error())
			// 从 response 中读取 byte 数据，并生成新的 reader 供给 goquery 使用
			byteData, _ := attentionPage.ReadAll()
			r := bytes.NewReader(byteData)

			doc, _ = goquery.NewDocumentFromReader(r)
		} else {
			bodyString, _ := attentionPage.ToString()
			dstFile.WriteString(bodyString)
			glog.Info("Create cache file %s, len: %d\n", cacheHtmlFilePath, len(bodyString))
			dstFile.Close()

			fi, err := os.Open(cacheHtmlFilePath)
			if err != nil {
				panic(err)
			}
			doc, err = goquery.NewDocumentFromReader(fi)
			fi.Close()
		}

		attentionPage.Body.Close()

		if err != nil {
			glog.Error("AnalysePostCosImageInfo read body err! userCOSPostsUrlPath: %s err: %s \n", this.Url, err.Error())
			return err
		}

		target := doc.Find(".detail_std.detail_clickable")
		if target != nil {
			target.Each(func(indexNumber int, nodeObj *goquery.Selection) {
				COSPictureUrlStr, bo := nodeObj.Attr("src")
				if bo {
					image := &ImageInfo{}

					COSPictureUrlStr = COSPictureUrlStr[:strings.LastIndex(COSPictureUrlStr, "/")]
					//pictureDown(COSPictureUrlStr, mkdirPathFileNamePath)

					image.Url = COSPictureUrlStr
					image.Length = 0

					this.Image = append(this.Image, image)
				}
			})
		}
	}

	return nil
}

// 将 PostInfo 序列化，保存为用户作品页面解析的 json 文件，用于下次缓存使用, 适配于 cos 作品
func (this *PostInfo) SavePostCosImageInfo() (err error) {
	cacheFilePath := this.GetCacheFileDirPath() + "/" + "post.json"

	json, err := this.Marshal()
	if err != nil {
		return err
	}

	var d1 = []byte(json)
	err = ioutil.WriteFile(cacheFilePath, d1, 0666) //写入文件(字节数组)
	return err
}

// 下载用户作品页面的图片 适配于cos作品
// 下载 PostInfo.Image 数组中存储的图片地址
// 如果本地存在同名文件且大小和 ImageInfo 中存储的大小相等则不再重复下载
// 下载文件的时候会将网络获取的图片大小刷新到 ImageInfo 结构体中
func (this *PostInfo) DownloadPostCosImage() (err error) {
	isIgnore := this.IsIngore()

	if isIgnore {
		glog.Info("DownloadPostCosImage ignorePath judge %s [ignore]\n", this.IgnorePathStorage)
		return nil
	} else {
		glog.Info("DownloadPostCosImage ignorePath judge %s [normal]\n", this.IgnorePathStorage)
	}

	// 获取作品 ignore 根目录,并进行目录创建
	ignoreTypePathStorageArray := strings.Split(this.IgnorePathStorage, "/")
	ignoreTypePathStorageArray = ignoreTypePathStorageArray[:len(ignoreTypePathStorageArray)-1]
	ignoreTypePathStorage := strings.Join(ignoreTypePathStorageArray, "/")
	err = os.MkdirAll(ignoreTypePathStorage, 0777)

	// 创建正常的存储目录
	err = os.MkdirAll(this.PathStorage, 0777)
	if err != nil {
		glog.Error("DownloadPostCosImage create file err! mkdirPath: %s err: %s \n", this.PathStorage, err.Error())
		return err
	}

	for _, image := range this.Image {
		urlPathArray := strings.Split(image.Url, "/")
		imageFileName := urlPathArray[len(urlPathArray)-1]
		imageFilePath := this.PathStorage + "/" + imageFileName

		if fileExist(imageFilePath) {
			// 如果本地存在同名文件且大小和 ImageInfo 中存储的大小相等则不再重复下载
			localFileSize, _ := fileSize(imageFilePath)
			if localFileSize == image.Length {
				glog.Info("DownloadPostCosImage %s [Exist]\n", image.Url)
				continue
			} else {
				glog.Info("DownloadPostCosImage %s [localFileSize: %d jsonFileSize: %d][Exist but fileSize not matched] [redownload]\n",
					image.Url, localFileSize, image.Length)
			}
		}

		//res, err := http.WithHeaders(appConfig.HttpHeaderForNormal).Get(image.Url)
		res, err := http.Get(image.Url)
		if err != nil {
			glog.Error("DownloadPostCosImage send http err! url: %s err: %s \n", image.Url, err.Error())
			return err
		}

		// 将网络获取的图片大小刷新到 ImageInfo 结构体中
		image.Length = res.ContentLength

		file, err := os.Create(imageFilePath)
		if err != nil {
			glog.Error("DownloadPostCosImage picute create err! filePath: %s err: %s \n", imageFilePath, err.Error())
			return err
		}
		io.Copy(file, res.Body)
		glog.Info("DownloadPostCosImage image filePath: %s Length: %d \n", imageFilePath, image.Length)
		res.Body.Close()
		file.Close()

		// 每次下载成功一个图片就会保存一次 json (因为有图片大小更新)
		this.SavePostCosImageInfo()
		if err != nil {
			glog.Error("err %s\n", err.Error())
		}
	}
	return nil
}

// 清理用户作品页面的图片，前提是文件夹被放置到了 ignore 目录， 适配于cos作品
// 删除 PostInfo.Image 数组中存储的图片地址，只保留第一个图片
func (this *PostInfo) ClearPostCosImage() (err error) {
	isIgnore := this.IsIngore()

	if isIgnore {
		glog.Info("ClearPostCosImage ignorePath judge %s [ignore]\n", this.IgnorePathStorage)
	} else {
		glog.Info("ClearPostCosImage ignorePath judge %s [normal]\n", this.IgnorePathStorage)
		return nil
	}

	for key, image := range this.Image {
		if key > 0 {
			urlPathArray := strings.Split(image.Url, "/")
			imageFileName := urlPathArray[len(urlPathArray)-1]
			imageFilePath := this.GetCacheFileDirPath() + "/" + imageFileName

			glog.Info("ClearPostCosImage remove ignore file %s\n", imageFilePath)
			os.Remove(imageFilePath)
		}
	}
	return nil
}

// 判断是否被忽略
func (this *PostInfo) IsIngore() bool {
	return fileExist(this.IgnorePathStorage)
}

func (this *PostInfo) GetCacheFileDirPath() string {
	isIgnore := this.IsIngore()

	if isIgnore {
		return this.IgnorePathStorage
	} else {
		return this.PathStorage
	}
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
	bufByte, err := json.MarshalIndent(this, "", "    ")
	return string(bufByte), err
}

// json 反序列化
func (this *PostInfo) Unmarshal(buf string) (user *PostInfo, err error) {
	err = json.Unmarshal([]byte(buf), this)
	if err != nil {
		return nil, err
	}
	return this, nil
}

// json 序列化
func (this *PostInfo) Marshal() (buf string, err error) {
	bufByte, err := json.MarshalIndent(this, "", "    ")
	return string(bufByte), err
}

/////////////////////////////////////////////////////////////////////

// 删除不能用于文件名的字符
func trimInvalidChar(name string) string {
	var invalid []string = []string{"\\", "/", ":", "?", "*", "\"", "<", "|", ">"}
	for _, value := range invalid {
		name = strings.Replace(name, value, "", -1)
	}
	return name
}

func getVaildName(name string) string {
	name = trimInvalidChar(name)
	// 用户名中的 "-" 会被替换为 "="
	name = strings.Replace(name, "-", "=", -1)
	return name
}

// 获取自己的uid
func getMyUserIndex() (uid string, errRet error) {
	uid = ""
	userIndexPage, err := httpclient.WithHeaders(appConfig.HttpHeaderForNormal).Get("http://bcy.net/home/user/index", nil)
	if err != nil {
		glog.Error("getMyUserIndex http get index page err! err: %s \n", err.Error())
		return uid, errors.New(fmt.Sprintf("getMyUserIndex http get index page err! err: %s \n", err.Error()))
	}
	defer userIndexPage.Body.Close()

	// 从 response 中读取 byte 数据，并生成新的 reader 供给 goquery 使用
	byteData, err := userIndexPage.ReadAll()
	r := bytes.NewReader(byteData)

	doc, err := goquery.NewDocumentFromReader(r)
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
	} else {
		return uid, errors.New("uid dom item not found")
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
