package douyin

import (
	"crypto/rsa"
	"github.com/bytedance/sonic"
	"reflect"
	"testing"
	"time"
)

func TestPayClient_RequestOrder(t *testing.T) {
	type fields struct {
		config *PayConfig
	}
	type args struct {
		data *RequestOrderData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			name: "",
			fields: fields{
				config: &PayConfig{
					AppId:      "testAppId",
					PrivateKey: "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA2YQ6yshguPnxmqPzUt16jubAz3ds67cQa/zAfAK1nd5gAwop\nH4Jxtpz0dyZvT/cNd5KUmV/lfVRT93ltMWxQ5kzjUse5D+g+9rSh8rBuZf0eUUQO\nESepp28/r2nOYVa0mOy30QK7qMfrZ9+TbAp0sS4GPvygpiYKk914jKZ5m8oLvcp7\nYg20fuV3nhTG1C/GeCmI9294yVTGiFCSG3/3WzQBitQnXbye4rRm/sIFN0GN22dm\n8a84kQLVh6ZcrMLKZXXFzpz+bN+HOjt4t2RS478GD+TDHq/YsqKnUBaPOuM5RxZW\nqz/3qN2Jrpl9RHxKNRpMGziz8xjkrK8FXPoI5QIDAQABAoIBADgXRSHtsiOBMLBz\n/tcrjeMz1hyp60iNmIqATxKrkDH5mkCuahRaCwDQUKo5GxM/3hUrk25JsGA1UsHK\nakIIcIQy55v9LNfRSAtOYUS4AoACWcMTDZ2W4MTwhzewzSuEtGWLBYu8bLAFfcr3\neIiv2Y+nEq1DcBnoTWn7/o4mj82APufdlYz+rehVU619kFaNO9YsECKeYeN/7dqs\n4Y92yUe1Ka9wCIYitIsjSBgnJ9QhYTB3K5wqJkiTIleLjimJQqoxfkwBKmgeDC/T\nglACo3S9Ou1pTZV49NZASjegTvtov4n4XCOLO93yrfyG0ZKBUozgI6VlXEAfjG1N\nZuAfAzECgYEA9FfrXqXGjFUaTjvR34NONfxwLTpeAjUOAKq0o+Dxal5YZ1ICNj6q\nREYd/1+K3dXcSQO8gprNGJt8fk9fs65r+DjS358pM4aGLlLssxZkrXCuXBZr4i3D\nitNnUlTCP0qhj14GlTxYnBQeOM6JOEOWPXdWWO7AhCcjtY2JEbrnXdsCgYEA4+Su\nvRUpmtFXXud71QP3Zt2kTE7AztU0B5F2580Ac67mXd0O5dGGpkNHKZWnP7BEh4am\npj4yISEwnq1ugO0ExtobpqmdBN3b+nkoFQLmgkFPFBdXJPGNFGHpORxfyQFJb3V0\nPsyHBp0SBkyCsgxthFDm50ZELJAYnzQV1OPY0D8CgYEAxMniaJH+/Jq12vhWqTsp\nTFWJSwPNHt337xWM8seB53cgn+Xunh2OJ/qIwloCj3NkPPHjaxSdxgnEFD59B0uc\n7YdmXm/jUPoxKzHiLMIGR6GO69+q97h/2lk0x5w37Z1/zOWfS6YUf2+8f2foIAZf\nBBYO1wVCy6xyGBBrqnnrSS0CgYAnq1b/cv+bA3XB/2l+2wHl1g8TeWH2nwY/iwK3\ntuetO3S+Qgyl1KMrreplQreqTnSfYsD/jzQKsExWUro5lwiN1MmbaUr73eK85voj\nLi4R3mx1gtqYg7ObKLAAUQAbbS3rSPbDN7cJX64Tip31gFRQBAUtnP2hBDRFAjwK\not7K4QKBgFhMpVXzYaRlavj1xy/RPuQeox8ppeIQUeddI8VZlF2eZs5fr9QdxBs3\ntliU3OgCkfuNqIfxpxFeCnhIKzKnHG5/qxifV5VMyQFV5nBwkA2ZIh1p0zx9n2Db\nxncdJPuwppUAVP2D16wMdSIw7sdvohZ6dUzUBw2XV/YpeuXHP3Pl\n-----END RSA PRIVATE KEY-----",
					KeyVersion: "1",
					NotifyUrl:  "http://test.api.pay-gateway.yunxiacn.com/notify/douyin",
				},
			},
			args: args{
				data: &RequestOrderData{
					SkuList: []*Sku{
						{
							SkuId:       "testSkuId",
							Price:       1,
							Quantity:    1,
							Title:       "testSkuId",
							ImageList:   nil,
							Type:        0,
							TagGroupId:  "",
							EntrySchema: nil,
						},
					},
					OutOrderNo:       "",
					TotalAmount:      0,
					PayExpireSeconds: 0,
					PayNotifyUrl:     "",
					MerchantUid:      "",
					OrderEntrySchema: nil,
					LimitPayWayList:  nil,
				},
			},
			want:    "",
			want1:   "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PayClient{
				config: tt.fields.config,
			}
			got, got1, err := c.RequestOrder(tt.args.data)
			t.Log(got, got1, err)
			if (err != nil) != tt.wantErr {
				t.Errorf("RequestOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("RequestOrder() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("RequestOrder() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestParsePKCS1PrivateKey(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		wantKey *rsa.PrivateKey
		wantErr bool
	}{
		{
			name: "",
			args: args{
				data: []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA2YQ6yshguPnxmqPzUt16jubAz3ds67cQa/zAfAK1nd5gAwop
H4Jxtpz0dyZvT/cNd5KUmV/lfVRT93ltMWxQ5kzjUse5D+g+9rSh8rBuZf0eUUQO
ESepp28/r2nOYVa0mOy30QK7qMfrZ9+TbAp0sS4GPvygpiYKk914jKZ5m8oLvcp7
Yg20fuV3nhTG1C/GeCmI9294yVTGiFCSG3/3WzQBitQnXbye4rRm/sIFN0GN22dm
8a84kQLVh6ZcrMLKZXXFzpz+bN+HOjt4t2RS478GD+TDHq/YsqKnUBaPOuM5RxZW
qz/3qN2Jrpl9RHxKNRpMGziz8xjkrK8FXPoI5QIDAQABAoIBADgXRSHtsiOBMLBz
/tcrjeMz1hyp60iNmIqATxKrkDH5mkCuahRaCwDQUKo5GxM/3hUrk25JsGA1UsHK
akIIcIQy55v9LNfRSAtOYUS4AoACWcMTDZ2W4MTwhzewzSuEtGWLBYu8bLAFfcr3
eIiv2Y+nEq1DcBnoTWn7/o4mj82APufdlYz+rehVU619kFaNO9YsECKeYeN/7dqs
4Y92yUe1Ka9wCIYitIsjSBgnJ9QhYTB3K5wqJkiTIleLjimJQqoxfkwBKmgeDC/T
glACo3S9Ou1pTZV49NZASjegTvtov4n4XCOLO93yrfyG0ZKBUozgI6VlXEAfjG1N
ZuAfAzECgYEA9FfrXqXGjFUaTjvR34NONfxwLTpeAjUOAKq0o+Dxal5YZ1ICNj6q
REYd/1+K3dXcSQO8gprNGJt8fk9fs65r+DjS358pM4aGLlLssxZkrXCuXBZr4i3D
itNnUlTCP0qhj14GlTxYnBQeOM6JOEOWPXdWWO7AhCcjtY2JEbrnXdsCgYEA4+Su
vRUpmtFXXud71QP3Zt2kTE7AztU0B5F2580Ac67mXd0O5dGGpkNHKZWnP7BEh4am
pj4yISEwnq1ugO0ExtobpqmdBN3b+nkoFQLmgkFPFBdXJPGNFGHpORxfyQFJb3V0
PsyHBp0SBkyCsgxthFDm50ZELJAYnzQV1OPY0D8CgYEAxMniaJH+/Jq12vhWqTsp
TFWJSwPNHt337xWM8seB53cgn+Xunh2OJ/qIwloCj3NkPPHjaxSdxgnEFD59B0uc
7YdmXm/jUPoxKzHiLMIGR6GO69+q97h/2lk0x5w37Z1/zOWfS6YUf2+8f2foIAZf
BBYO1wVCy6xyGBBrqnnrSS0CgYAnq1b/cv+bA3XB/2l+2wHl1g8TeWH2nwY/iwK3
tuetO3S+Qgyl1KMrreplQreqTnSfYsD/jzQKsExWUro5lwiN1MmbaUr73eK85voj
Li4R3mx1gtqYg7ObKLAAUQAbbS3rSPbDN7cJX64Tip31gFRQBAUtnP2hBDRFAjwK
ot7K4QKBgFhMpVXzYaRlavj1xy/RPuQeox8ppeIQUeddI8VZlF2eZs5fr9QdxBs3
tliU3OgCkfuNqIfxpxFeCnhIKzKnHG5/qxifV5VMyQFV5nBwkA2ZIh1p0zx9n2Db
xncdJPuwppUAVP2D16wMdSIw7sdvohZ6dUzUBw2XV/YpeuXHP3Pl
-----END RSA PRIVATE KEY-----
`),
			},
			wantKey: nil,
			wantErr: false,
		},
		{
			name: "",
			args: args{
				data: []byte(`-----BEGIN RSA PRIVATE KEY----- MIIEowIBAAKCAQEA1HpJ+Uz8/E4cOxwSacgIIx+jwKpOsZJTSvH//ehue+m/HMSB hBMijf1WPlfoHxpWdIA9bBzbN3awyDVvpiIf0UtZbckuNlEJDs14uaaq/Heacp6r toYp2WiVCVJCiJS4F6Ktp3JlY3MLkGlF8t/Sg1qVjwvpwzs6icOloNbFLcQ+ioOL KZnuZ3j8wAokEslNgtMtdKMiHocYm4oNKEPBCtbqhGiw+li9Wh7TtUHo6L5g63Pb vC8OIegbErUBMSbobFcadefTK5vVntgRSgrSZ4rJrT4l0oyKcRY+Z/5Mx8ov+O3z b//08Be3QGwfH0jX+laKO/qVDIkxTf4aqqyUywIDAQABAoIBAEnxUMMAZt4K9Moh R8smQKawgRUwb3heWrwvIY4kECbxPn/tZsEmw5S0QAosH2yLhuC+LCHunN9dX8Ic zoD7SSVV2oZZR8rBQqyzFrtM5B4+JTKUQ1+eqvus6Ii45syPLM2U4GfwaJZGWBTm feA6whDSOk/wrmYxu3pr6rzhYPc+vMzGH5hU2ONrkJ7nSALZIKmKrXJr+ZWUlRR9 3kO7569kkmKIRLps50oaxjoabn5YrhcWN3I02LuWyGdWfKC5bb03OTq2DfIUcwMq GjadEeW9tw3hWyF6wNJH/TQbElJ3SgIbrr8p7gVAiIhuMpNyPdeBDjbwLMVkssk4 Vw5eMNECgYEA6d4HawJgNGON4acFG2Ly8X0AuqJelEZaMJLPeKC1IoBMRgC0aAAx DLE3KL+Mg9AfBLlBJEMKnMIXKRlv1XwMX5Y1j+w8j3N5mjsS7PSuZ2Z4/EXp7JsS v3goGqGDJt3MOcqDE+6x4J+hansgZlq8IhdLp5ZfE3/eBmIXfuVncNkCgYEA6JYM NkK4ijqlAfMGgVyBEayu+pUCob1NQnKWviHnrpbLxTjLhGGd1RsmAXJoIQ4UpC6l ztlvY+77AQNrIf7qiGcWE2lcgVRo49StYun165K5gCNKvRL/PTGbATawxkkRp9FY qxlbfZ8m19r8Gvy7Hv9DKzP0KVFL23xZXa3i7EMCgYEAqJ/0zU2bPGsD5E5fOk3w SfsNyYRFmbfYU+mnOpz1vfiwBlF/wvVQaIxm7zSeBnTLyMYimBjW0AyKUpIKtu2I pmtSF4IIcI6cgX5SuRP7pIaYeZ2Xe4iczf2/POR0AlQuawT/2iFjlEjFRFefFu4Z dKNDY4Ti7NZnqkaUFmUmXpECgYAb8NKcvh0vNeZWupxAdB1pQyZuIfKD/ZrHbb6g LrCHb8Qg+Dayu4tba3yAVf8eYXOnWZc/py1TgwUSVRfMqLQCGOg1AsZDHyHOpOED bfnGCAVS2GqFFkAlWM03Mxu/Zk3BrEuLmms8Rx9CdSMuFshf6+hky0P8prCHOIP/ 1gDZwwKBgBNe98Tse4s7QyrVb3yGpegfHD8LlxWQUjNV0jJNHU3BcYeqYu35AWfS Q4VL2P5/IVv60uSv7Z/mLoMDSXUvOy4ciKlbCB6shm9fJ59FTR3OLBqyHTwZgJvC YMVNPCaq0KtZBynE0BpwNWP+1LEvUzjCMxlb3Yq5X9tXmjEY40S3 -----END RSA PRIVATE KEY-----`),
			},
			wantKey: nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKey, err := ParsePKCS1PrivateKey(tt.args.data)
			t.Log(gotKey, err)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePKCS1PrivateKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotKey, tt.wantKey) {
				t.Errorf("ParsePKCS1PrivateKey() gotKey = %v, want %v", gotKey, tt.wantKey)
			}
		})
	}
}

func TestPayClient_CheckSign(t *testing.T) {
	type fields struct {
		config *PayConfig
	}
	type args struct {
		timestamp string
		nonce     string
		body      string
		signature string
		pubKeyStr string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "",
			fields: fields{
				config: &PayConfig{
					AppId:             "",
					PrivateKey:        "",
					KeyVersion:        "",
					NotifyUrl:         "",
					PlatformPublicKey: "-----BEGIN RSA PUBLIC KEY-----\nMIIBCgKCAQEA2YQ6yshguPnxmqPzUt16jubAz3ds67cQa/zAfAK1nd5gAwopH4Jx\ntpz0dyZvT/cNd5KUmV/lfVRT93ltMWxQ5kzjUse5D+g+9rSh8rBuZf0eUUQOESep\np28/r2nOYVa0mOy30QK7qMfrZ9+TbAp0sS4GPvygpiYKk914jKZ5m8oLvcp7Yg20\nfuV3nhTG1C/GeCmI9294yVTGiFCSG3/3WzQBitQnXbye4rRm/sIFN0GN22dm8a84\nkQLVh6ZcrMLKZXXFzpz+bN+HOjt4t2RS478GD+TDHq/YsqKnUBaPOuM5RxZWqz/3\nqN2Jrpl9RHxKNRpMGziz8xjkrK8FXPoI5QIDAQAB\n-----END RSA PUBLIC KEY-----",
				},
			},
			args:    args{},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PayClient{
				config: tt.fields.config,
			}
			got, err := c.CheckSign(tt.args.timestamp, tt.args.nonce, tt.args.body, tt.args.signature, tt.args.pubKeyStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckSign() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CheckSign() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPayClient_QueryOrder(t *testing.T) {
	type fields struct {
		config *PayConfig
	}
	type args struct {
		orderId    string
		outOrderId string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *QueryOrderResp
		wantErr bool
	}{
		{
			name: "",
			fields: fields{
				config: &PayConfig{
					AppId:             "tt1683603e89bd1ac801",
					PrivateKey:        "-----BEGIN RSA PRIVATE KEY-----\nMIIEpQIBAAKCAQEAxLzQfjf6lXYWWhd/vxxVL6AwEb6ZxhoWUw3xYDTqI9ZkcYxL\nTeX7ABlhPT0uIIqS1Lw56fWzDra73RMaDk7XhhPq8jViZxlrn7eQaJa5v2gf1IaZ\nIenPBmgbQ1fjDL6dW5grmE/9uHpP1+Yc2a3zTS/z0wkMm6F7z5FNpwotzCqAp4zL\nQcW2dlJj/oxEGWnvFmqCXukZ94fAoXMinK+qZ1jCZCNavw0g8aLLPCNje7cV02kB\nIqJjhLCbsjP3CMpehHUwUUJV4Jtq5mTatsDjGmPv68Hgo/SvmM1bOn2jdyYz0x8W\n68uB9l8HwwVVEXdlXPb3GAfZzV08Jk75bq3g6QIDAQABAoIBAGxfaiYtJe8RBo0I\nJsmajN5YSkJsEP8MPcHwi0covtWQ8vGNi6nUhuhuEp+ORQuN6gYfzXMwcjsns+K6\n8/5vtc9Yx3I2sAcE/MEVeAn1BUsHy4jhwBbrWaw4ytPU5PCPS9U5xMH5RlVJoxPV\n4YyTgtPBF3nnoTdVxAL6EqFyPPoacr2/CPp18NG10A+6sP9wrLh0HNr+G4mpJkk+\nqRgYmpvJci2JR55IKHxpd8LMQ6ajc6PY6SvtfMsrPz9CF64aWyhI5H7tYw14rXUe\navUAfcDMGRbzwhkKX43aNpfMn2Fjyf92/5HJ1HHGpNsqy9rMuDQkOyV+C/EtcTpr\nywy4JwECgYEA+22iZhAvCgRnF3NFNY7n8Xl/DK0yrkEsPxaL+rnfJFFkflgIW/tC\nbe0iU+FsS8gSaszICtJZY70tWdhMhx0Qmc/xvPFKx2xw3a4sgPQj55NYLzOazmHD\nQo5FzUSHgfUEJgN3FaipT/WzNL17JP4MA3tHCWz8chjlqJFmwnXnxIkCgYEAyFCa\nHEkPojNewLG/yWTHGbxe2Ifji6VOJ1UaLooOUZgDhr9VK6CAGfFKw2JuJi7EbXqm\nvcppmXSduc14gn3GrfzHmx2F7Uy/gM3o1EWtTuk9MY3uvtTzFTUtgAFTv3NuhdtH\nX2ggKJL4LZPrcZSNtC0XlfOmb/F/m6Nk8FUg4WECgYEAqpowqJpwoJZuMU5I9td5\n8LLlD3/yNKUKVeCBqOY4UBdeXhBz054A7EAMm+gIqL8gKBG95wHmH7Q8sor/GmsR\nWZzsxazgdcLSLslBb1q5hifHnXehokpZyK5rFKZcYEUVxIlzY2HnSNdJ+w5bIbW0\nByS+BdpKzUyxgJjwpiCE3CkCgYEAwQvkqXPDzEaDf2MN+JHVyzidk0HKig8iNYev\ndsB3siy04UxNUYEZU2cV7RxUGRojFXsJbIjAojIfuyuIgwGh0pV07ElUg2/ecsx+\nIOyRbCYdYj9toZ1qMrsQAXfF9RDSp8++hfS8YT3aTVprogdPVR/LxiiM8v8jQqQC\nKBdyW6ECgYEArY+mZsdmykhRmCsUfut9mP1FulXh+Bs+TYmABsa3L0NNeoufXnyY\n6Nis8PT5Loa+k2P98/oBqozDvJpazNPqanBP+4ZH3teDCWejEHiYt+BmRnOLCcb8\nwsv4gC1V9tHlpSaT5dDNlN5F0M6ReTe90WHAnAhOoSbMr+pKmkhdJWI=\n-----END RSA PRIVATE KEY-----",
					KeyVersion:        "1",
					NotifyUrl:         "",
					PlatformPublicKey: "",
					AppSecret:         "",
					GetClientTokenUrl: "http://test.base-app-config-server-api.yunxiacn.com/api_server/base/douyin/token/get",
				},
			},
			args: args{
				orderId:    "",
				outOrderId: "1211719316563693568",
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "",
			fields: fields{
				config: &PayConfig{
					AppId:             "tt1683603e89bd1ac801",
					PrivateKey:        "-----BEGIN RSA PRIVATE KEY-----\nMIIEpQIBAAKCAQEAxLzQfjf6lXYWWhd/vxxVL6AwEb6ZxhoWUw3xYDTqI9ZkcYxL\nTeX7ABlhPT0uIIqS1Lw56fWzDra73RMaDk7XhhPq8jViZxlrn7eQaJa5v2gf1IaZ\nIenPBmgbQ1fjDL6dW5grmE/9uHpP1+Yc2a3zTS/z0wkMm6F7z5FNpwotzCqAp4zL\nQcW2dlJj/oxEGWnvFmqCXukZ94fAoXMinK+qZ1jCZCNavw0g8aLLPCNje7cV02kB\nIqJjhLCbsjP3CMpehHUwUUJV4Jtq5mTatsDjGmPv68Hgo/SvmM1bOn2jdyYz0x8W\n68uB9l8HwwVVEXdlXPb3GAfZzV08Jk75bq3g6QIDAQABAoIBAGxfaiYtJe8RBo0I\nJsmajN5YSkJsEP8MPcHwi0covtWQ8vGNi6nUhuhuEp+ORQuN6gYfzXMwcjsns+K6\n8/5vtc9Yx3I2sAcE/MEVeAn1BUsHy4jhwBbrWaw4ytPU5PCPS9U5xMH5RlVJoxPV\n4YyTgtPBF3nnoTdVxAL6EqFyPPoacr2/CPp18NG10A+6sP9wrLh0HNr+G4mpJkk+\nqRgYmpvJci2JR55IKHxpd8LMQ6ajc6PY6SvtfMsrPz9CF64aWyhI5H7tYw14rXUe\navUAfcDMGRbzwhkKX43aNpfMn2Fjyf92/5HJ1HHGpNsqy9rMuDQkOyV+C/EtcTpr\nywy4JwECgYEA+22iZhAvCgRnF3NFNY7n8Xl/DK0yrkEsPxaL+rnfJFFkflgIW/tC\nbe0iU+FsS8gSaszICtJZY70tWdhMhx0Qmc/xvPFKx2xw3a4sgPQj55NYLzOazmHD\nQo5FzUSHgfUEJgN3FaipT/WzNL17JP4MA3tHCWz8chjlqJFmwnXnxIkCgYEAyFCa\nHEkPojNewLG/yWTHGbxe2Ifji6VOJ1UaLooOUZgDhr9VK6CAGfFKw2JuJi7EbXqm\nvcppmXSduc14gn3GrfzHmx2F7Uy/gM3o1EWtTuk9MY3uvtTzFTUtgAFTv3NuhdtH\nX2ggKJL4LZPrcZSNtC0XlfOmb/F/m6Nk8FUg4WECgYEAqpowqJpwoJZuMU5I9td5\n8LLlD3/yNKUKVeCBqOY4UBdeXhBz054A7EAMm+gIqL8gKBG95wHmH7Q8sor/GmsR\nWZzsxazgdcLSLslBb1q5hifHnXehokpZyK5rFKZcYEUVxIlzY2HnSNdJ+w5bIbW0\nByS+BdpKzUyxgJjwpiCE3CkCgYEAwQvkqXPDzEaDf2MN+JHVyzidk0HKig8iNYev\ndsB3siy04UxNUYEZU2cV7RxUGRojFXsJbIjAojIfuyuIgwGh0pV07ElUg2/ecsx+\nIOyRbCYdYj9toZ1qMrsQAXfF9RDSp8++hfS8YT3aTVprogdPVR/LxiiM8v8jQqQC\nKBdyW6ECgYEArY+mZsdmykhRmCsUfut9mP1FulXh+Bs+TYmABsa3L0NNeoufXnyY\n6Nis8PT5Loa+k2P98/oBqozDvJpazNPqanBP+4ZH3teDCWejEHiYt+BmRnOLCcb8\nwsv4gC1V9tHlpSaT5dDNlN5F0M6ReTe90WHAnAhOoSbMr+pKmkhdJWI=\n-----END RSA PRIVATE KEY-----",
					KeyVersion:        "1",
					NotifyUrl:         "",
					PlatformPublicKey: "",
					AppSecret:         "",
					GetClientTokenUrl: "http://test.base-app-config-server-api.yunxiacn.com/api_server/base/douyin/token/get",
				},
			},
			args: args{
				orderId:    "",
				outOrderId: "1211717853972140032",
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func(start time.Time) {
				t.Logf("payClient.QueryOrder timecost:%v", time.Since(start))
			}(time.Now())

			c := &PayClient{
				config: tt.fields.config,
			}
			got, err := c.QueryOrder(tt.args.orderId, tt.args.outOrderId)
			jsonData, _ := sonic.MarshalString(got)
			t.Logf("err:%v, resp:%s,", err, jsonData)

			if (err != nil) != tt.wantErr {
				t.Errorf("QueryOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("QueryOrder() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPayClient_CreateRefundOrder(t *testing.T) {
	type fields struct {
		config *PayConfig
	}
	type args struct {
		req *CreateRefundOrderReq
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *CreateRefundResp
		wantErr bool
	}{
		{
			name:    "",
			fields:  fields{},
			args:    args{},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PayClient{
				config: tt.fields.config,
			}
			got, err := c.CreateRefundOrder(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateRefundOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateRefundOrder() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPayClient_GetClientToken(t *testing.T) {
	type fields struct {
		config *PayConfig
	}
	tests := []struct {
		name    string
		fields  fields
		want    *GetClientTokenResp
		wantErr bool
	}{
		{
			name: "",
			fields: fields{
				config: &PayConfig{
					AppId:             "tt1683603e89bd1ac801",
					PrivateKey:        "",
					KeyVersion:        "",
					NotifyUrl:         "",
					PlatformPublicKey: "",
					AppSecret:         "040c366e8014ed70a549c47f88ac3cacb05f7f6c",
				},
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PayClient{
				config: tt.fields.config,
			}
			got, err := c.GetClientToken()
			t.Log(got, err)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetClientToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetClientToken() got = %v, want %v", got, tt.want)
			}
		})
	}
}
