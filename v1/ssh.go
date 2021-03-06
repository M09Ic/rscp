package v1

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	url2 "net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func NewSSH(connectStr, pkfile string) *Rscp {
	if !strings.HasPrefix(connectStr, "ssh://") {
		connectStr = "ssh://" + connectStr
	}
	url, err := url2.Parse(connectStr)
	if err != nil {
		return nil
	}

	if pair := strings.Split(url.Host, ":"); len(pair) == 1 {
		url.Host += ":22"
	}
	var SSH *Rscp = &Rscp{
		target: url.Host,
	}
	user := url.User
	SSH.username = user.Username()
	if pkfile != "" {
		SSH.auth = pkAuth(pkfile)
	} else {
		password, ok := user.Password()
		if ok {
			SSH.auth = ssh.Password(password)
		} else {
			return nil
		}
	}
	return SSH
}

type Rscp struct {
	target   string
	auth     ssh.AuthMethod
	username string
	client   *ssh.Client
	agents   []ssh.AuthMethod
}

func (s *Rscp) Connect() error {
	var err error
	config := &ssh.ClientConfig{
		User: s.username,
		Auth: []ssh.AuthMethod{
			s.auth,
		},
		Timeout:         time.Duration(10) * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	s.client, err = ssh.Dial("tcp", s.target, config)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	return nil
}

func (s *Rscp) Run(command string, logHistory bool) (string, error) {
	if s.client == nil {
		return "", errors.New("nil ssh client")
	}
	session, err := s.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	if !logHistory {
		command = " " + command
		//command += " && history -r "
	}
	command += " && echo sangfor ; echo finish "
	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", err
	}
	if !bytes.Contains(output, []byte("sangfor")) {
		return "", errors.New("command exec failed")
	}
	if bytes.Equal(output, []byte("\nsangfor\nfinish\n")) {
		return "", io.EOF
	}
	return strings.Split(string(output), "\n")[0], nil
}

func (s *Rscp) md5(filename string) string {
	md5, err := s.Run("md5sum "+filename, false)
	if err != nil {
		return ""
	}
	if strings.Contains(md5, filename) {
		return strings.Split(md5, " ")[0]
	} else {
		return ""
	}

}

var filemask = "1632002513"

func (s *Rscp) echo(content, tmpfile string) (string, error) {
	cmd := fmt.Sprintf("echo %s | base64 -d > %s && md5sum %s", content, tmpfile, tmpfile)
	output, err := s.Run(cmd, false)
	if err != nil {
		return "", nil
	}
	//if !strings.Contains(output, "sangfor") {
	//	return "", errors.New("write fail")
	//}
	outs := strings.Split(output, " ")
	md5sum := outs[0]
	return md5sum, nil
}

func (s *Rscp) read(remotefile string, off int) ([]byte, error) {
	cmd := fmt.Sprintf("dd if=%s bs=%d count=1 skip=%d 2>/dev/null  | base64 -w 0 && echo", remotefile, blockSize, off)
	output, err := s.Run(cmd, false)
	if err != nil {
		return []byte{}, err
	}
	//if !strings.Contains(output, "sangfor") {
	//	return []byte{}, errors.New("read fail")
	//}
	//if output == "\nsangfor\nfinish\n" {
	//	return []byte{}, io.EOF
	//}
	//outs := strings.Split(output, "\n")
	return Base64Decode(output), nil
}

func pkAuth(kPath string) ssh.AuthMethod {

	key, err := ioutil.ReadFile(kPath)
	if err != nil {
		log.Fatal("ssh key file read failed", err)
	}
	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatal("ssh key signer failed", err)
	}
	return ssh.PublicKeys(signer)
}
func (s *Rscp) Upload(filename string, path string, offset int) {
	var err error
	localfilemd5 := filename
	ch := splitFile(filename, blockSize)
	if path[len(path)-1] != '/' {
		path += "/"
	}
	var i int
	_, _ = s.Run("mkdir "+path, false)
	var files []string
	tmpbase := path + filemask
	for block := range ch {
		var md5sum string
		var err error
		if i < offset {
			i++
			continue
		}
		tmpfile := fmt.Sprintf("%s_%d", tmpbase, i)
		for {
			var retry int
			md5sum, err = s.echo(block.content, tmpfile)
			if err != nil {
				fmt.Println(err.Error())
				_ = s.Connect()
				retry++
				continue
			}
			if md5sum == block.md5sum {
				fmt.Printf("block %d write to %s successfully, next block %d \n", i, tmpfile, i+1)
				files = append(files, fmt.Sprintf("%s_%d", filemask, i))
				break
			} else {
				fmt.Printf("uploaded checksum: %s, correert checksum %s, retry %d", md5sum, block.md5sum, retry)
				retry++
			}
		}
		i++
	}

	// ????????????
	_, err = s.Run(fmt.Sprintf("cd %s && cat %s > %s", path, strings.Join(files, " "), path+filename), false)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("files merge successfully, final file: " + tmpbase + filename)
	_, err = s.Run(fmt.Sprintf("rm -rf %s*", tmpbase), false)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("rm all blocks successfully")

	remotefilemd5 := s.md5(filepath.Join(tmpbase, filename))
	fmt.Printf("local file md5: %s, remote file md5:%s, check status: %t\n", localfilemd5, remotefilemd5, localfilemd5 == remotefilemd5)

}

func (s Rscp) Download(remoteFile, localFile string, offset int) {
	remotefilemd5 := s.md5(remoteFile)
	fmt.Println("remote file md5: " + remotefilemd5)
	if localFile == "" {
		_, localFile = filepath.Split(remoteFile)
	} else {
		return
	}
	f, err := fileInitialize(localFile)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	writer := bufio.NewWriter(f)
	for {
		var err error
		var retry int
		content, err := s.read(remoteFile, offset)
		if err != nil {
			if err == io.EOF {
				break
			}
			_ = s.Connect()
			retry++
			fmt.Printf("%s, retry %d times\n", err.Error(), retry)
			continue
		}
		_, err = writer.Write(content)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		_ = f.Sync()
		fmt.Printf("read %d block %d bytes successfully, next block id: %d \n", offset, len(content), offset+1)
		offset++
		retry = 0
		time.Sleep(1000)
	}
	localfilemd5 := fileMd5(localFile)

	fmt.Printf("download %s successfully, save to %s \n", remoteFile, localFile)
	fmt.Printf("local file md5: %s, remote file md5: %s, check status: %t\n", localfilemd5, remotefilemd5, localfilemd5 == remotefilemd5)
}

type block struct {
	md5sum  string
	content string
}

func splitFile(filename string, length int) chan block {
	ch := make(chan block)
	var err error
	f, err := os.Open(filename)
	if err != nil {
		println(err.Error())
		os.Exit(0)
	}
	go func() {
		bs := make([]byte, length)
		for {
			n, err := f.Read(bs)
			bs = bs[:n]
			b := block{
				md5sum:  Md5Hash(bs),
				content: Base64Encode(bs),
			}
			if err == io.EOF {
				close(ch)
				break
			} else {
				ch <- b
			}
		}
	}()
	return ch
}
