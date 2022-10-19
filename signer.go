package xweb

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

var rs string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

//获得随机字符串
func RandStr() string {
	s := ""
	idxs := [32]byte{}
	err := binary.Read(rand.Reader, binary.BigEndian, &idxs)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 32; i++ {
		v := uint(idxs[i]) % uint(len(rs))
		s += string(rs[v])
	}
	return s
}

//参数签名处理,暂时只支持json签名,文件数据签名

type ISigner interface {
	//创建签名
	Create(ads ...string) (string, string, string, error)
	//验证签名
	Verify(data []byte, sign string, ts string, nonce string, ads ...string) error
	//添加签名数据
	Write(data []byte) error
}

var (
	UseSigner ISigner = nil
)

//签名用http头
const (
	NF_Nonce     = "NF-Nonce"     //随机字符串 32字节
	NF_Timestamp = "NF-Timestamp" //时间戳
	NF_Signature = "NF-Signature" //签名
)

type standsigner struct {
	key string
	buf *bytes.Buffer
}

func (ss *standsigner) getAds(ads ...string) string {
	return strings.Join(ads, "")
}

func (ss *standsigner) Create(ads ...string) (string, string, string, error) {
	nonce := RandStr()
	ts := fmt.Sprintf("%d", time.Now().Unix())
	sign, err := ss.getSign(ts, nonce, ads...)
	if err != nil {
		return "", "", "", err
	}
	return sign, ts, nonce, nil
}

//sha256(host+post+path+sha256(body)+key)
func (ss *standsigner) getSign(ts string, nonce string, ads ...string) (string, error) {
	adss := ss.getAds(ads...)
	if adss == "" {
		return "", fmt.Errorf("adss emtpy")
	}
	sbb := &bytes.Buffer{}
	strs := adss + nonce + ts
	_, err := sbb.Write([]byte(strs))
	if err != nil {
		return "", err
	}
	bhash := sha256.Sum256(ss.buf.Bytes())
	_, err = sbb.Write([]byte(hex.EncodeToString(bhash[:])))
	if err != nil {
		return "", err
	}
	_, err = sbb.Write([]byte(ss.key))
	if err != nil {
		return "", err
	}
	bb := sha256.Sum256(sbb.Bytes())
	sign := hex.EncodeToString(bb[:])
	return sign, nil
}

func (ss *standsigner) Verify(data []byte, sign string, ts string, nonce string, ads ...string) error {
	_, _ = ss.buf.Write(data)
	if sign == "" || ts == "" || nonce == "" {
		return fmt.Errorf("sign args empty")
	}
	ssg, err := ss.getSign(ts, nonce, ads...)
	if err != nil {
		return err
	}
	if ssg != sign {
		return fmt.Errorf("sign %s error", sign)
	}
	return nil
}

func (ss *standsigner) Write(data []byte) error {
	_, err := ss.buf.Write(data)
	return err
}

func NewStandSigner(key string) ISigner {
	return &standsigner{key: key, buf: &bytes.Buffer{}}
}
