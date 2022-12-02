package qqmail

import (
    "github.com/sqzxcv/glog"
    "gopkg.in/gomail.v2"
)

type mail struct {
    user   string
    passwd string
}

//初始化用户名和密码
func New(u string, p string) mail {
    temp := mail{user: u, passwd: p}
    return temp
}

//标题 文本 目标邮箱
func (m mail) Send(title string, text string, toId string) error {

    msg := gomail.NewMessage()

    //发送人
    msg.SetHeader("From", m.user)
    //接收人
    msg.SetHeader("To", toId)
    //抄送人
    //m.SetAddressHeader("Cc", "xxx@qq.com", "xiaozhujiao")
    //主题
    msg.SetHeader("Subject", title)
    //内容
    msg.SetBody("text/html", text)
    //附件
    //m.Attach("./myIpPic.png")

    //拿到token，并进行连接,第4个参数是填授权码
    d := gomail.NewDialer("smtp.qq.com", 587, m.user, m.passwd)

    // 发送邮件
    if err := d.DialAndSend(msg); err != nil {
        glog.Error("DialAndSend err %v:", err)
        return err
    }
    glog.Info("send mail success\n")
    return nil
}

