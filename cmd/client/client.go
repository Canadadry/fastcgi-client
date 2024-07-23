package client

import (
	"app/fcgi/fcgiclient"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

const Action = "client"

func Run(args []string) error {

	cwd, _ := os.Getwd()
	rawUrl := "/"
	req := fcgiclient.Request{
		Method:       "GET",
		DocumentRoot: cwd,
		Index:        "index.php",
	}
	host := "127.0.0.1:9000"
	env := ""
	header := "{}"
	help := false
	fs := flag.NewFlagSet(Action, flag.ContinueOnError)
	fs.StringVar(&host, "host", host, "php-fmp hostname")
	fs.StringVar(&req.Method, "method", req.Method, "request method")
	fs.StringVar(&rawUrl, "url", rawUrl, "request url")
	fs.StringVar(&req.Index, "index", req.Index, "request index")
	fs.StringVar(&req.DocumentRoot, "document-root", req.DocumentRoot, "request document root")
	fs.StringVar(&req.Body, "body", req.Body, "request body")
	fs.StringVar(&env, "env", env, "request env as json or filename to env.json")
	fs.StringVar(&header, "header", header, "request header as json or filename to header.json")
	fs.BoolVar(&help, "help", help, "print cmd help")
	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("cannot parse argument : %w", err)
	}
	if help {
		fs.PrintDefaults()
		return nil
	}
	err = DecodeOrLoad(env, &req.Env)
	if err != nil {
		return fmt.Errorf("cannot read env data : %w", err)
	}

	err = DecodeOrLoad(header, &req.Header)
	if err != nil {
		return fmt.Errorf("cannot read env data : %w", err)
	}

	req.Url, err = url.Parse(rawUrl)
	if err != nil {
		return fmt.Errorf("cannot parse input url : %w", err)
	}

	conn, err := net.Dial("tcp", host)
	if err != nil {
		return fmt.Errorf("cannot dial php server : %w", err)
	}
	defer conn.Close()

	resp, err := fcgiclient.Do(conn, req)
	fmt.Printf("%#v\n", resp)
	return err
}

func DecodeOrLoad(filename string, data interface{}) error {
	if filename == "" {
		return nil
	}
	f, err := os.Open(filename)
	if err != nil {
		errJson := json.NewDecoder(strings.NewReader(filename)).Decode(data)
		if errJson != nil {
			return fmt.Errorf(
				"cannot open file %s nor decoded it as json value : %w",
				filename,
				errors.Join(err, errJson),
			)
		}
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(data)
	if err != nil {
		return fmt.Errorf("cannot read env json data : %w", err)
	}
	return nil
}
