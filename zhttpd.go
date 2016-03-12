package gimg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"image/png"
	"io/ioutil"
	_ "mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	//"time"

	"io"

	"github.com/issue9/identicon"
)

type ZHttpd struct {
	context      *ZContext
	storage      ZStorage
	contentTypes map[string]string
}

func NewHttpd(c *ZContext) *ZHttpd {
	return &ZHttpd{context: c,
		storage:      genStorageHandler(c),
		contentTypes: genContentTypes()}
}

func genStorageHandler(c *ZContext) ZStorage {
	var s ZStorage = nil
	var i int = c.Config.Storage.Mode

	switch i {
	case 1:
		s = NewFileStorage(c)
	case 2:
		break
	case 3:
		s = NewSSDBStorage(c)
		break
	}
	return s
}

func genContentTypes() map[string]string {
	types := make(map[string]string)
	types["jpg"] = "image/jpeg"
	types["jpeg"] = "image/jpeg"
	types["png"] = "image/png"
	types["gif"] = "image/gif"
	types["webp"] = "image/webp"

	return types
}

func (z *ZHttpd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method

	if "GET" == method {
		if path == "/" {
			z.doDefault(w, r)
		} else if path == "/info" {
			z.doInfo(w, r)
		} else {
			z.context.Logger.Info("path:" + path)
			md5Sum := path[1:len(path)]
			z.context.Logger.Info("md5Sum:" + md5Sum)

			if is_md5(md5Sum) {
				z.doGet(w, r, md5Sum)
			} else {
				http.NotFound(w, r)
			}
		}

	} else if "POST" == method {
		if path == "/upload" {
			z.doUpload(w, r)
		} else if path == "/identicon" {
			z.generateAvatar(w, r)
		} else {
			http.NotFound(w, r)
		}

	} else {
		http.NotFound(w, r)
	}

	return
}

func (z *ZHttpd) doDefault(w http.ResponseWriter, r *http.Request) {
	z.context.Logger.Info("call doDefault function........")

	html := `<!DOCTYPE html>
<html>
    <head>
        <meta charset="UTF-8"/>
    </head>
    <body>
        <form action="/upload" method="POST" enctype="multipart/form-data">
            <label for="field1">file:</label>
            <input name="upload_file" type="file" />
            <input type="submit"></input>
        </form>
    </body>
</html>`
	fmt.Fprint(w, html)

}

func (z *ZHttpd) doInfo(w http.ResponseWriter, r *http.Request) {
	z.context.Logger.Info("call doInfo function........")
	if err := r.ParseForm(); err != nil {
		z.context.Logger.Error(err.Error())
		z.doError(w, err, http.StatusForbidden)
		return
	}

	md5Sum := r.Form.Get("md5")
	z.context.Logger.Info("search md5  : %s", md5Sum)

	imgInfo, err := z.storage.InfoImage(md5Sum)
	if err != nil {
		z.context.Logger.Error(err.Error())
		z.doError(w, err, http.StatusForbidden)
		return
	}

	json, _ := json.Marshal(imgInfo)
	fmt.Fprint(w, string(json))

}

func (z *ZHttpd) doUpload(w http.ResponseWriter, r *http.Request) {
	z.context.Logger.Info("call doUpload function........")

	if err := r.ParseMultipartForm(CACHE_MAX_SIZE); err != nil {
		z.context.Logger.Error(err.Error())
		z.doError(w, err, http.StatusForbidden)
		return
	}

	file, _, err := r.FormFile("upload_file")
	if err != nil {
		z.doError(w, err, 500)
		return
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		z.doError(w, err, 500)
		return
	}

	md5Sum, err := z.storage.SaveImage(data)
	if err != nil {
		z.doError(w, err, 500)
		return
	}
	res, _ := json.Marshal(map[string]string{"Message": "upload success!", "md5": md5Sum})
	fmt.Fprint(w, string(res))

}

func (z *ZHttpd) generateAvatar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	fmt.Println(r.Form.Get("key"))
	img, err := identicon.Make(128, color.NRGBA{102, 204, 204, 50}, color.NRGBA{0, 0, 204, 100}, []byte(r.Form.Get("key")))
	if err != nil {
		z.doError(w, err, 500)
		return
	}
	fi, err := os.Create("u1.png")
	if err != nil {
		z.doError(w, err, 500)
		return
	}
	err = png.Encode(fi, img)
	if err != nil {
		z.doError(w, err, 500)
		return
	}
	defer fi.Close()
	fi.Seek(0, 0)
	buf, err := ioutil.ReadAll(fi)
	if err != nil {
		z.doError(w, err, 500)
		return
	}
	if len(buf) == 0 {
		z.doError(w, fmt.Errorf("buf is nil"), 500)
		return
	}
	md5Sum, err := z.storage.SaveImage(buf)
	if err != nil {
		z.doError(w, err, 500)
		return
	}
	res, _ := json.Marshal(map[string]string{"Message": "upload success!", "md5": md5Sum})
	fmt.Fprint(w, string(res))
}
func (z *ZHttpd) doGet(writer http.ResponseWriter, req *http.Request, md5Sum string) {
	z.context.Logger.Info("call doGet function........")
	if err := req.ParseForm(); err != nil {
		z.context.Logger.Error(err.Error())
		z.doError(writer, err, http.StatusForbidden)
		return
	}

	imgInfo, err := z.storage.InfoImage(md5Sum)
	if err != nil {
		z.context.Logger.Error(err.Error())
		z.doError(writer, err, http.StatusForbidden)
		return
	}

	var w, h, p, g, x, y, r, s, q int = 0, 0, 0, 0, 0, 0, 0, 0, 0
	var i, f string

	width := req.Form.Get("w")
	height := req.Form.Get("h")
	gary := req.Form.Get("g")
	xx := req.Form.Get("x")
	yy := req.Form.Get("y")
	rotate := req.Form.Get("r")

	w = str2Int(width)
	if w >= imgInfo.Width || w <= 0 {
		w = imgInfo.Width
	}

	h = str2Int(height)
	if h >= imgInfo.Height || h <= 0 {
		h = imgInfo.Height
	}

	g = str2Int(gary)
	if g != 1 {
		g = 0
	}

	x = str2Int(xx)
	if x < 0 {
		x = -1
	}

	// else if x > imgInfo.Width {
	// 	x = imgInfo.Width
	// }

	y = str2Int(yy)
	if y < 0 {
		y = -1
	}
	// else if y > imgInfo.Height {
	// 	y = imgInfo.Height
	// }

	r = str2Int(rotate)

	quality := req.Form.Get("q")
	q = str2Int(quality)
	if q <= 0 {
		//q = imgInfo.Quality
		q = z.context.Config.System.Quality //加载默认保存图片质量
	} else if q > 100 {
		q = 100
	}

	save := strings.Trim(req.Form.Get("s"), " ")
	if len(save) == 0 {
		s = z.context.Config.Storage.SaveNew
	} else {
		s = str2Int(save)
		if s != 1 {
			s = 0
		}
	}

	format := strings.Trim(req.Form.Get("f"), " ")
	if len(format) == 0 {
		//f = "none"
		//f = imgInfo.Format
		f = z.context.Config.System.Format //加载默认保存图片格式
	} else {
		format = strings.ToLower(format)
		formats := strings.Split(z.context.Config.Storage.AllowedTypes, ",")
		isExist := false
		for _, v := range formats {
			if format == v {
				isExist = true
			}
		}
		if !isExist {
			f = z.context.Config.System.Format
		} else {
			f = format
		}
	}
	// if f == strings.ToLower(imgInfo.Format) {
	// 	f = "none"
	// }

	request := &ZRequest{
		Md5:        md5Sum,
		Width:      w,
		Height:     h,
		Gary:       g,
		X:          x,
		Y:          y,
		Rotate:     r,
		Quality:    q,
		Proportion: p,
		Save:       s,
		Format:     f,
		ImageType:  i,
	}

	z.context.Logger.Debug("request params: md5 : %s, width: %d, height: %d, gary: %d, x: %d, y: %d, rotate: %d, quality: %d, proportion: %d, save: %d, format: %s, imageType: %s", request.Md5, request.Width, request.Height, request.Gary, request.X, request.Y, request.Rotate, request.Quality, request.Proportion, request.Save, request.Format, request.ImageType)

	data, err := z.storage.GetImage(request)

	if err != nil {
		z.doError(writer, err, 500)
		return
	}

	//add etag support
	if z.context.Config.System.Etag == 1 {
		newMd5Sum := gen_md5_str(data)

		ifNoneMatch := req.Header.Get("If-None-Match")
		if len(ifNoneMatch) == 0 {
			writer.Header().Set("Etag", newMd5Sum)
		} else {
			if ifNoneMatch == newMd5Sum {
				z.context.Logger.Debug("Etag Matched Return 304 EVHTP_RES_NOTMOD.")
				z.doError(writer, fmt.Errorf("Not Modified"), http.StatusNotModified)
				return
			} else {
				writer.Header().Set("Etag", newMd5Sum)
			}
		}

	}

	headers := z.context.Config.System.Headers
	if len(headers) > 0 {
		arr := strings.Split(headers, ",")
		for i := 0; i < len(arr); i++ {
			header := arr[i]
			kvs := strings.Split(header, ":")
			writer.Header().Set(kvs[0], kvs[1])
		}
	}

	imageFormat := strings.ToLower(f)
	if contentType, ok := z.contentTypes[imageFormat]; ok {
		writer.Header().Set("Content-Type", contentType)
		writer.Header().Set("Accept-Ranges", "bytes")
		if writer.Header().Get("Content-Encoding") == "" {
			writer.Header().Set("Content-Length", strconv.Itoa(len(data)))
		}
		io.Copy(writer, bytes.NewReader(data))

	} else {
		err = fmt.Errorf("can not found content type!!!")
		z.doError(writer, err, http.StatusForbidden)
		return
	}
}

func (z *ZHttpd) doError(w http.ResponseWriter, err error, statusCode int) {
	http.Error(w, err.Error(), statusCode)
	return
}

func str2Int(str string) int {
	str = strings.Trim(str, " ")
	if len(str) > 0 {
		i, err := strconv.Atoi(str)
		if err != nil {
			return 0
		} else {
			return i
		}
	} else {
		return 0
	}
}
