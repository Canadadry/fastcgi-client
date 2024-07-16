package client

import (
	"app/fcgi/fcgiclient"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
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
	fs.StringVar(&env, "env", env, "request env as json")
	fs.StringVar(&header, "header", header, "request header as json")
	fs.BoolVar(&help, "help", help, "print cmd help")
	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("cannot parse argument : %w", err)
	}
	if help {
		fs.PrintDefaults()
		return nil
	}
	if env != "" {
		f, err := os.Open(env)
		if err != nil {
			return fmt.Errorf("cannot open env file : %w", err)
		}
		defer f.Close()
		err = json.NewDecoder(f).Decode(&req.Env)
		if err != nil {
			return fmt.Errorf("cannot read env json data : %w", err)
		}
	}

	err = json.Unmarshal([]byte(header), &req.Header)
	if err != nil {
		return fmt.Errorf("cannot read header json data : %w", err)
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
