package tools

import "time"

// Imap imap结构体 供远程调用
type Imap struct{}

// MailServer 邮件服务
type MailServer struct {
	Server, Email, Password string
}

// MailItem 邮件
type MailItem struct {
	Subject string
	Fid     string
	ID      uint32
	UID     uint32
	From    []*MailAddress
	To      []*MailAddress
	Body    string
	Date    time.Time
	Flags   []string
}

// MailAddress 邮件地址＋昵称
type MailAddress struct {
	Personal, Address string
}

//MailPageList 邮件分页
type MailPageList struct {
	MailItems []*MailItem
}
