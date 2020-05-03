package main

import (
	"bytes"
	"flag"
	"github.com/otiai10/gosseract/v2"
	"image"
	"image/jpeg"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// this function exists because I have an MJPEG source returns an invalid boundary
func parseMediaType(contentType string) (mediaType string, params map[string]string, err error) {
	mediaType, params, err = mime.ParseMediaType(contentType)
	if err == nil {
		return mediaType, params, err
	}

	contentType = strings.ReplaceAll(contentType, "; ", ";")
	contentType = strings.ReplaceAll(contentType, ";", "; ")
	parts := strings.Split(contentType, "; boundary=")

	params = make(map[string]string)
	params["boundary"] = parts[1]

	return parts[0], params, nil
}

func getRequest(url, username, password string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return req, err
	}

	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	return req, nil
}

func doRequest(req *http.Request, timeout time.Duration) (*http.Response, error) {
	client := http.Client{
		Timeout: timeout,
	}

	return client.Do(req)
}

func handleResponse(resp *http.Response) (*multipart.Reader, error) {
	_, param, err := parseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	return multipart.NewReader(resp.Body, param["boundary"]), nil
}

func handlePart(part *multipart.Part, callback func(image.Image) error) error {
	img, err := jpeg.Decode(part)
	if err != nil {
		return err
	}

	return callback(img)
}

func handleParts(reader *multipart.Reader, callback func(image.Image) error) error {
	for {
		part, err := reader.NextPart()
		if err != nil {
			log.Fatal(err)
		}

		err = handlePart(part, callback)
		if err != nil {
			return err
		}
	}
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	url := flag.String("url", "", "url")
	username := flag.String("username", "", "username (optional)")
	password := flag.String("password", "", "password (optional)")

	flag.Parse()

	if *url == "" {
		log.Fatal("need to specify -username flag")
	}

	req, err := getRequest(*url, *username, *password)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := doRequest(req, time.Second*5)
	if err != nil {
		log.Fatal(err)
	}

	reader, err := handleResponse(resp)
	if err != nil {
		log.Fatal(err)
	}

	client := gosseract.NewClient()
	defer client.Close()

	err = handleParts(
		reader,
		func(image image.Image) error {
			b := make([]byte, 0)

			buf := bytes.NewBuffer(b)

			err = jpeg.Encode(buf, image, nil)
			if err != nil {
				log.Fatal(err)
			}

			err = client.SetImageFromBytes(buf.Bytes())
			if err != nil {
				log.Fatal(err)
			}

			text, err := client.Text()
			if err != nil {
				log.Fatal(err)
			}

			log.Printf("text: %+v", text)

			return nil
		},
	)
	if err != nil {
		log.Fatal(err)
	}
}
