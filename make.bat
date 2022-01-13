rm ./bin/*
set name=rscp
gox.exe -osarch="linux/amd64 linux/arm64 linux/386 windows/amd64 linux/mips64 windows/386 darwin/amd64" -ldflags="-s -w" -gcflags="-trimpath=$GOPATH" -asmflags="-trimpath=$GOPATH" -output=".\bin\%name%_{{.OS}}_{{.Arch}}" .


