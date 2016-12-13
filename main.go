package main

import (
	"flag"
	"github.com/guotie/config"
	"github.com/guotie/deferinit"
	"github.com/smtc/glog"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

var (
	configFn  = flag.String("config", "./config.json", "config file path")
	debugFlag = flag.Bool("d", false, "debug mode")
	timing    = 3 * time.Hour //定时运行
	email     string          //账号
	password  string          //密码
)

/**
构造函数主体
创建人:邵炜
创建时间:2016年12月13日14:44:34
*/
func init() {
	deferinit.AddRoutine(watchAttention)
}

/**
程序退出
创建人:邵炜
创建时间:2016年12月13日14:44:17
*/
func serverExit() {
	// 结束所有go routine
	deferinit.StopRoutines()

	deferinit.FiniAll()
}

/**
主函数入口
创建人:邵炜
创建时间:2016年12月13日14:44:10
*/
func main() {
	if checkPid() {
		return
	}
	flag.Parse()
	logInit(*debugFlag)
	config.ReadCfg(*configFn)
	email = config.GetString("email")
	password = config.GetString("password")

	// 初始化
	deferinit.InitAll()
	glog.Info("init all module successfully.\n")
	// 设置多cpu运行
	runtime.GOMAXPROCS(runtime.NumCPU())
	deferinit.RunRoutines()
	glog.Info("run routines successfully.\n")

	c := make(chan os.Signal, 1)
	writePid()
	// 信号处理
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)
	// 等待信号
	<-c

	serverExit()
	rmPidFile()
	glog.Close()
	os.Exit(0)
}

/**
定时运行
*/
func watchAttention(ch chan struct{}, wg *sync.WaitGroup) {
	jsTmr := time.NewTimer(timing)
	go func() {
		<-ch

		jsTmr.Stop()
		wg.Done()
	}()
	for {
		go serverRun()
		jsTmr.Reset(timing)
		<-jsTmr.C
	}
}
