package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func getCoserExistDirPath(uid string, userName string, autoCreateOrUpdate bool) string {
	userName = getVaildName(userName)

	dirList, dirNameList, _ := getDirList("./cos", uid+"-")
	//glog.Info("dirList %v\n", dirList)
	//glog.Info("dirNameList %v\n", dirNameList)

	if !autoCreateOrUpdate {
		if len(dirList) > 0 {
			return dirList[0]
		} else {
			return ""
		}
	}

	formerPath := ""
	secondPart := ""
	if len(dirList) > 0 {
		formerPath = dirList[0]
		secondPart = strings.TrimLeft(dirNameList[0], uid+"-")
	}

	nameArrayTmp := strings.Split(secondPart, "-")

	nameArray := make([]string, 0, 0)
	for i := 0; i < len(nameArrayTmp); i++ {
		if nameArrayTmp[i] != "" && nameArrayTmp[i] != "-" {
			nameArray = append(nameArray, nameArrayTmp[i])
		}
	}

	//glog.Info("nameArray %v len %d", nameArray, len(nameArray))

	if len(nameArray) > 1 {
		if nameArray[1] != userName {
			nameArray[1] = userName
		}
	} else if len(nameArray) == 1 {
		if nameArray[0] != userName {
			nameArray = append(nameArray, userName)
		}
	} else if len(nameArray) == 0 {
		nameArray = append(nameArray, userName)
	}

	nameUnion := uid + "-" + nameArray[0]

	//glog.Info("%v %v %v", nameUnion, uid, nameArray)

	if len(nameArray) > 1 {
		nameUnion += "-" + nameArray[1]
	}

	//glog.Info(fmt.Sprintf("nameUnion %s\n", nameUnion))

	currentPath := "./cos/" + nameUnion

	//glog.Info(fmt.Sprintf("formerPath %s\n", formerPath))

	if formerPath != currentPath && formerPath != "" && !fileExist(currentPath) {
		err := os.Rename(formerPath, currentPath)
		_ = err
		//glog.Info(fmt.Sprintf("%v\n", err))
	}

	//glog.Info(fmt.Sprintf("currentPath %s\n", currentPath))

	return currentPath
}

// 检查文件或目录是否存在
// 如果由 filename 指定的文件或目录存在则返回 true，否则返回 false
func fileExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func getDirList(dirPth string, prefix string) (dirList []string, dirNameList []string, err error) {
	dirList = make([]string, 0, 0)
	dirNameList = make([]string, 0, 0)

	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		return dirList, dirNameList, err
	}
	//PthSep := string(os.PathSeparator)
	PthSep := "/"
	for _, fi := range dir {
		if fi.IsDir() {
			if strings.HasPrefix(strings.ToUpper(fi.Name()), prefix) { //匹配文件
				dirList = append(dirList, dirPth+PthSep+fi.Name())
				dirNameList = append(dirNameList, fi.Name())
			}
		}
	}
	return dirList, dirNameList, nil
}

//获取指定目录下的所有文件，不进入下一级目录搜索，可以匹配后缀过滤。
func ListDir(dirPth string, suffix string) (files []string, err error) {
	files = make([]string, 0, 10)
	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		return nil, err
	}
	PthSep := string(os.PathSeparator)
	suffix = strings.ToUpper(suffix) //忽略后缀匹配的大小写
	for _, fi := range dir {
		if fi.IsDir() { // 忽略目录
			continue
		}
		if strings.HasSuffix(strings.ToUpper(fi.Name()), suffix) { //匹配文件
			files = append(files, dirPth+PthSep+fi.Name())
		}
	}
	return files, nil
}

//获取指定目录及所有子目录下的所有文件，可以匹配后缀过滤。
func WalkDir(dirPth, suffix string) (files []string, err error) {
	files = make([]string, 0, 30)
	suffix = strings.ToUpper(suffix)                                                     //忽略后缀匹配的大小写
	err = filepath.Walk(dirPth, func(filename string, fi os.FileInfo, err error) error { //遍历目录
		//if err != nil { //忽略错误
		// return err
		//}
		if fi.IsDir() { // 忽略目录
			return nil
		}
		if strings.HasSuffix(strings.ToUpper(fi.Name()), suffix) {
			files = append(files, filename)
		}
		return nil
	})
	return files, err
}

//func main() {
//	files, err := ListDir("D:\\Go", ".txt")
//	fmt.Println(files, err)
//	files, err = WalkDir("E:\\Study", ".pdf")
//	fmt.Println(files, err)
//}
