package v1

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"os"
)

func fileInitialize(filename string) (*os.File, error) {
	var err error
	var filehandle *os.File
	if isExist(filename) { //如果文件存在
		return nil, errors.New("file already exists")
	} else {
		filehandle, err = os.Create(filename) //创建文件
		if err != nil {
			return nil, err
		}
	}
	return filehandle, err
}

func isExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

func Base64Decode(s string) []byte {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		println(err.Error())
		os.Exit(0)
	}
	return data
}

func Base64Encode(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func Md5Hash(raw []byte) string {
	m := md5.Sum(raw)
	return hex.EncodeToString(m[:])
}

func splitFile(filename string, length int)chan block {
	ch := make(chan block)
	var err error
	f, err := os.Open(filename)
	if err != nil{
		println(err.Error())
		os.Exit(0)
	}
	go func() {
		bs := make([]byte, length)
		for{
			n, err := f.Read(bs)
			bs = bs[:n]
			b := block{
				md5sum:  Md5Hash(bs),
				content: Base64Encode(bs),
			}
			if err == io.EOF{
				close(ch)
				break
			}else{
				ch <- b
			}
		}
	}()
	return ch
}

func fileSize(filename string)int  {
	file,err:=os.Open(filename)

	if err == nil {
		fi,_ := file.Stat()
		return int(fi.Size())
	}
	return 0
}
