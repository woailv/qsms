package qsms

import (
	"bytes"
	"crypto/sha256"
	"dog/util/log"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"sync"
	"time"
)

// 基本参数
type baseReqParams struct {
	Sig  string `json:"sig"`
	Time int64  `json:"time"`
	Path string `json:"-"`
}

func (b *baseReqParams) initialize(o *info, extPath string, mobile ...string) error {
	now := time.Now().Unix()
	rands := strconv.Itoa(100000 + rand.Intn(999999-100000))
	data := "appkey=" + o.appKey + "&random=" + rands + "&time=" + strconv.FormatInt(now, 10)
	if len(mobile) == 1 {
		data += "&mobile=" + mobile[0]
	}
	hash := sha256.New()
	hash.Write([]byte(data))
	md := hash.Sum(nil)
	mdx := fmt.Sprintf("%x", md)
	b.Sig = string(mdx)
	b.Time = now
	b.Path = "https://yun.tim.qq.com/v5/tlssmssvr/" + extPath + "?sdkappid=" + o.appID + "&random=" + rands
	return nil
}

// 发送请求
func httpFetch(req, res interface{}) error {
	v := reflect.ValueOf(req).Elem()
	path := v.FieldByName("Path")
	bs, _ := json.Marshal(req)
	request, _ := http.NewRequest("POST", path.String(), bytes.NewBuffer(bs))
	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return errors.New("请求失败:" + fmt.Sprintf("%d", response.StatusCode))
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, res); err != nil {
		return err
	}
	return nil
}

// 用户参数配置
type info struct {
	appID  string
	appKey string
}

func New(appID, appKey string) *info {
	return &info{appID: appID, appKey: appKey}
}

// 添加短信签名
type signAddRes struct {
	Result int    `json:"result"`
	Msg    string `json:"errmsg"`
	Data   struct {
		Id     int    `json:"id"`     //签名 id
		Status int    `json:"status"` //签名状态，Enum{0：已通过, 1：待审核, 2：已拒绝}
		Text   string `json:"text"`   //签名内容
	} `json:"data"`
}

func (o *info) SignAdd(text, pic, remark string) (*signAddRes, error) {
	var req = struct {
		Pic    string `json:"pic"`
		Remark string `json:"remark"`
		Text   string `json:"text"`
		baseReqParams
	}{
		Pic:    pic,
		Remark: remark,
		Text:   text,
	}
	if err := req.baseReqParams.initialize(o, "add_sign"); err != nil {
		return nil, err
	}
	var res = new(signAddRes)
	return res, httpFetch(&req, res)
}

// 删除短信签名
type delRes struct {
	Result int    `json:"result"`
	Msg    string `json:"errmsg"`
}

func (o *info) SignDel(sids []int) (*delRes, error) {
	var req = struct {
		SignId []int `json:"sign_id"`
		baseReqParams
	}{
		SignId: sids,
	}
	if err := req.baseReqParams.initialize(o, "del_sign"); err != nil {
		return nil, err
	}
	var res = new(delRes)
	return res, httpFetch(&req, res)
}

// 查询签名状态
type getSignRes struct {
	Result int    `json:"result"`
	Msg    string `json:"errmsg"`
	Count  int    `json:"count"`
	Data   []struct {
		Id     int    `json:"id"`
		Reply  string `json:"reply"`
		Status int    `json:"status"`
		Text   string `json:"text"`
	} `json:"data"`
}

func (o *info) SignGet(sids []int) (*getSignRes, error) {
	var req = struct {
		SignId []int `json:"sign_id"`
		baseReqParams
	}{
		SignId: sids,
	}
	var res = new(getSignRes)
	req.baseReqParams.initialize(o, "get_sign")
	return res, httpFetch(&req, res)
}

// 模版添加
type tplAddRes struct {
	Result int    `json:"result"`
	Msg    string `json:"errmsg"`
	Data   struct {
		Id     int    `json:"id"`
		Status int    `json:"status"`
		Text   string `json:"text"`
		Type   int    `json:"type"`
	} `json:"data"`
}

func (o *info) TplAdd(text, title, remark string, tp int) (*tplAddRes, error) {
	var req = struct {
		Remark string `json:"remark"`
		Text   string `json:"text"`
		Title  string `json:"title"`
		Type   int    `json:"type"`
		baseReqParams
	}{
		Remark: remark,
		Text:   text,
		Title:  title,
		Type:   tp,
	}
	req.baseReqParams.initialize(o, "add_template")
	var res = new(tplAddRes)
	return res, httpFetch(&req, &res)
}

// 模版删除
func (o *info) TplDel(tplids []int) (*delRes, error) {
	var req = struct {
		TplId []int `json:"tpl_id"`
		baseReqParams
	}{
		TplId: tplids,
	}
	req.baseReqParams.initialize(o, "del_template")
	var res = new(delRes)
	return res, httpFetch(&req, &res)
}

// 查询模版状态
type getTplRes struct {
	Result int    `json:"result"`
	Msg    string `json:"errmsg"`
	Count  int    `json:"count"`
	Data   []struct {
		Id     int    `json:"id"`
		Reply  string `json:"reply"`
		Status int    `json:"status"`
		Text   string `json:"text"`
		Type   int    `json:"type"`
	} `json:"data"`
}

func (o *info) TplGet(tplids []int) (*getTplRes, error) {
	var req = struct {
		TplId []int `json:"tpl_id"`
		baseReqParams
	}{
		TplId: tplids,
	}
	req.baseReqParams.initialize(o, "get_template")
	res := new(getTplRes)
	return res, httpFetch(&req, res)
}

// 指定模版单发短信
type SendSMSSingleRes struct {
	Result int    `json:"result"`
	Errmsg string `json:"errmsg"`
	Fee    int    `json:"fee"`
	Sid    string `json:"sid"`
}

func (ssmssr *SendSMSSingleRes) CheckSelf() error {
	if ssmssr.Fee == 0 {
		return fmt.Errorf("发送错误,result:%d,errmsg:%s,sid:%s", ssmssr.Result, ssmssr.Errmsg, ssmssr.Sid)
	}
	return nil
}

func (o *info) SendSMSSingle(mobile, sign string, tplId int, params []string) (*SendSMSSingleRes, error) {
	var req = struct {
		Sign   string   `json:"sign"`
		TplId  int      `json:"tpl_id"`
		Params []string `json:"params"`
		Tel    struct {
			Mobile     string `json:"mobile"`
			Nationcode string `json:"nationcode"`
		} `json:"tel"`
		baseReqParams
	}{
		Sign:   sign,
		TplId:  tplId,
		Params: params,
	}
	req.Tel.Nationcode = "86"
	req.Tel.Mobile = mobile
	req.baseReqParams.initialize(o, "sendsms", mobile)
	sRes := new(SendSMSSingleRes)
	var err error
	if true {
		err = httpFetch(&req, sRes)
	} else { //调试
		log.Warning.Println("短信没有真正发出")
		sRes.Fee = 1
	}
	log.Info.Println("SendSMSSingle", sRes, err)
	return sRes, err
}

// 获取要发送短信的字数
var rg = regexp.MustCompile("{\\d}")

var textM = sync.Map{}

func (o *info) TextNum(sign string, tplId int, params []string) (int, error) {
	key := o.appID + strconv.Itoa(tplId)
	var text string
	val, ok := textM.Load(key)
	if !ok {
		tplRes, err := o.TplGet([]int{tplId})
		if err != nil {
			return 0, err
		}
		if tplRes.Msg != "" || tplRes.Result != 0 {
			return 0, fmt.Errorf("拉取模板失败,tpl_id:%d,result:%d,errmsg:%s", tplId, tplRes.Result, tplRes.Msg)
		}
		if len(tplRes.Data) != 1 {
			return 0, fmt.Errorf("拉取模板失败,tpl_id:%d,拉取到条数:%d,错误", tplId, len(tplRes.Data))
		}
		tpl := tplRes.Data[0]
		text = rg.ReplaceAllString(tpl.Text, "")
		textM.Store(key, text)
	} else {
		text = val.(string)
	}
	log.Info.Println("textPure:", text)
	totalNum := len([]rune(text))
	totalNum += len([]rune(sign)) + 2
	for _, param := range params {
		totalNum += len([]rune(param))
	}
	return totalNum, nil
}
