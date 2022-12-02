package Path

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/sqzxcv/glog"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetCurrentDirectory 获取运行exe程序时的目录
func GetCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0])) //返回绝对路径  filepath.Dir(os.Args[0])去除最后一个元素的路径
	if err != nil {
		glog.Error(err)
	}
	return strings.Replace(dir, "\\", "/", -1) //将\替换成/
}

// GetExeDirectory 获取可执行文件所在目录
func GetExeDirectory() string {

	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	index := strings.LastIndex(path, string(os.PathSeparator))
	ret := path[:index]

	return strings.Replace(ret, "\\", "/", -1) //将\替换成/
}

// PathExist 判断文件是否存在
func PathExist(_path string) bool {
	_, err := os.Stat(_path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

// CreateDir 创建目录
func CreateDir(path string) error {

	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		fmt.Printf("mkdir failed![%v]\n", err)
	} else {
		fmt.Printf("mkdir success!\n")
	}
	return err
}

func FileHash(localPath string) string {
	file, err := os.Open(localPath)
	defer file.Close()
	if err != nil {
		glog.Error("读取文件失败！")
	}
	md5 := md5.New()
	if _, err := io.Copy(md5, file); err != nil {
		glog.Error(err)
	}
	return hex.EncodeToString(md5.Sum(nil))
}
