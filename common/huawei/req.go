package huawei

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// default http client with 5 seconds timeout
var RequestHttpClient = http.Client{Timeout: time.Second * 10}

// 发送请求
func SendRequest(authHeaderString string, url string, bodyMap map[string]string) (string, error) {
	bodyString, err := json.Marshal(bodyMap)
	if err != nil {
		logx.Errorf("json.Marshal, err: %v", err)
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyString))
	if err != nil {
		logx.Errorf("http.NewRequest, err: %v", err)
		return "", err
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Authorization", authHeaderString)
	response, err := RequestHttpClient.Do(req)
	if err != nil {
		logx.Errorf("RequestHttpClient.Do, err: %v", err)
		return "", err
	}

	defer response.Body.Close()
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logx.Errorf("ioutil.ReadAll, err: %v", err)
		return "", err
	}

	return string(bodyBytes), nil
}

// 验证签名
// content结果字符串
//
// sign 签名字符串
//
// 应用公钥 publicKey
func VerifyRsaSign(content, sign, publicKey string) error {
	publicKeyByte, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		logx.Errorf("StdEncoding.DecodeString, err:%v, publicKey raw data:%s", err, publicKey)
		return err
	}

	pub, err := x509.ParsePKIXPublicKey(publicKeyByte)
	if err != nil {
		logx.Errorf("x509.ParsePKIXPublicKey, err:%v, publicKey raw data:%s", err, publicKey)
		return err
	}

	hashed := sha256.Sum256([]byte(content))
	signature, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		logx.Errorf("StdEncoding.DecodeString, err:%v, content:%s, sign:%s, publicKey:%s", err, content, sign, publicKey)
		return err
	}

	return rsa.VerifyPKCS1v15(pub.(*rsa.PublicKey), crypto.SHA256, hashed[:], signature)
}
