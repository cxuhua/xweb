package xweb

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
)

var (
	TokenKey      []byte       = []byte("BM9dkmHWcJAqalwiuylIX4HcDElwDd7uauDsdWr646v")
	tokenAesBlock cipher.Block = nil
)

func HMACString(data, secret string) string {
	return HMACBytes([]byte(data), secret)
}

func HMACBytes(data []byte, secret string) string {
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

func SHA1String(data string) string {
	return SHA1Bytes([]byte(data))
}

func SHA1Bytes(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func MD5String(s string) string {
	return MD5Bytes([]byte(s))
}

func MD5Bytes(data []byte) string {
	m := md5.New()
	m.Write(data)
	return hex.EncodeToString(m.Sum(nil))
}

//整理key为 16 24 or 32
func TrimAESKey(key []byte) ([]byte, error) {
	size := len(key) / 8
	if size <= 2 {
		size = 2
	}
	if size > 4 {
		size = 4
	}
	iLen := size * 8
	ikey := make([]byte, iLen)
	if len(key) > iLen {
		copy(ikey[0:], key[:iLen])
	} else {
		copy(ikey[0:], key)
	}
	return ikey, nil
}

//创建加密算法
func NewAESChpher(key []byte) (cipher.Block, error) {
	ikey, err := TrimAESKey(key)
	if err != nil {
		return nil, err
	}
	return aes.NewCipher(ikey)
}

//检测最后几个字节是否是加密
func bytesEquInt(data []byte, n byte) bool {
	l := len(data)
	if l == 0 {
		return false
	}
	for i := 0; i < l; i++ {
		if data[i] != n {
			return false
		}
	}
	return true
}

// AES with IV解密
func AesDecryptWithIV(block cipher.Block, data []byte, iv []byte) ([]byte, error) {
	if block == nil {
		return nil, errors.New("block nil")
	}
	bytes := len(data)
	if bytes < 32 || bytes%aes.BlockSize != 0 {
		return nil, errors.New("decrypt data length error")
	}
	//16 bytes iv
	dd := data[0:]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(dd, dd)
	l := len(dd)
	if n := dd[l-1]; n <= aes.BlockSize {
		x := l - int(n)
		if bytesEquInt(dd[x:], n) {
			dd = dd[:x]
		}
	}
	return dd, nil
}

// AES加密
func AesEncrypt(block cipher.Block, data []byte) ([]byte, error) {
	if block == nil {
		return nil, errors.New("block nil")
	}
	//随机生成iv
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	dl := len(data)
	l := (dl/aes.BlockSize)*aes.BlockSize + aes.BlockSize
	if dl%aes.BlockSize == 0 {
		l = dl
	}
	//add iv length
	dd := make([]byte, l+aes.BlockSize)
	n := l - dl
	//copy iv to dd
	copy(dd[0:], iv)
	//copy data to dd
	copy(dd[aes.BlockSize:], data)
	//fill end bytes
	for i := 0; i < n; i++ {
		dd[dl+i+aes.BlockSize] = byte(n)
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(dd[aes.BlockSize:], dd[aes.BlockSize:])
	return dd, nil
}

// AES解密
func AesDecrypt(block cipher.Block, data []byte) ([]byte, error) {
	if block == nil {
		return nil, errors.New("block nil")
	}
	bytes := len(data)
	if bytes < 32 || bytes%aes.BlockSize != 0 {
		return nil, errors.New("decrypt data length error")
	}
	//16 bytes iv
	iv := data[:aes.BlockSize]
	dd := data[aes.BlockSize:]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(dd, dd)
	l := len(dd)
	if n := dd[l-1]; n <= aes.BlockSize {
		x := l - int(n)
		if bytesEquInt(dd[x:], n) {
			dd = dd[:x]
		}
	}
	return dd, nil
}

//动态创建编码器
func makeTokenAesBlock() error {
	if tokenAesBlock != nil {
		return nil
	}
	block, err := NewAESChpher(TokenKey)
	if err != nil {
		return err
	}
	tokenAesBlock = block
	return nil
}

//token加密
func TokenEncrypt(token string) (string, error) {
	if err := makeTokenAesBlock(); err != nil {
		return token, err
	}
	d, err := AesEncrypt(tokenAesBlock, []byte(token))
	if err != nil {
		return token, err
	}
	return base64.URLEncoding.EncodeToString(d), nil
}

//token解密
func TokenDecrypt(value string) (string, error) {
	if err := makeTokenAesBlock(); err != nil {
		return value, err
	}
	d, err := base64.URLEncoding.DecodeString(value)
	if err != nil {
		return value, err
	}
	s, err := AesDecrypt(tokenAesBlock, d)
	if err != nil {
		return value, err
	}
	return string(s), nil
}
