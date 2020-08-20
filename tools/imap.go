package tools

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"strconv"
	"strings"

	"github.com/axgle/mahonia"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// CheckEmailPassword 验证邮箱密码
func (i *Imap) CheckEmailPassword(param map[string]string, reply *bool) error {
	if !strings.Contains(param["server"], ":") {
		*reply = false
		return nil
	}
	var c *client.Client
	serverSlice := strings.Split(param["server"], ":")
	port, _ := strconv.Atoi(serverSlice[1])
	if port != 993 && port != 143 {
		*reply = false
		return nil
	}

	// 不要忘了退出
	//defer c.Logout()

	// 登陆
	c = connect(param["server"], param["email"], param["password"])
	if c == nil {
		*reply = false
		return nil
	}
	*reply = true
	return nil
}

//获取连接
func connect(server string, email string, password string) *client.Client {
	var c *client.Client
	var err error
	serverSlice := strings.Split(server, ":")
	uri := serverSlice[0]
	port, _ := strconv.Atoi(serverSlice[1])
	if port != 993 && port != 143 {
		return nil
	}
	if port == 993 {
		c, err = client.DialTLS(fmt.Sprintf("%s:%d", uri, port), nil)
	} else {
		c, err = client.Dial(fmt.Sprintf("%s:%d", uri, port))
	}
	if err != nil {
		return nil
	}

	// 登陆
	if err := c.Login(email, password); err != nil {
		return nil
	}
	return c
}

//获取邮件总数
func GetMailNum(server string, email string, password string) map[string]int {
	var c *client.Client
	//defer c.Logout()
	c = connect(server, email, password)
	if c == nil {
		return nil
	}
	// 列邮箱
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()
	//// 存储邮件夹
	var folders = make(map[string]int)
	for m := range mailboxes {
		folders[m.Name] = 0
	}
	for m, _ := range folders {
		mbox, _ := c.Select(m, true)
		if mbox != nil {
			folders[m] = int(mbox.Messages)
		}
	}
	return folders
}

//获取邮件夹
func GetFolders(server string, email string, password string, folder string) map[string]int {
	var c *client.Client
	//defer c.Logout()
	c = connect(server, email, password)
	if c == nil {
		return nil
	}
	// 列邮箱
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()
	// 存储邮件夹
	var folders = make(map[string]int)
	for m := range mailboxes {
		folders[m.Name] = 0
	}
	for m, _ := range folders {
		if m == folder {
			mbox, _ := c.Select(m, true)
			if mbox != nil {
				folders[m] = int(mbox.Messages)
			}
			break
		}
	}
	//log.Println(folders)
	return folders
}

//获取邮件夹邮件
func (i *Imap) GetFolderMail(param map[string]string, reply *[]*MailItem) error {
	var c *client.Client
	server := param["server"]
	email := param["email"]
	password := param["password"]
	folder := param["folder"]
	// currentPage, _ := strconv.ParseUint(param["currentPage"], 10, 32)
	// pagesize, _ := strconv.ParseUint(param["pagesize"], 10, 32)

	// defer c.Logout()
	c = connect(server, email, password)
	if c == nil {
		return nil
	}

	mbox, _ := c.Select(folder, true)
	// to := mbox.Messages - uint32((currentPage-1)*pagesize)
	// from := to - uint32(pagesize)
	// if to <= uint32(pagesize) {
	// 	from = 1
	// }

	seqset := new(imap.SeqSet)
	seqset.AddRange(1, mbox.Messages)

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	items := make([]imap.FetchItem, 0)
	items = append(items, imap.FetchItem(imap.FetchFlags))
	items = append(items, imap.FetchItem(imap.FetchUid))
	items = append(items, imap.FetchItem(imap.FetchEnvelope))
	go func() {
		done <- c.Fetch(seqset, items, messages)
	}()

	dec := GetDecoder()

	for msg := range messages {
		log.Println(msg.Uid)
		subject, err := dec.Decode(msg.Envelope.Subject)
		if err != nil {
			subject, _ = dec.DecodeHeader(msg.Envelope.Subject)
		}
		var mailitem = new(MailItem)
		mailitem.Subject = subject
		mailitem.ID = msg.SeqNum
		mailitem.UID = msg.Uid
		mailitem.Fid = folder
		mailitem.Date = msg.Envelope.Date
		mailitem.Flags = msg.Flags
		for _, from := range msg.Envelope.From {
			mailAddr := new(MailAddress)
			mailAddr.Personal = from.PersonalName
			mailAddr.Address = from.MailboxName + "@" + from.HostName
			mailitem.From = append(mailitem.From, mailAddr)
		}
		for _, to := range msg.Envelope.To {
			mailAddr := new(MailAddress)
			mailAddr.Personal = to.PersonalName
			mailAddr.Address = to.MailboxName + "@" + to.HostName
			mailitem.To = append(mailitem.To, mailAddr)
		}

		*reply = append(*reply, mailitem)
	}
	return nil
}

// func (i *Imap) GetMessage(param map[string]string, reply *MailItem) error {
// 	var c *client.Client
// 	//defer c.Logout()
// 	c = connect(param["server"], param["email"], param["password"])
// 	if c == nil {
// 		//return nil
// 	}
// 	// Select INBOX
// 	mbox, err := c.Select("INBOX", false)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// Get the last message
// 	if mbox.Messages == 0 {
// 		log.Fatal("No message in mailbox")
// 	}
// 	log.Println("邮件数：", mbox.Messages)
// 	seqSet := new(imap.SeqSet)
// 	id, _ := strconv.ParseUint(param["id"], 10, 32)
// 	seqSet.AddNum(uint32(id))

// 	// Get the whole message body
// 	section := &imap.BodySectionName{}
// 	items := []imap.FetchItem{section.FetchItem()}

// 	messages := make(chan *imap.Message, 1)
// 	go func() {
// 		if err := c.Fetch(seqSet, items, messages); err != nil {
// 			log.Fatal(err)
// 		}
// 	}()

// 	msg := <-messages
// 	if msg == nil {
// 		log.Fatal("Server didn't returned message")
// 	}

// 	r := msg.GetBody(section)

// 	if r == nil {
// 		log.Fatal("Server didn't returned message body")
// 	}
// 	var mailitem = new(MailItem)

// 	// Create a new mail reader
// 	mr, _ := mail.CreateReader(r)

// 	// Print some info about the message
// 	header := mr.Header
// 	date, _ := header.Date()

// 	mailitem.Date = date.String()

// 	var f string
// 	dec := GetDecoder()

// 	if from, err := header.AddressList("From"); err == nil {
// 		for _, address := range from {
// 			fromStr := address.String()
// 			temp, _ := dec.DecodeHeader(fromStr)
// 			f += " " + temp
// 		}
// 	}
// 	mailitem.From = f
// 	log.Println("From:", mailitem.From)

// 	var t string
// 	if to, err := header.AddressList("To"); err == nil {
// 		log.Println("To:", to)
// 		for _, address := range to {
// 			toStr := address.String()
// 			temp, _ := dec.DecodeHeader(toStr)
// 			t += " " + temp
// 		}
// 	}
// 	mailitem.To = t

// 	subject, _ := header.Subject()
// 	s, err := dec.Decode(subject)
// 	if err != nil {
// 		s, _ = dec.DecodeHeader(subject)
// 	}
// 	log.Println("Subject:", s)
// 	mailitem.Subject = s
// 	// Process each message's part
// 	var bodyMap = make(map[string]string)
// 	bodyMap["text/plain"] = ""
// 	bodyMap["text/html"] = ""

// 	for {
// 		p, err := mr.NextPart()
// 		if err == io.EOF {
// 			break
// 		} else if err != nil {
// 			//log.Fatal(err)
// 		}
// 		switch h := p.Header.(type) {
// 		case *mail.InlineHeader:
// 			// This is the message's text (can be plain-text or HTML)

// 			b, _ := ioutil.ReadAll(p.Body)
// 			ct := p.Header.Get("Content-Type")
// 			if strings.Contains(ct, "text/plain") {
// 				bodyMap["text/plain"] += Encoding(string(b), ct)
// 			} else {
// 				bodyMap["text/html"] += Encoding(string(b), ct)
// 			}
// 			//body,_:=dec.Decode(string(b))
// 		case *mail.AttachmentHeader:
// 			// This is an attachment
// 			filename, _ := h.Filename()
// 			log.Println("Got attachment: ", filename)
// 		}

// 	}
// 	if bodyMap["text/html"] != "" {
// 		mailitem.Body = bodyMap["text/html"]
// 	} else {
// 		mailitem.Body = bodyMap["text/plain"]
// 	}
// 	// *reply = *mailitem
// 	//log.Println(mailitem.Body)
// 	return nil
// }

func GetDecoder() *mime.WordDecoder {
	dec := new(mime.WordDecoder)
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		charset = strings.ToLower(charset)
		switch charset {
		case "gb2312":
			content, err := ioutil.ReadAll(input)
			if err != nil {
				return nil, err
			}
			//ret:=bytes.NewReader(content)
			//ret:=transform.NewReader(bytes.NewReader(content), simplifiedchinese.HZGB2312.NewEncoder())

			utf8str := ConvertToStr(string(content), "gbk", "utf-8")
			t := bytes.NewReader([]byte(utf8str))
			//ret:=utf8.DecodeRune(t)
			//log.Println(ret)
			return t, nil
		case "gbk":
			content, err := ioutil.ReadAll(input)
			if err != nil {
				return nil, err
			}
			//ret:=bytes.NewReader(content)
			//ret:=transform.NewReader(bytes.NewReader(content), simplifiedchinese.HZGB2312.NewEncoder())

			utf8str := ConvertToStr(string(content), "gbk", "utf-8")
			t := bytes.NewReader([]byte(utf8str))
			//ret:=utf8.DecodeRune(t)
			//log.Println(ret)
			return t, nil
		case "gb18030":
			content, err := ioutil.ReadAll(input)
			if err != nil {
				return nil, err
			}
			//ret:=bytes.NewReader(content)
			//ret:=transform.NewReader(bytes.NewReader(content), simplifiedchinese.HZGB2312.NewEncoder())

			utf8str := ConvertToStr(string(content), "gbk", "utf-8")
			t := bytes.NewReader([]byte(utf8str))
			//ret:=utf8.DecodeRune(t)
			//log.Println(ret)
			return t, nil
		default:
			return nil, fmt.Errorf("unhandle charset:%s", charset)

		}
	}
	return dec
}

// ConvertToStr 任意编码转特定编码
func ConvertToStr(src string, srcCode string, tagCode string) string {
	result := mahonia.NewDecoder(srcCode).ConvertString(src)
	//srcCoder := mahonia.NewDecoder(srcCode)
	//srcResult := srcCoder.ConvertString(src)
	//tagCoder := mahonia.NewDecoder(tagCode)
	//_, cdata, _ := tagCoder.Translate([]byte(srcResult), true)
	//result := string(cdata)
	return result
}

// Encoding 转换编码
func Encoding(html string, ct string) string {
	e, name := DetermineEncoding(html)
	if name != "utf-8" {
		html = ConvertToStr(html, "gbk", "utf-8")
		e = unicode.UTF8
	}
	r := strings.NewReader(html)

	utf8Reader := transform.NewReader(r, e.NewDecoder())
	//将其他编码的reader转换为常用的utf8reader
	all, _ := ioutil.ReadAll(utf8Reader)
	return string(all)
}
func DetermineEncoding(html string) (encoding.Encoding, string) {
	e, name, _ := charset.DetermineEncoding([]byte(html), "")
	return e, name
}
