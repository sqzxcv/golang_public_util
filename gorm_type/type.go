package gorm_type

import (
    "database/sql/driver"
    "encoding/json"
    "errors"
    "fmt"
    "strings"
)

type JSON map[string]interface{}

func (j *JSON) Scan(value interface{}) error {
    bytes, ok := value.([]byte)
    if !ok {
        return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
    }

    if len(bytes) == 0 {
        return nil
    }

    result := map[string]interface{}{}
    err := json.Unmarshal(bytes, &result)
    *j = result
    return err
}

// Value 实现 driver.Valuer 接口，Value 返回 json value
func (j JSON) Value() (driver.Value, error) {
    if len(j) == 0 {
        return nil, nil
    }
    jsonStr, err := json.Marshal(j)
    if err != nil {
        return nil, err
    }
    return jsonStr, nil
}

type StringArr []string

func (j *StringArr) Scan(value interface{}) error {
    if value == nil {
        return nil
    }
    b, ok := value.([]byte)
    if !ok {
        msg := fmt.Sprint("content 不是有效的字符串:", value)
        return errors.New(msg)
    }
    arr := strings.Split(string(b), ",")

    for _, item := range arr {
        *j = append(*j, strings.TrimSpace(item))
    }
    return nil
}

// Value 实现 driver.Valuer 接口，Value 返回 json value
func (j StringArr) Value() (driver.Value, error) {
    if len(j) == 0 {
        return nil, nil
    }
    content := strings.Join(j, ",")
    return content, nil
}
