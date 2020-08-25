package tools

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/axgle/mahonia"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
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

//GetMailNum 获取邮件总数
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

//GetFolders 获取邮件夹
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

//GetFolderMail 获取邮件夹邮件
func (i *Imap) GetFolderMail(param GetMessagesType, reply *[]MailItem) error {
	var c *client.Client
	server := param.Server.Server
	email := param.Server.Email
	password := param.Server.Password
	folder := param.Folder

	// defer c.Logout()
	c = connect(server, email, password)
	if c == nil {
		return nil
	}

	mbox, _ := c.Select(folder, true)

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
			mailAddr := new(mail.Address)
			mailAddr.Name = from.PersonalName
			mailAddr.Address = from.MailboxName + "@" + from.HostName
			mailitem.From = append(mailitem.From, mailAddr)
		}
		for _, to := range msg.Envelope.To {
			mailAddr := new(mail.Address)
			mailAddr.Name = to.PersonalName
			mailAddr.Address = to.MailboxName + "@" + to.HostName
			mailitem.To = append(mailitem.To, mailAddr)
		}

		*reply = append(*reply, *mailitem)
	}
	return nil
}

//GetMessagesFlag 获取多个邮件的flag。
func (i *Imap) GetMessagesFlag(param GetMessagesType, reply *[]MailFlagtype) error {
	var c *client.Client
	server := param.Server.Server
	email := param.Server.Email
	password := param.Server.Password

	// defer c.Logout()
	c = connect(server, email, password)
	if c == nil {
		return nil
	}

	mbox, _ := c.Select(param.Folder, true)
	start := mbox.Messages - uint32((param.Page-1)*param.Limit)
	end := start - uint32(param.Limit)
	if start <= uint32(param.Limit) {
		end = 1
	}

	seqset := new(imap.SeqSet)
	seqset.AddRange(start, end)

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	items := make([]imap.FetchItem, 0)
	items = append(items, imap.FetchItem(imap.FetchFlags))
	items = append(items, imap.FetchItem(imap.FetchUid))
	go func() {
		done <- c.Fetch(seqset, items, messages)
	}()

	for msg := range messages {
		flagitem := new(MailFlagtype)
		flagitem.UID = msg.Uid
		flagitem.Flags = msg.Flags
		*reply = append(*reply, *flagitem)
	}
	return nil
}

//GetRecent 获取最新的邮件
func (i *Imap) GetRecent(param GetMessagesType, reply *[]MailItem) error {
	var c *client.Client
	server := param.Server.Server
	email := param.Server.Email
	password := param.Server.Password
	folder := param.Folder

	c = connect(server, email, password)
	if c == nil {
		return nil
	}

	mbox, err := c.Select(folder, true)

	if err != nil {
		log.Fatal(err)
	}

	if mbox.Recent == 0 {
		return nil
	}

	// Set search criteria
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	ids, err := c.Search(criteria)
	if err != nil {
		log.Fatal(err)
	}

	if len(ids) > 0 {
		seqset := new(imap.SeqSet)
		seqset.AddNum(ids...)

		messages := make(chan *imap.Message, 10)
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
				mailAddr := new(mail.Address)
				mailAddr.Name = from.PersonalName
				mailAddr.Address = from.MailboxName + "@" + from.HostName
				mailitem.From = append(mailitem.From, mailAddr)
			}
			for _, to := range msg.Envelope.To {
				mailAddr := new(mail.Address)
				mailAddr.Name = to.PersonalName
				mailAddr.Address = to.MailboxName + "@" + to.HostName
				mailitem.To = append(mailitem.To, mailAddr)
			}

			*reply = append(*reply, *mailitem)
		}
		if err := <-done; err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

//GetMessage 获取邮件详情
func (i *Imap) GetMessage(param GetMessageType, reply *MailItem) error {
	t1 := time.Now()
	var c *client.Client
	server := param.Server.Server
	email := param.Server.Email
	password := param.Server.Password
	folder := param.Folder
	mailpath := param.Mailpath
	uid := param.UID

	imagesmap := make(map[string]string)

	// defer c.Logout()
	c = connect(server, email, password)
	if c == nil {
		return nil
	}
	// 选择 文件夹
	mbox, err := c.Select(folder, false)
	if err != nil {
		log.Println(err)
		return nil
	}
	t2 := time.Now()
	fmt.Println("打开连接用了：", t2.Sub(t1))
	// 获取邮件
	if mbox.Messages == 0 {
		log.Println("No message in mailbox")
		return nil
	}
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	// 获取邮件 body
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem()}

	messages := make(chan *imap.Message, 1)
	go func() {
		if err := c.UidFetch(seqSet, items, messages); err != nil {
			log.Fatal(err)
		}
	}()

	msg := <-messages
	if msg == nil {
		log.Println("Server didn't returned message")
		return nil
	}
	t3 := time.Now()
	fmt.Println("读取邮件用了：", t3.Sub(t2))
	fmt.Println("一共用了：", t3.Sub(t1))
	r := msg.GetBody(section)
	if r == nil {
		log.Println("Server didn't returned message body")
		return nil
	}

	mailitem := new(MailItem)
	mailitem.Attachments = 0

	mr, _ := mail.CreateReader(r)

	var bodyMap = make(map[string]string)
	bodyMap["text/plain"] = ""
	bodyMap["text/html"] = ""

	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Println(err) //TODO 写日志
		}
		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			b, _ := ioutil.ReadAll(p.Body)
			//正文
			ct := p.Header.Get("Content-Type")
			if strings.Contains(ct, "text/plain") {
				b, _ = parseText(b) // 只获取最新回复的内容，以前的邮件对话之类的冗余内容就不要了。
				bodyMap["text/plain"] += Encoding(string(b), ct)
			} else if strings.Contains(ct, "text/html") {
				bodyMap["text/html"] += string(b)
				// ioutil.WriteFile(mailpath+"/"+fmt.Sprint(uid)+".html", b, 0777)
			} else {
				// 内联附件 主要是内容中的图片
				contentID := h.Get("Content-ID")
				fmt.Println(contentID)
				_, pr, _ := h.ContentType()
				// 获取图片 cid
				reg := regexp.MustCompile(`[^<^>]+`)
				contentID = reg.FindString(contentID)
				// 获取图片后缀
				s := strings.Split(pr["name"], ".")
				ext := s[len(s)-1]
				base64image := "data:image/" + ext + ";base64," + base64.StdEncoding.EncodeToString(b)
				imagesmap[contentID] = base64image
			}

		case *mail.AttachmentHeader:
			// 附件处理
			filename, _ := h.Filename()
			b, _ := ioutil.ReadAll(p.Body)
			os.MkdirAll(mailpath+"/"+fmt.Sprint(uid), os.ModePerm)
			ioutil.WriteFile(mailpath+"/"+fmt.Sprint(uid)+"/"+filename, b, 0777) //附件保存目录，保存在已邮件的 UID 为名字的文件夹里
			mailitem.Attachments++
		}
	}
	// 写入 html 邮件内容到磁盘， html 中 <img src="cid:xxx" /> 处理成 base64
	if bodyMap["text/html"] == "" {
		bodyMap["text/html"] = bodyMap["text/plain"]
	}
	os.MkdirAll(mailpath, os.ModePerm) //创建文件夹
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyMap["text/html"]))
	if err == nil {
		doc.Find("img").Each(func(i int, s *goquery.Selection) {
			//解析<img>标签
			v, _ := s.Attr("src")
			fmt.Println(v)
			cid := strings.Split(v, ":")
			fmt.Println(cid)
			if cid[0] == "cid" {
				s.SetAttr("src", imagesmap[cid[len(cid)-1]]) //修改标签的内容
			}
		})
		html, _ := doc.Html()
		ioutil.WriteFile(mailpath+"/"+fmt.Sprint(uid)+".html", []byte(html), 0777)
		mailitem.HTMLBody = html
	}

	// recent := false
	// //标记该邮件为已读
	// for i := range msg.Flags {
	// 	if strings.Contains(msg.Flags[i], "Recent") { //未读的邮件才设置状态
	// 		recent = true
	// 		break
	// 	}
	// 	log.Println(msg.Flags[i])
	// }
	// if recent {
	// 	item := imap.FormatFlagsOp(imap.AddFlags, true)
	// 	flags := []interface{}{imap.SeenFlag}
	// 	go func() {
	// 		c.UidStore(seqSet, item, flags, nil)
	// 	}()
	// }

	// TEXT 内容返回给 PHP
	if bodyMap["text/plain"] == "" {
		bodyMap["text/plain"] = bodyMap["text/html"]
	}
	mailitem.Body = bodyMap["text/plain"]
	*reply = *mailitem

	return nil
}

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
