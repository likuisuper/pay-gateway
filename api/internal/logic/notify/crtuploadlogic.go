package notify

import (
	"bufio"
	"context"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CrtUploadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCrtUploadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CrtUploadLogic {
	return &CrtUploadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CrtUploadLogic) CrtUpload(req *types.CrtUploadReq, r *http.Request) (resp *types.ResultResp, err error) {

	acpk, acpkInfo, err := r.FormFile("AppCertPublicKey")
	pk, pkInfo, err := r.FormFile("PublicKey")
	prCert, prCertInfo, err := r.FormFile("PayRootCert")

	if acpkInfo != nil && req.AppCertPublicKeyPath != "" {
		err = l.writeCrtFile(acpk, req.AppCertPublicKeyPath)
		if err != nil {
			return
		}
	}

	if pkInfo != nil && req.PublicKeyPath != "" {
		err = l.writeCrtFile(pk, req.PublicKeyPath)
		if err != nil {
			return
		}
	}

	if prCertInfo != nil && req.PayRootCertPath != "" {
		err = l.writeCrtFile(prCert, req.PayRootCertPath)
		if err != nil {
			return
		}
	}

	return
}

func (l *CrtUploadLogic) writeCrtFile(file multipart.File, filePath string) (err error) {
	os.Remove(filePath)

	path, fileName := filepath.Split(filePath)
	println(fileName)
	//创建文件夹
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return
	}

	nf, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	defer nf.Close()
	if err != nil {
		return
	}
	writer := bufio.NewWriter(nf)

	info, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}
	_, err = writer.Write(info)
	if err != nil {
		return
	}
	err = writer.Flush()

	return
}
