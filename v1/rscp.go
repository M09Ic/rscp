package v1

import (
	"flag"
	"fmt"
	"os"
)

type options struct {
	connStr     string
	keyPath     string
	localfile   string
	remotefile  string
	blockoffset int
	upload      bool
	download    bool
}

var blockSize int

func CMD() {
	var opt options
	flag.StringVar(&opt.connStr, "c", "", "connect string, e.g ssh://root:root@1.1.1.1")
	flag.StringVar(&opt.keyPath, "i", "", "private key file path")
	flag.StringVar(&opt.localfile, "lf", "", "local file path")
	flag.StringVar(&opt.remotefile, "rf", "/tmp/164200825/", "remote file path")
	flag.IntVar(&opt.blockoffset, "offset", 0, "offset")
	flag.BoolVar(&opt.upload, "upload", false, "upload mod")
	flag.BoolVar(&opt.download, "download", false, "download mod")
	flag.IntVar(&blockSize, "b", 0, "each block size (bytes)")
	flag.Parse()
	opt.initOptions()

	var err error
	sshs := NewSSH(opt.connStr, opt.keyPath)
	err = sshs.Connect()
	if err != nil {
		return
	}
	fmt.Println("ssh connect successfully")

	if opt.upload {
		sshs.Upload(opt.localfile, opt.remotefile, opt.blockoffset)
	} else if opt.download {
		sshs.Download(opt.remotefile, opt.localfile, opt.blockoffset)
	}
}

func (opt *options) initOptions() {
	if opt.connStr == "" {
		fmt.Println("please input ssh url")
		os.Exit(0)
	}

	if opt.upload {
		if opt.localfile == "" {
			fmt.Println("please input upload file path")
			os.Exit(0)
		}
	}

	if opt.download {
		if opt.remotefile == "/tmp/164200825/" {
			fmt.Println("please input download file path")
			os.Exit(0)
		}
	}

	if blockSize == 0 {
		if opt.upload {
			blockSize = 20480
		} else if opt.download {
			blockSize = 102400
		} else {
			fmt.Println("please set -download/-upload flags")
			os.Exit(0)
		}
	}
}
