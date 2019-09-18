package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	httpc "github.com/vislee/go-httppc"
)

var (
	ver        = "1.00"
	addr       = flag.String("addr", "", "Spec service listen addr. 'IP:port'")
	vmHostAddr = flag.String("vmHostAddr", "127.0.0.1", "The proxy protol vm host addr. 'IP[:port]'")
	cliAddr    = flag.String("cliAddr", "127.0.0.1", "The proxy protol client addr. 'IP'")
	testPlay   = flag.String("case", "./case.json", "Test play case")
	proxyAddr  = flag.String("proxyAddr", "", "The proxy model listen addr. '[IP]:port'")
	ppEnable   = flag.Bool("PPEnable", true, "The proxy protocol enable. true or false")
)

type matchKV struct {
	Key string `json:"key"`
	Typ string `json:"type"`
	Val string `json:"value"`
}

func (self *matchKV) Match(v string) (bool, string) {
	msg := fmt.Sprintf("\033[34m%s: got '%s', expected: '%s'\033[0m", self.Key, v, self.Val)

	if (self.Val == "" || len(self.Val) == 0) && (self.Typ == "" || len(self.Typ) == 0) {
		return true, msg
	}

	if self.Typ == "contain" {
		return strings.Contains(v, self.Val), msg

	} else if self.Typ == "regex" {
		rp, err := regexp.Compile(self.Val)
		if err != nil {
			log.Fatalln(err.Error())
			return false, msg
		}

		return rp.Match([]byte(v)), msg
	}

	return self.Val == v, msg
}

type request struct {
	Timeout int64             `json:"timeout,omitempty"`
	Method  string            `json:"method"`
	Url     string            `json:"url"`
	Host    string            `json:"host"`
	Addr    string            `json:"realAddr,omitempty"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func (self *request) newRequest(addr string) (*http.Request, error) {
	method := "GET"
	var b *bytes.Buffer = bytes.NewBufferString("")

	url := fmt.Sprintf("http://%s%s", addr, self.Url)

	if len(self.Body) > 0 {
		method = "POST"
		b = bytes.NewBufferString(self.Body)
	}

	if self.Method != "" {
		method = self.Method
	}

	req, err := http.NewRequest(method, url, b)
	if err != nil {
		return req, err
	}

	req.Host = self.Host

	for k, v := range self.Headers {
		req.Header.Add(k, v)
	}

	return req, nil
}

type response struct {
	Status  matchKV   `json:"status"`
	Headers []matchKV `json:"headers"`
	Cookies []matchKV `json:"cookies"`
	Body    matchKV   `json:"body"`
}

func (self *response) Match(resp *http.Response) bool {
	res := true

	if len(self.Status.Key) == 0 {
		self.Status.Key = "status"
	}
	if ok, msg := self.Status.Match(resp.Status); !ok {
		log.Println(msg)
		res = false
	}

	// resp.Header
	for _, header := range self.Headers {
		var h string

		if header.Key == "set-cookie" || header.Key == "Set-Cookie" {
			for _, ck := range resp.Cookies() {
				if len(h) == 0 {
					h = ck.String()
				} else {
					h = h + ", " + ck.String()
				}
			}

		} else {
			h = resp.Header.Get(header.Key)
		}

		if ok, msg := header.Match(h); !ok {
			log.Println(msg)
			res = false
		}
	}

	// resp cookie
	if len(self.Cookies) > 0 {
		resp_cks := make(map[string]string, len(resp.Cookies()))
		for _, rck := range resp.Cookies() {
			resp_cks[rck.Name] = rck.Value
		}

		for _, ck := range self.Cookies {

			val, ok := resp_cks[ck.Key]
			if !ok {
				val = ""
			}

			if ok, msg := ck.Match(val); !ok {
				log.Println(msg)
				res = false
				break
			}
		}
	}

	// body
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("got resp body error. %s\n", err.Error())
		res = false
	} else {
		resp.Body.Close()
	}

	if len(self.Body.Key) == 0 {
		self.Body.Key = "body"
	}
	if ok, msg := self.Body.Match(string(data)); !ok {
		log.Println(msg)
		res = false
	}

	return res
}

type Case struct {
	Title  string   `json:"title"`
	Delay  int64    `json:"delay"`
	Repeat uint     `json:"repeat"`
	Req    request  `json:"req"`
	Resp   response `json:"resp"`
}

type HttpCli interface {
	Do(req *http.Request) (*http.Response, error)
	SetTimeout(d time.Duration)
	// SetProxyProClientIP(remoteAddr string)
}

func (self *Case) Play(cli HttpCli, addr string) bool {
	var times uint = 0
	res := true

	time.Sleep(time.Duration(self.Delay) * time.Second)

	log.Printf("====[Title:%s][repeat:%d]====\n", self.Title, self.Repeat)

	req, err := self.Req.newRequest(addr)
	if err != nil {
		log.Println(err.Error())
		res = false
		goto endl
	}

	cli.SetTimeout(time.Duration(self.Req.Timeout))

	// cli.SetProxyProClientIP("127.0.0.1")
	// if len(self.Req.Addr) > 0 {
	// 	cli.SetProxyProClientIP(self.Req.Addr)
	// }

	for {
		resp, err := cli.Do(req)
		if err != nil {
			log.Println(err.Error())
			res = false
			goto endl
		}
		if !self.Resp.Match(resp) {
			res = false
			goto endl
		}

		times = times + 1
		if times > self.Repeat {
			break
		}
	}

endl:
	ok := "OK"
	if !res {
		ok = "\033[33m\033[01m\033[05mField\033[0m"
	}
	log.Printf("====[%s]====\n\n", ok)

	return res
}

type cc struct {
	http.Client
}

func NewClient() *cc {
	return &cc{
		http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Timeout: 90 * time.Second,
		},
	}
}

func (self *cc) SetTimeout(d time.Duration) {
	self.Client.Timeout = d * time.Second
}

type TestCases []Case

func casePlay() {
	data, err := ioutil.ReadFile(*testPlay)
	if err != nil {
		log.Println(err.Error())
		return
	}

	var t TestCases
	err = json.Unmarshal(data, &t)
	if err != nil {
		log.Println(err.Error())
		return
	}

	var cli HttpCli
	if *ppEnable {
		pc := httpc.NewProxyProClient()
		pc.NotFollowRedirects()
		pc.SetProxyProServerIP(*vmHostAddr)
		pc.SetProxyProClientIP(*cliAddr)
		cli = pc
	} else {
		cli = NewClient()
	}

	var x, y int = 0, 0
	start := time.Now()
	for _, c := range t {
		y = y + 1
		if !c.Play(cli, *addr) {
			x = x + 1
		}
	}
	dura := time.Now().Sub(start)

	log.Printf("\033[35mField=%d, Cases=%d, %v\033[0m", x, y, dura)
	if x > 0 {
		log.Println("Result: \033[31m\033[01m\033[05mFAIL\033[0m")
	} else {
		log.Println("Result: \033[32mPASS\033[0m")
	}

	return
}

func proxyCopy(errc chan<- error, dst io.Writer, src io.Reader) {
	_, err := io.Copy(dst, src)

	errc <- err
}

func proxy_handler(conn net.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

	var dial net.Dialer
	// upsconn, err := dial.DialContext(ctx, "tcp", "127.0.0.1:8083")
	upsconn, err := dial.DialContext(ctx, "tcp", *addr)

	if cancel != nil {
		cancel()
	}

	if err != nil {
		conn.Write([]byte("ups error: " + err.Error()))
		conn.Close()
		return
	}

	defer conn.Close()
	defer upsconn.Close()

	srvport := strings.SplitN(*addr, ":", 2)[1]
	cliport := strings.SplitN(conn.RemoteAddr().String(), ":", 2)[1]

	s := *vmHostAddr
	if ss := strings.Split(s, ":"); len(ss) == 2 {
		s = ss[0]
		srvport = ss[1]
	}

	if *ppEnable {
		pp := fmt.Sprintf("PROXY TCP4 %s %s %s %s\r\n", *cliAddr, s, cliport, srvport)

		_, err = upsconn.Write([]byte(pp))
		if err != nil {
			conn.Write([]byte("proxy protol error:" + err.Error()))
			conn.Close()
			return
		}
	}

	errc := make(chan error, 1)

	go proxyCopy(errc, conn, upsconn)
	go proxyCopy(errc, upsconn, conn)

	<-errc
}

func proxy() {
	ln, err := net.Listen("tcp", *proxyAddr)
	if err != nil {
		log.Fatalln(err.Error())
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err.Error())
			continue
		}

		d := time.Now().Add(10 * time.Second)
		conn.SetReadDeadline(d)
		go proxy_handler(conn)
	}
}

func caseExample() string {
	var e = make([]Case, 1, 1)
	e[0].Req.Headers = map[string]string{"": ""}
	e[0].Resp.Headers = make([]matchKV, 1, 1)
	e[0].Resp.Cookies = make([]matchKV, 1, 1)

	es, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		log.Println(err.Error())
		return ""
	}

	return string(es)
}

func main() {
	flag.Parse()

	if len(*addr) == 0 {
		if *testPlay == "help" {
			fmt.Println("$ cat case.json")
			fmt.Println(caseExample())
		} else {
			log.Println("Error: Not Spec addr")
		}
		return
	}

	if len(*proxyAddr) > 0 {
		proxy()
	} else {
		casePlay()
	}
}

func init() {
	flag.CommandLine.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Name\n  %s - Play testcase or proxy tool (version:%s)\n", os.Args[0], ver)
		flag.Usage()
		fmt.Fprintf(flag.CommandLine.Output(), "Examples\n\t$ %s -addr=127.0.0.1:80 -vmHostAddr=127.0.0.1 -cliAddr=127.0.0.1 -case=test.json\n\t$ %s -addr=127.0.0.1:80 -vmHostAddr=127.0.0.1:8989 -cliAddr=127.0.0.1 -proxyAddr=:8080\n\t$ %s --case help\n", os.Args[0], os.Args[0], os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Authors\n\tvislee\n")
	}
}
