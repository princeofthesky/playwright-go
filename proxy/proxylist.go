package proxy

import (
	"context"
	"crypto/tls"
	"github.com/playwright-community/playwright-go"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var TestUrl = ""
var UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36"

func TestProxy(url *url.URL) bool {

	dialer, err := proxy.SOCKS5("tcp", url.Host, nil, proxy.Direct)
	// Create a custom transport that uses the SOCKS5 dialer.
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
	}
	if strings.Contains(url.Scheme, "http://") || strings.Contains(url.Scheme, "https://") {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyURL(url),
		}
	}
	// Create an HTTP client that uses the custom transport.
	client := &http.Client{Transport: tr}
	defer client.CloseIdleConnections()
	res, err := client.Get(TestUrl)
	if err != nil {
		println("fail test proxy ", url.String(), " with link ", TestUrl, "err", err.Error())
		return false
	}
	if res != nil {
		println("fail test proxy ", url.String(), " with link ", TestUrl, "res == nil")
		return false
	}
	if res.StatusCode != 200 {
		println("fail test proxy ", url.String(), " with link ", TestUrl, " status code = ", res.StatusCode)
		return false
	}
	return true
}
func GetListFromProxyList(browser playwright.Browser) ([]*url.URL, error) {
	proxys := []*url.URL{}
	page, _ := browser.NewPage(playwright.BrowserNewContextOptions{
		UserAgent: &UserAgent,
	})
	page.Goto("https://free-proxy-list.net/")
	defer page.Close()
	time.Sleep(30 * time.Second)
	dataElements, _ := page.QuerySelectorAll("div[class*=fpl-list] tbody tr:has(td[class*=hm])")
	for _, dataElement := range dataElements {
		infoElements, err := dataElement.QuerySelectorAll("td")
		ip, err := infoElements[0].TextContent()
		port, err := infoElements[1].TextContent()
		https, err := infoElements[6].TextContent()
		if https != "yes" {
			continue
		}
		rawUrl := "https://" + ip + ":" + port
		url, err := url.Parse(rawUrl)
		if err != nil {
			continue
		}
		if TestProxy(url) {
			println("Success find a new proxy" + url.String())
			proxys = append(proxys, url)
		}
	}
	return proxys, nil
}
