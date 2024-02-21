package strs

import (
	"encoding/json"
	"github.com/sqzxcv/glog"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// CompareVersionStr v1 > v2 return 1;
//
// v1 < v2 return -1;
//
// v1 = v2 return 0
func CompareVersionStr(v1 string, v2 string) int {
	versionA := strings.Split(v1, ".")
	versionB := strings.Split(v2, ".")

	for i := len(versionA); i < 4; i++ {
		versionA = append(versionA, "0")
	}
	for i := len(versionB); i < 4; i++ {
		versionB = append(versionB, "0")
	}
	for i := 0; i < 4; i++ {
		version1, _ := strconv.Atoi(versionA[i])
		version2, _ := strconv.Atoi(versionB[i])
		if version1 == version2 {
			continue
		} else if version1 > version2 {
			return 1
		} else {
			return -1
		}
	}
	return 0
}

func IsContain(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

// TimeFormat format 包含字符串: "2006/01/02 15:04:05"
func TimeFormat(date time.Time, format string) string {
	return date.Format(format)
}

func VerifyEmailFormat(email string) bool {
	pattern := `\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*` //匹配电子邮箱
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(email)
}

func Json2Map(content string) (result map[string]interface{}, err error) {
	JsonBytes := []byte(content)
	result = make(map[string]interface{})
	err = json.Unmarshal(JsonBytes, &result)

	if err != nil {
		glog.Error("json转化失败:", err.Error())
		return nil, err
	}
	return result, nil
}

func MapStr2JSONStr(value map[string]string) (string, error) {
	data, err := json.Marshal(value)

	if err != nil {
		return "", err
	}
	return string(data), nil
}

func WriteToFile(content string, filepath string) error {
	err := ioutil.WriteFile(filepath, []byte(content), os.ModePerm)
	if err != nil {
		glog.Error("文件写入出错, 原因:", err.Error())
		return err
	}
	return nil
}

// GetFirstSubStr 获取字符串的前100个有效字符, 支持获取非Ascii字符, 确保返回有效字符, 避免返回乱码
func GetFirstSubStr(str string, max int) string {
	var result string
	count := 0
	for i, w := 0, 0; i < len(str) && count < max; i += w {
		runeValue, width := utf8.DecodeRuneInString(str[i:])
		if width > 0 {
			w = width
			result += string(runeValue)
			count++
		} else {
			break
		}
	}
	return result
}
