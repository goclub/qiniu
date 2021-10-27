package xqiniu_test

import (
	"encoding/json"
	xqiniu "github.com/goclub/qiniu"
	"github.com/qiniu/api.v7/v7/storage"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"
)

func ExampleBasic() {
	qiniuClient := xqiniu.Client{
		AK: TestAK,
		SK: TestSK,
		Domain: TestDomain,
		Bucket: TestBucket,
		StorageConfig: storage.Config{
			Zone:          &storage.ZoneHuanan,
		},
	}
	resp, err := qiniuClient.Upload(xqiniu.Upload{
		LocalFilename: "localfile.txt",
		QiniuFileKey: "name.txt",
	}) ; if err != nil {panic(err)}
	// 公开空间
	qiniuClient.PublicURL(resp.Key)

	// 分片上传大文件
	qiniuClient.ResumeUpload(xqiniu.ResumeUpload{
		LocalFilename: "localfile.text",
		QiniuFileKey: "name.txt",
		RputExtra:     storage.RputExtra{},
	})

	// 直接上传少了字节，大量文件建议 分批读取通过 file os.O_APPEND 插入本地文件后使用 ResumeUpload 上传
	qiniuClient.BytesUpdate(xqiniu.BytesUpdate{
		QiniuFileKey: "name.txt",
		Data: []byte("abc"),
		RputExtra:     storage.RputExtra{},
	})
}
func TestClient_Pfop(t *testing.T) {
	qiniuClient := xqiniu.Client{
		AK: TestAK,
		SK: TestSK,
		Domain: TestDomain,
		Bucket: TestBucket,
		StorageConfig: storage.Config{
			Zone:          &storage.ZoneHuanan,
		},
	}
	{
		source  := []xqiniu.ZipData{}
		{
			one, err := qiniuClient.BytesUpdate(xqiniu.BytesUpdate{
				QiniuFileKey: "zip-source-1.txt",
				Data:         []byte("1"),
			})
			assert.NoError(t, err)
			source = append(source, xqiniu.ZipData{
				QiniuFileKey: one.Key,
				ZipRename: "zip-1.txt",
			})
		}
		{
			two, err := qiniuClient.BytesUpdate(xqiniu.BytesUpdate{
				QiniuFileKey: "zip-source-2.txt",
				Data:         []byte("2"),
			})
			assert.NoError(t, err)
			source = append(source, xqiniu.ZipData{
				QiniuFileKey: two.Key,
				ZipRename: "zip-2.txt",
			})
		}
		persistentID, err := qiniuClient.Pfop(xqiniu.Pfop{
			Source:          source,
			QiniuZipFileKey: "zip/" + time.Now().Format("20060102150405") + "file.zip",
		})
		assert.NoError(t, err)
		log.Print("persistentID ", persistentID)
		time.Sleep(time.Second*5) // time.Sleep 测试用，生成环境请实现查询机制或使用 xqiniu.Prop 的 NotifyURL字段
		status, err := qiniuClient.Prefop(persistentID)
		assert.NoError(t, err)
		b, err := json.MarshalIndent(status,"", "  ")
		log.Print("pfop status :", string(b))
	}
}
func TestFile(t *testing.T) {
	
	qiniuClient := xqiniu.Client{
		AK: TestAK,
		SK: TestSK,
		Domain: TestDomain,
		Bucket: TestBucket,
		StorageConfig: storage.Config{
			Zone:          &storage.ZoneHuanan,
		},
	}
	{
		resp, err := qiniuClient.Upload(xqiniu.Upload{
			LocalFilename: "go.mod",
			QiniuFileKey: time.Now().Format("20060102150405") + "golangmod",
		})
		assert.NoError(t, err)
		log.Print(resp)
	}
	{
		_, err := qiniuClient.ResumeUpload(xqiniu.ResumeUpload{
			LocalFilename: "go.mod",
			QiniuFileKey: time.Now().Format("20060102150405") + "demo.txt",
		})
		assert.NoError(t, err)
	}
	{
		cloudFilename := time.Now().Format("20060102150405") + "byte.txt"
		resp, err := qiniuClient.BytesUpdate(xqiniu.BytesUpdate{
			QiniuFileKey: cloudFilename,
			Data:          []byte("abc"),
			RputExtra:     storage.RputExtra{},
		})
		assert.NoError(t, err)
		assert.Equal(t,resp.Key, cloudFilename)
		url := qiniuClient.PrivateURL(xqiniu.PrivateURL{
			Key:      resp.Key,
			Duration: time.Second*10,
			Attname:  "中文.txt",
		})
		httpResp , err := http.Get(url) ; assert.NoError(t, err)
		data, err := ioutil.ReadAll(httpResp.Body) ;assert.NoError(t, err)
		log.Print(url)
		assert.Equal(t,data, []byte("abc"))
		err = qiniuClient.BucketManager().Delete(TestBucket, resp.Key) ; if err != nil {panic(err)}
	}
	{
		cloudFilename := time.Now().Format("20060102150405") + "byte.txt"
		resp, err := qiniuClient.BytesUpdate(xqiniu.BytesUpdate{
			QiniuFileKey: cloudFilename,
			Data:          []byte("abc"),
			RputExtra:     storage.RputExtra{},
		})
		assert.NoError(t, err)
		assert.Equal(t,resp.Key, cloudFilename)
		url := qiniuClient.PrivateURL(xqiniu.PrivateURL{
			Key:      resp.Key,
			Duration: time.Second*10,
			Attname:  time.Now().Format("20060102150405") + "othername.csv",
		})
		log.Print(url)
	}
}
func TestPing(t *testing.T) {
	
	{
		qiniuClient := xqiniu.Client{
			AK: TestAK,
			SK: TestSK,
			Domain: TestDomain,
			Bucket: TestBucket,
			StorageConfig: storage.Config{
				Zone:          &storage.ZoneHuanan,
			},
		}
		assert.NoError(t,qiniuClient.Ping())
	}

	{
		qiniuClient := xqiniu.Client{
			AK: "",
			SK: TestSK,
			Domain: TestDomain,
			Bucket: TestBucket,
			StorageConfig: storage.Config{
				Zone:          &storage.ZoneHuanan,
			},
		}
		assert.EqualError(t,qiniuClient.Ping(),"AK can not be empty string")
	}
	{
		qiniuClient := xqiniu.Client{
			AK: TestAK,
			SK: "",
			Domain: TestDomain,
			Bucket: TestBucket,
			StorageConfig: storage.Config{
				Zone:          &storage.ZoneHuanan,
			},
		}
		assert.EqualError(t,qiniuClient.Ping(),"SK can not be empty string")
	}
	{
		qiniuClient := xqiniu.Client{
			AK: TestAK,
			SK: TestSK,
			Domain: TestDomain,
			Bucket: "",
			StorageConfig: storage.Config{
				Zone:          &storage.ZoneHuanan,
			},
		}
		assert.EqualError(t,qiniuClient.Ping(),"Bucket can not be empty string")
	}

}

func TestClient_ImageCensor(t *testing.T) {
	qiniuClient := xqiniu.Client{
		AK: TestAK,
		SK: TestSK,
		Domain: TestDomain,
		Bucket: TestBucket,
		StorageConfig: storage.Config{
			Zone:          &storage.ZoneHuanan,
		},
	}
	reply, err := qiniuClient.ImageCensor(xqiniu.ImageCensor{
		URL:       "https://i.picsum.photos/id/237/536/354.jpg?hmac=i0yVXW1ORpyCZpQ-CknuyV-jbtU7_x9EBQVhvT5aRr0",
		Scenes:    []string{"pulp", "terror", "politician"},
		PutPolicy: storage.PutPolicy{},
	}) ; assert.NoError(t, err)
	b, err := json.MarshalIndent(reply, "", "  ") ; if err != nil {
	    return
	}
	log.Print(string(b))
}
