package xweb

import (
	"flag"
	"github.com/sevlyar/go-daemon"
	"log"
	"os"
	"syscall"
)

var (
	//服务控制 app -s (stop|reload)
	signal = flag.String("s", "", "stop,reload")
	//运行daemon模式app -d
	runDaemon = flag.Bool("d", false, "run daemon mode")
	//退出时执行
	QuitHandler func() = nil
	//重新加载
	ReloadHandler func() = nil
)

func quitHandler(sig os.Signal) error {
	log.Println("daemon quit")
	if QuitHandler != nil {
		QuitHandler()
	} else {
		os.Exit(0)
	}
	return nil
}

func reloadHandler(sig os.Signal) error {
	if ReloadHandler != nil {
		ReloadHandler()
	}
	return nil
}

func Daemon(mainFunc func(), workdir, pidFile, logFile string) {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	flag.Parse()
	if !*runDaemon && len(*signal) == 0 {
		log.Println("use: -s(stop|reload) -d(daemon mode)")
		mainFunc()
		return
	}
	daemon.AddCommand(daemon.StringFlag(signal, "stop"), syscall.SIGQUIT, quitHandler)
	daemon.AddCommand(daemon.StringFlag(signal, "reload"), syscall.SIGHUP, reloadHandler)
	cntxt := &daemon.Context{
		PidFileName: pidFile,
		PidFilePerm: 0644,
		LogFileName: logFile,
		LogFilePerm: 0640,
		WorkDir:     workdir,
		Umask:       027,
		Args:        os.Args,
	}
	if len(daemon.ActiveFlags()) > 0 {
		if d, err := cntxt.Search(); err == nil {
			daemon.SendCommands(d)
		}
		return
	}
	d, err := cntxt.Reborn()
	if err != nil {
		log.Fatalln(err)
	}
	if d != nil {
		return
	}
	defer cntxt.Release()
	log.Println("daemon start")
	go mainFunc()
	err = daemon.ServeSignals()
	if err != nil {
		log.Println("Error:", err)
	}
}
