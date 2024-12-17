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
)

// default http client with 5 seconds timeout
var RequestHttpClient = http.Client{Timeout: time.Second * 5}

// 发送请求
func SendRequest(authHeaderString string, url string, bodyMap map[string]string) (string, error) {
	bodyString, err := json.Marshal(bodyMap)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyString))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Authorization", authHeaderString)
	response, err := RequestHttpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}

// 验证签名
func VerifyRsaSign(content string, sign string, publicKey string) error {
	publicKeyByte, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return err
	}

	pub, err := x509.ParsePKIXPublicKey(publicKeyByte)
	if err != nil {
		return err

	}
	hashed := sha256.Sum256([]byte(content))
	signature, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return err
	}

	return rsa.VerifyPKCS1v15(pub.(*rsa.PublicKey), crypto.SHA256, hashed[:], signature)
}
