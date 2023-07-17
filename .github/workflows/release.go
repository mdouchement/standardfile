package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type controller struct {
	epath   string
	token   string
	baseurl string
	event   event
}

type event struct {
	Release struct {
		ID              int64  `json:"id"`
		URL             string `json:"url"`
		AssetsURL       string `json:"assets_url"`
		UploadURL       string `json:"upload_url"`
		TagName         string `json:"tag_name"`
		TargetCommitish string `json:"target_commitish"`
		Name            string `json:"name"`
		Draft           bool   `json:"draft"`
		Prerelease      bool   `json:"prerelease"`
	} `json:"release"`
}

func main() {
	c := &controller{
		// https://docs.github.com/en/actions/reference/environment-variables#default-environment-variables
		epath:   os.Getenv("GITHUB_EVENT_PATH"),
		token:   os.Getenv("GITHUB_TOKEN"),
		baseurl: "https://api.github.com/",
	}
	fmt.Println("Token size:", len(c.token))

	//
	//

	err := c.load()
	if err != nil {
		log.Fatal("load: ", err)
	}

	fmt.Println("RELEASE:", c.event.Release.ID)
	fmt.Println("URL:", c.event.Release.URL)
	fmt.Println("UPLOAD_URL:", c.event.Release.UploadURL)
	fmt.Println("TAG:", c.event.Release.TagName)

	//

	err = c.update()
	if err != nil {
		log.Fatal("update: ", err)
	}

	//

	err = c.upload()
	if err != nil {
		log.Fatal("upload: ", err)
	}
}

// https://docs.github.com/en/developers/webhooks-and-events/webhooks/webhook-events-and-payloads#release
func (c *controller) load() error {
	payload, err := os.ReadFile(c.epath)
	if err != nil {
		return fmt.Errorf("readfile: %w", err)
	}

	err = json.Unmarshal(payload, &c.event)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	if !strings.HasPrefix(c.event.Release.URL, c.baseurl) {
		return fmt.Errorf("bad url prefix: %s", c.event.Release.URL)
	}
	if !strings.HasPrefix(c.event.Release.AssetsURL, c.baseurl) {
		return fmt.Errorf("bad upload url prefix: %s", c.event.Release.UploadURL)
	}

	return nil
}

func (c *controller) update() error {
	fmt.Println("=> Updating readme")

	readme, err := c.readme()
	if err != nil {
		return fmt.Errorf("readme: %w", err)
	}
	fmt.Println(readme)

	body, err := json.Marshal(map[string]any{
		"body": readme,
	})

	if err != nil {
		return fmt.Errorf("body: %w", err)
	}

	//

	req, err := c.request("PATCH", c.event.Release.URL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}

	err = c.perform(req, nil)
	if err != nil {
		return fmt.Errorf("perform: %w", err)
	}

	return nil
}

func (c *controller) readme() (string, error) {
	checksums, err := os.ReadFile("dist/checksum.txt")
	if err != nil {
		return "", fmt.Errorf("readfile: %w", err)
	}

	return fmt.Sprintf("```\n%s\n\n%s\n```", runtime.Version(), bytes.TrimSpace(checksums)), nil
}

func (c *controller) upload() error {
	baseurl := c.event.Release.UploadURL
	if idx := strings.IndexRune(baseurl, '{'); idx > 0 {
		baseurl = baseurl[:idx]
	}

	//

	err := filepath.Walk("dist/", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		fmt.Println("=> Uploading", path)

		//

		u, err := url.Parse(baseurl)
		if err != nil {
			return fmt.Errorf("parse upload url: %w", err)
		}
		params := u.Query()
		params.Set("name", info.Name())
		u.RawQuery = params.Encode()

		//

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
		defer f.Close()

		//

		req, err := c.request("POST", u.String(), f)
		if err != nil {
			return fmt.Errorf("request: %w", err)
		}

		req.ContentLength = info.Size()
		req.Header.Set("Content-Type", "application/octet-stream")

		//

		err = c.perform(req, nil)
		if err != nil {
			return fmt.Errorf("perform: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("walk: %w", err)
	}

	return nil
}

/////////////////////
//                 //
// HTTP            //
//                 //
/////////////////////

func (c *controller) request(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
	}

	return req, nil
}

func (c *controller) perform(req *http.Request, v any) error {
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		return fmt.Errorf("status: %s", response.Status)
	}

	if v != nil {
		codec := json.NewDecoder(response.Body)
		if err = codec.Decode(v); err != nil {
			return fmt.Errorf("decode: %s", response.Status)
		}
	}

	return nil
}
