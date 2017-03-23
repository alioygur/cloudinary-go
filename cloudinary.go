// Package cloudinary cloudinary provides support for managing static assets
// on the Cloudinary service.
package cloudinary

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

type (
	// UploadType image, video, raw
	UploadType string

	// Cloudinary the service
	Cloudinary struct {
		cloudName string
		apiKey    string
		apiSecret string
	}

	// UploadResponse ...
	UploadResponse struct {
		PublicID     string `json:"public_id"`
		Version      uint   `json:"version"`
		Signature    string `json:"signature"`
		Width        int    `json:"width"`
		Height       int    `json:"width"`
		Format       string `json:"format"`
		ResourceType string `json:"resource_type"`
		CreatedAt    string `json:"created_at"`
		Bytes        int    `json:"bytes"`
		URL          string `json:"url"`
		SecureURL    string `json:"secure_url"`
	}

	// APIError ...
	APIError struct {
		Message string `json:"message"`
	}
)

const (
	// cloudname, uploadType, operation:upload,destry, e.g.
	baseURL = "https://api.cloudinary.com/v1_1/%s/%s/%s"
)

// Upload types
const (
	ImageType UploadType = "image"
	VideoType            = "video"
)

// New instances new Cloudinary
// the uri param must be a valid URI with the cloudinary:// scheme.
// e.g. cloudinary://api_key:api_secret@cloud_name
func New(uri string) (*Cloudinary, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "cloudinary" {
		return nil, errors.New("missing cloudinary:// scheme in URI")
	}

	if u.Host == "" {
		return nil, errors.New("no cloud name provided in URI")
	}

	if u.User.Username() == "" {
		return nil, errors.New("no api_key provided in URI")
	}

	secret, exists := u.User.Password()
	if !exists || secret == "" {
		return nil, errors.New("no api secret provided in URI")
	}

	return &Cloudinary{
		cloudName: u.Host,
		apiKey:    u.User.Username(),
		apiSecret: secret,
	}, nil
}

func (e *APIError) Error() string {
	return e.Message
}

// UploadImage uploads image. if name keep "" the file name will be random
func (c *Cloudinary) UploadImage(r io.Reader, name string) (*UploadResponse, error) {
	return c.upload(r, name, ImageType)
}

// UploadVideo uploads video. if name keep "" the file name will be random
func (c *Cloudinary) UploadVideo(r io.Reader, name string) (*UploadResponse, error) {
	return c.upload(r, name, VideoType)
}

func (c *Cloudinary) upload(r io.Reader, name string, ut UploadType) (*UploadResponse, error) {
	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)

	// write public_id
	// if file name provided then set public_id else it will be random
	if name != "" {
		if err := w.WriteField("public_id", name); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	// write api_key
	if err := w.WriteField("api_key", c.apiKey); err != nil {
		return nil, errors.WithStack(err)
	}

	// write timestamp
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	if err := w.WriteField("timestamp", timestamp); err != nil {
		return nil, errors.WithStack(err)
	}

	// write signature
	hash := sha1.New()
	part := fmt.Sprintf("timestamp=%s%s", timestamp, c.apiSecret)
	if name != "" {
		part = fmt.Sprintf("public_id=%s&%s", name, part)
	}
	if _, err := io.WriteString(hash, part); err != nil {
		return nil, errors.WithStack(err)
	}
	signature := fmt.Sprintf("%x", hash.Sum(nil))
	if err := w.WriteField("signature", signature); err != nil {
		return nil, errors.WithStack(err)
	}

	// write file
	fw, err := w.CreateFormFile("file", "file")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if _, err := io.Copy(fw, r); err != nil {
		return nil, errors.WithStack(err)
	}

	// ok, let's close the writer.
	if err := w.Close(); err != nil {
		return nil, errors.WithStack(err)
	}

	uri := fmt.Sprintf(baseURL, c.cloudName, ut, "upload")
	req, err := http.NewRequest(http.MethodPost, uri, buf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	result := new(UploadResponse)
	return result, unmarshalResponse(res, result)
}

// DeleteImage deletes image from cloudinary
func (c *Cloudinary) DeleteImage(name string) error {
	return c.delete(name, ImageType)
}

// DeleteVideo deletes video from cloudinary
func (c *Cloudinary) DeleteVideo(name string) error {
	return c.delete(name, VideoType)
}

// delete deletes resource to uploaded
func (c *Cloudinary) delete(name string, ut UploadType) error {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	data := url.Values{
		"api_key":   []string{c.apiKey},
		"public_id": []string{name},
		"timestamp": []string{timestamp},
	}

	// set signature
	hash := sha1.New()
	part := fmt.Sprintf("public_id=%s&timestamp=%s%s", name, timestamp, c.apiSecret)
	io.WriteString(hash, part)
	data.Set("signature", fmt.Sprintf("%x", hash.Sum(nil)))

	uri := fmt.Sprintf(baseURL, c.cloudName, ut, "destroy")

	res, err := http.PostForm(uri, data)
	if err != nil {
		return errors.WithStack(err)
	}

	var result struct {
		Result string `json:"result"`
	}
	if err := unmarshalResponse(res, &result); err != nil {
		return err
	}

	// wtf cloudinary! why do you return 2x codes on failure?
	if result.Result == "ok" {
		return nil
	}

	return &APIError{result.Result}
}

// unmarshalResponse will unmarshal a http.Response from a Cloudinary API request
// into result, possibly returning an error if the process fails or if the API
// returned an error.
func unmarshalResponse(res *http.Response, result interface{}) error {
	defer res.Body.Close()

	if res.StatusCode > 399 || res.StatusCode < 200 {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return errors.WithStack(err)
		}
		var apiErrorResponse struct {
			Error APIError `json:"error"`
		}
		if err := json.Unmarshal(body, &apiErrorResponse); err != nil {
			return errors.WithStack(err)
		}
		return &apiErrorResponse.Error
	}

	return json.NewDecoder(res.Body).Decode(result)
}
