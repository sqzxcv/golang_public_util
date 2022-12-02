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

func WriteToFile(content string, filepath string) error {
    err := ioutil.WriteFile(filepath, []byte(content), os.ModePerm)
    if err != nil {
        glog.Error("文件写入出错, 原因:", err.Error())
        return err
    }
    return nil
}
