# RSCP

弱网环境下的ssh文件上传/下载工具

SSH file upload/download tool in weak network environment


## USAGE
```
  -b int
        each block size (bytes)
  -c string
        connect string, e.g ssh://root:root@1.1.1.1
  -download
        download mod
  -i string
        private key file path
  -lf string
        local file path
  -offset int
        offset
  -rf string
        remote file path (default "/tmp/164200825/")
  -upload
        upload mod
```
## quickstart
**upload**

`rscp -c ssh://root:root@123.123.123.123 -lf bin -rf /tmp/bin -upload`

**download**

`rscp -c ssh://root:root@123.123.123.123 -rm /home/bin -lf bin -download` 

options:

It will retry indefinitely, until the file is successfully downloaded or uploaded.

If the command is ended manually. You can use -offset flag to specify the block id of the breakpoint to continue the transfer

example:
`rscp -c ssh://root:root@123.123.123.123 -rm /home/bin -lf bin -download -offset 15` 

## TODO
* add ssh proxy
* add socks proxy
* add http/https proxy

## THANKS
@https://github.com/PassingFoam
