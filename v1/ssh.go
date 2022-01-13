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
	"path/filepath"
	"strings"
	"time"
)

type publicKey struct {
	keypath []string
	key     []string
}

func NewSSH(connectStr, pkfile string) *sshConfig {
	if !strings.HasPrefix(connectStr, "ssh://") {
		connectStr = "ssh://" + connectStr
	}
	url, err := url2.Parse(connectStr)
	if err != nil {
		return nil
	}

	var SSH *sshConfig = &sshConfig{
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

type sshConfig struct {
	target   string
	auth     ssh.AuthMethod
	username string
	client   *ssh.Client
	agents   []ssh.AuthMethod
}

func (s *sshConfig) Connect() error {
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

func (s *sshConfig) Run(command string, logHistory bool) (string, error) {
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
	if bytes.Contains(output, []byte("finish")) {
		return string(output), nil
	} else {
		return "", errors.New("no success")
	}
}

var filemask = "1632002513"

func (s *sshConfig) Upload(filename string, path string, offset int) {
	var err error
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

	// 合并文件
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
}

func (s *sshConfig) echo(content, tmpfile string) (string, error) {
	cmd := fmt.Sprintf("echo %s | base64 -d > %s && md5sum %s", content, tmpfile, tmpfile)
	output, err := s.Run(cmd, false)
	if err != nil {
		return "", nil
	}
	if !strings.Contains(output, "sangfor") {
		return "", errors.New("write fail")
	}
	outs := strings.Split(output, " ")
	md5sum := outs[0]
	return md5sum, nil
}

func (s sshConfig) Download(remoteFile, localFile string, offset int) {
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
			fmt.Printf("%s\n retry %d times\n", err.Error(), retry)
			retry++
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
	}
	fmt.Printf("download %s successfully, write it to %s \n", remoteFile, localFile)
}

func (s *sshConfig) read(remotefile string, off int) ([]byte, error) {
	cmd := fmt.Sprintf("dd if=%s bs=%d count=1 skip=%d 2>/dev/null  | base64 -w 0 && echo", remotefile, blockSize, off)
	output, err := s.Run(cmd, false)
	if err != nil {
		return []byte{}, err
	}
	if !strings.Contains(output, "sangfor") {
		return []byte{}, errors.New("read fail")
	}
	if output == "\nsangfor\nfinish\n" {
		return []byte{}, io.EOF
	}
	outs := strings.Split(output, "\n")
	return Base64Decode(outs[0]), nil
}

type block struct {
	md5sum  string
	content string
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
