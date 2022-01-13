package v1

import (
	"flag"
)
var blockSize int
func CMD() {
	connectStr := flag.String("c", "" ,"")
	keyPath := flag.String("i","","")
	lfile := flag.String("lf", "", "")
	//rfile := flag.String("rf", "", "")
	rfile := flag.String("rf", "/tmp/164200825/","")
	blockoffset := flag.Int("offset", 0 , "")
	upload := flag.Bool("upload", false, "")
	download := flag.Bool("download", false, "")
	flag.IntVar(&blockSize, "b", 0, "")
	flag.Parse()
	//source := "127.0.0.1"
	var err error
	sshs := NewSSH(*connectStr, *keyPath)
	err = sshs.Connect()
	if err != nil{
		return
	}
	if *upload{
		if blockSize == 0{
			blockSize = 20480
		}
		sshs.Upload(*lfile, *rfile, *blockoffset)
	}else if *download{
		if blockSize == 0{
			blockSize = 102400
		}
		sshs.Download(*rfile, *lfile, *blockoffset)
	}
}
