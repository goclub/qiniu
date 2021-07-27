## goclub/qiniu

> 封装七牛 Go SDK，以更友好的接口上传文件 

## 上传

```go
package main 
import (
    xqiniu "github.com/goclub/qiniu"
	"github.com/qiniu/api.v7/v7/storage"
	"time"
)
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
    QiniuFilename: "name.txt",
    PutExtra:      storage.PutExtra{},
}) ; if err != nil {panic(err)}

// 分片上传大文件
qiniuClient.ResumeUpload(xqiniu.ResumeUpload{
    LocalFilename: "localfile.text",
    QiniuFilename: "name.txt",
    RputExtra:     storage.RputExtra{},
})

// 直接上传少了字节，大量文件建议 分批读取通过 file os.O_APPEND 插入本地文件后使用 ResumeUpload 上传
qiniuClient.BytesUpdate(xqiniu.BytesUpdate{
    QiniuFilename: "name.txt",
    Data: []byte("abc"),
    RputExtra:     storage.RputExtra{},
})

// 私有空间的文件需要公开访问需要使用

qiniuClient.PrivateURL()
```

## 压缩文件

```go
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
```