package xqiniu

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	xerr "github.com/goclub/error"
	"github.com/qiniu/api.v7/v7/auth"
	"github.com/qiniu/api.v7/v7/auth/qbox"
	"github.com/qiniu/api.v7/v7/storage"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func createCallReader(reader func() (end bool, data []byte), file *os.File) (err error) {
	end, data := reader()
	_, err = file.Write(data)
	if err != nil {
		return
	}
	if !end {
		return createCallReader(reader, file)
	}
	return
}
func Create(filename string, reader func() (end bool, data []byte)) (err error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return
	}
	return createCallReader(reader, file)

}

type Client struct {
	AK            string
	SK            string
	Domain        string
	Bucket        string
	StorageConfig storage.Config
	httpClient    *http.Client
}

func (q Client) getHttpClient() *http.Client {
	if q.httpClient == nil {
		q.httpClient = &http.Client{}
	}
	return q.httpClient
}

type OptionPutPolicy struct {
	Valid     bool
	PutPolicy storage.PutPolicy
}

func (o OptionPutPolicy) Unwrap() storage.PutPolicy {
	if !o.Valid {
		panic(errors.New("OptionPutPolicy.Valid is false"))
	}
	return o.PutPolicy
}
func PutPolicy(putPolicy storage.PutPolicy) OptionPutPolicy {
	return OptionPutPolicy{
		Valid:     true,
		PutPolicy: putPolicy,
	}
}
func (q Client) DefaultPutPolicy() storage.PutPolicy {
	return storage.PutPolicy{
		Scope: q.Bucket,
	}
}

func (q Client) UploadToken(policy storage.PutPolicy) string {
	return policy.UploadToken(q.Mac())
}
func (q Client) Mac() *qbox.Mac {
	return qbox.NewMac(q.AK, q.SK)
}

// get token q.Credentials().AddToken(auth.TokenQiniu, request)
func (q Client) Credentials() *auth.Credentials {
	return auth.New(q.AK, q.SK)
}

type ResumeUpload struct {
	LocalFilename string
	QiniuFileKey  string
	RputExtra     storage.RputExtra
	PutPolicy     storage.PutPolicy
}

func (q Client) ResumeUpload(data ResumeUpload) (reply Reply, err error) {
	if data.PutPolicy.Scope == "" {
		data.PutPolicy.Scope = q.Bucket
	}
	uploader := storage.NewResumeUploader(&q.StorageConfig)
	err = uploader.PutFile(context.Background(), &reply, q.UploadToken(data.PutPolicy), data.QiniuFileKey, data.LocalFilename, &data.RputExtra)
	if err != nil {
		return
	}
	return
}

type BytesUpdate struct {
	QiniuFileKey string
	Data         []byte
	RputExtra    storage.RputExtra
	PutPolicy    storage.PutPolicy
}

func (q Client) BytesUpdate(data BytesUpdate) (reply Reply, err error) {
	if data.PutPolicy.Scope == "" {
		data.PutPolicy.Scope = q.Bucket
	}
	uploader := storage.NewResumeUploader(&q.StorageConfig)
	err = uploader.Put(context.Background(), &reply, q.UploadToken(data.PutPolicy), data.QiniuFileKey, bytes.NewReader(data.Data), int64(len(data.Data)), &data.RputExtra)
	if err != nil {
		return
	}
	return
}

type Upload struct {
	LocalFilename string
	QiniuFileKey  string
	PutExtra      storage.PutExtra
	PutPolicy     storage.PutPolicy
}

func (q Client) Upload(data Upload) (reply Reply, err error) {
	if data.PutPolicy.Scope == "" {
		data.PutPolicy.Scope = q.Bucket
	}
	uploader := storage.NewFormUploader(&q.StorageConfig)
	err = uploader.PutFile(context.Background(), &reply, q.UploadToken(data.PutPolicy), data.QiniuFileKey, data.LocalFilename, &data.PutExtra)
	if err != nil {
		return
	}
	return
}

type Reply struct {
	Hash         string       `json:"hash"`
	PersistentID PersistentID `json:"persistentId"`
	Key          string       `json:"key"`
}

func (q Client) PublicURL(key string) string {
	return storage.MakePublicURL(q.Domain, key)
}

type PrivateURL struct {
	Key string
	// 访问有效期,超过时间则无法访问
	Duration time.Duration
	// 如果文件是会被下载的,可以通过 Attname 控制下载的文件名
	Attname string
}

func (q Client) PrivateURL(data PrivateURL) string {
	publicURL := q.PublicURL(data.Key)
	urlToSign := publicURL
	e := time.Now().Add(data.Duration).Unix()
	if strings.Contains(publicURL, "?") {
		urlToSign = fmt.Sprintf("%s&e=%d", urlToSign, e)
	} else {
		urlToSign = fmt.Sprintf("%s?e=%d", urlToSign, e)
	}
	if len(data.Attname) != 0 {
		v := url.Values{}
		v.Add("attname", data.Attname)
		urlToSign += "&" + v.Encode()
	}
	token := q.Credentials().Sign([]byte(urlToSign))
	privateURL := fmt.Sprintf("%s&token=%s", urlToSign, token)
	return privateURL
}
func (q Client) BucketManager() *storage.BucketManager {
	return storage.NewBucketManager(q.Credentials(), &q.StorageConfig)
}

func (q Client) Ping() error {
	if q.AK == "" {
		return errors.New("AK can not be empty string")
	}
	if q.SK == "" {
		return errors.New("SK can not be empty string")
	}
	if q.Domain == "" {
		return errors.New("Domain can not be empty string")
	}
	if q.Bucket == "" {
		return errors.New("Bucket can not be empty string")
	}
	err := q.BucketManager().DeleteAfterDays(q.Bucket, "Nonexistentfile__0102012", 0)
	if xerr.Unwrap(err).Error() == "no such file or directory" {
		return nil
	}
	return err
}

type ImageCensor struct {
	URL       string
	Scenes    []string
	PutPolicy storage.PutPolicy
}
type ImageCensorReply struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Result  struct {
		Scenes struct {
			Terror struct {
				Details []struct {
					Score      float64 `json:"score"`
					Suggestion string  `json:"suggestion"`
					Label      string  `json:"label"`
				} `json:"details"`
				Suggestion string `json:"suggestion"`
			} `json:"terror"`
			Politician struct {
				Suggestion string `json:"suggestion"`
			} `json:"politician"`
			Pulp struct {
				Details []struct {
					Score      float64 `json:"score"`
					Suggestion string  `json:"suggestion"`
					Label      string  `json:"label"`
				} `json:"details"`
				Suggestion string `json:"suggestion"`
			} `json:"pulp"`
		} `json:"scenes"`
		Suggestion string `json:"suggestion"`
	} `json:"result"`
}

func (q Client) ImageCensor(data ImageCensor) (reply ImageCensorReply, err error) {
	httpClient := q.getHttpClient()
	jsonb, err := json.Marshal(map[string]interface{}{
		"data": map[string]interface{}{
			"uri": data.URL,
		},
		"params": map[string]interface{}{
			"scenes": data.Scenes,
		},
	})
	if err != nil {
		return
	}
	body := bytes.NewReader(jsonb)
	request, err := http.NewRequest("POST", "https://ai.qiniuapi.com/v3/image/censor", body)
	if err != nil {
		return
	}
	request.Header.Add("Content-Type", "application/json")
	err = q.Credentials().AddToken(auth.TokenQiniu, request)
	if err != nil {
		return
	}
	resp, err := httpClient.Do(request)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		var data []byte
		data, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		err = xerr.New("https://ai.qiniuapi.com/v3/image/censor response error,\nstatus code " + strconv.FormatInt(int64(resp.StatusCode), 10) + "\nbody:" + string(data))
		return
	}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&reply)
	if err != nil {
		return
	}
	return
}
