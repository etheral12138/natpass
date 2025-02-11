package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	rt "runtime"

	"github.com/kardianos/service"
	"github.com/lwch/logging"
	"github.com/lwch/natpass/code/client/app"
	"github.com/lwch/natpass/code/client/global"
	"github.com/lwch/natpass/code/client/rule/vnc"
	"github.com/lwch/natpass/code/utils"
	"github.com/lwch/runtime"
)

var (
	version      string = "0.0.0"
	gitHash      string
	gitReversion string
	buildTime    string
)

func showVersion() {
	fmt.Printf("version: v%s\ntime: %s\ncommit: %s.%s\n",
		version,
		buildTime,
		gitHash, gitReversion)
	os.Exit(0)
}

func main() {
	user := flag.String("user", "", "service user")
	conf := flag.String("conf", "", "configure file path")
	ver := flag.Bool("version", false, "show version info")
	act := flag.String("action", "", "install or uninstall")
	name := flag.String("name", "", "rule name")
	vport := flag.Uint("vport", 6155, "vnc worker listen port")
	vcursor := flag.Bool("vcursor", false, "vnc show cursor")
	flag.Parse()

	if *ver {
		showVersion()
		os.Exit(0)
	}

	if len(*conf) == 0 {
		fmt.Println("missing -conf param")
		os.Exit(1)
	}

	// for test
	// work := worker.NewWorker()
	// work.TestCapture()
	// return

	dir, err := filepath.Abs(*conf)
	runtime.Assert(err)

	var depends []string
	if rt.GOOS != "windows" {
		depends = append(depends, "After=network.target")
	}
	var opt service.KeyValue
	switch rt.GOOS {
	case "windows":
		opt = service.KeyValue{
			"StartType":              "automatic",
			"OnFailure":              "restart",
			"OnFailureDelayDuration": "5s",
			"OnFailureResetPeriod":   10,
		}
	case "linux":
		opt = service.KeyValue{
			"LimitNOFILE": 65000,
		}
	case "darwin":
		opt = service.KeyValue{
			"SessionCreate": true,
		}
	}

	appCfg := &service.Config{
		Name:         "np-cli",
		DisplayName:  "np-cli",
		Description:  "nat forward service",
		UserName:     *user,
		Arguments:    []string{"-conf", dir},
		Dependencies: depends,
		Option:       opt,
	}

	cfg := global.LoadConf(*conf)

	if *act == "vnc.worker" {
		defer utils.Recover("vnc.worker")
		stdout := true
		if rt.GOOS == "windows" {
			stdout = false
		}
		// go func() {
		// 	http.ListenAndServe(":9001", nil)
		// }()
		logging.SetSizeRotate(logging.SizeRotateConfig{
			Dir:         cfg.LogDir,
			Name:        "np-cli.vnc." + *name,
			Size:        int64(cfg.LogSize.Bytes()),
			Rotate:      cfg.LogRotate,
			WriteStdout: stdout,
			WriteFile:   true,
		})
		defer logging.Flush()
		vnc.RunWorker(uint16(*vport), *vcursor)
		return
	}

	app := app.New(version, *conf, cfg)
	sv, err := service.New(app, appCfg)
	runtime.Assert(err)

	switch *act {
	case "install":
		runtime.Assert(sv.Install())
		utils.BuildDir(cfg.LogDir, *user)
		utils.BuildDir(cfg.CodeDir, *user)
	case "uninstall":
		runtime.Assert(sv.Uninstall())
	default:
		runtime.Assert(sv.Run())
	}
}
