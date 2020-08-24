package tools

import (
	"time"

	"github.com/emersion/go-message/mail"
)

// Imap imap结构体 供远程调用
type Imap struct{}

// MailServer 邮件服务
type MailServer struct {
	Server, Email, Password string
}

// MailItem 邮件
type MailItem struct {
	Subject     string
	Fid         string
	ID          uint32
	UID         uint32
	From        []*mail.Address
	To          []*mail.Address
	Body        string
	HTMLBody    string
	Date        time.Time
	Flags       []string
	Attachments int
}

//MailPageList 邮件分页
type MailPageList struct {
	MailItems []*MailItem
}

// Attachment .
type Attachment struct {
	Filename string
	Content  []byte
}
