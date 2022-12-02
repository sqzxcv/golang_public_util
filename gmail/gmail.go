package gmail

import (
    "crypto/tls"
    "fmt"
    "net/smtp"

    "github.com/sqzxcv/glog"
)

type mail struct {
    user   string
    passwd string
}

// func check(err error) {
// 	if err != nil {
// 		// log.Panic(err)
// 		glog.Error(err)
// 	}
// }

//初始化用户名和密码
func New(u string, p string) mail {
    temp := mail{user: u, passwd: p}
    return temp
}

//标题 文本 目标邮箱
func (m mail) Send(title string, text string, toId string) error {
    auth := smtp.PlainAuth("", m.user, m.passwd, "smtp.gmail.com")

    tlsconfig := &tls.Config{
        InsecureSkipVerify: true,
        ServerName:         "smtp.gmail.com",
    }

    conn, err := tls.Dial("tcp", "smtp.gmail.com:465", tlsconfig)
    if err != nil {
        glog.Error("邮件stmp 连接失败, 原因:", err)
        return err
    }

    client, err := smtp.NewClient(conn, "smtp.gmail.com")
    if err != nil {
        glog.Error("邮件客户端创建失败, 原因:", err)
        return err
    }

    if err = client.Auth(auth); err != nil {
        glog.Error("邮件客户端登录失败, 原因:", err)
        return err
    }

    if err = client.Mail(m.user); err != nil {
        glog.Error("邮件发送失败, 原因:", err)
        return err
    }

    if err = client.Rcpt(toId); err != nil {
        glog.Error("邮件发送失败, 原因:", err)
        return err
    }

    w, err := client.Data()
    if err != nil {
        glog.Error("邮件发送失败, 原因:", err)
        return err
    }

    msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", m.user, toId, title, text)

    _, err = w.Write([]byte(msg))
    if err != nil {
        glog.Error("邮件发送失败, 原因:", err)
        return err
    }

    err = w.Close()
    if err != nil {
        glog.Error("邮件发送失败, 原因:", err)
        return err
    }

    client.Quit()
    return nil
}
