package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v1"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary
var _client = &http.Client{}
var _buf = &bytes.Buffer{}

var base = "https://www.amazon.in/s?i=digital-text&bbn=1634753031&rh=n%3A1571277031%2Cn%3A1571278031%2Cn%3A1634753031%2Cp_36%3A-1%2Cp_n_feature_three_browse-bin%3A10837939031%7C10837942031%7C11301931031&dc&qid=1600193250&rnid=10837938031&ref=sr_nr_p_n_feature_three_browse-bin_5"

const _startPage = 1
const _maxPagesToScroll = 400 // can't be more than 400

var (
	_jar       = &cookiejar.Jar{}
	_m         = map[string]book{}
	_purchased = map[string]bool{}
	_invalid   = map[string]bool{}
)

var config = struct {
	CSRF            string `yaml:"csrf"`
	CSRFCheckout    string `yaml:"csrfCheckout"`
	Cookies         string `yaml:"cookies"`
	CookiesCheckout string `yaml:"cookiesCheckout"`
}{}

type book struct {
	asin  string
	href  string
	title string
	stars string
}

func getCookiesIn() []*http.Cookie {
	return getCookies(config.Cookies)
}

func getCookies(cookieStr string) []*http.Cookie {
	rawRequest := fmt.Sprintf("GET / HTTP/1.0\r\n%s\r\n\r\n", cookieStr)

	req, _ := http.ReadRequest(bufio.NewReader(strings.NewReader(rawRequest)))

	return req.Cookies()
}

func setCookies() {
	u, _ := url.Parse("https://www.amazon.in")
	_jar.SetCookies(u, getCookiesIn())
}

func setBrowserHeader(req *http.Request) {
	req.Header.Set("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.100 Safari/537.36")
}

var cmdVarArray []string

func checkout() {
	fmt.Println("Checking out books...")
	cmdVarArray = []string{
		`--verbose`,
		`https://www.amazon.in/api/bifrost/acquisitions/v1/collections/midgard/bundles/Active`,
		`-H`,
		`'content-type: application/x-www-form-urlencoded'`,
		`-H`,
		`'user-agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.100 Safari/537.36'`,
		`-H`,
		fmt.Sprintf(`$'%s'`, strings.TrimRight(config.CookiesCheckout, "\n")),
		`--data`,
		fmt.Sprintf(`'csrf=%s&x-client-id=ebook-cart&payment.mode=checkout&proceedToCheckout=1'`, config.CSRFCheckout),
		`--compressed`,
	}
	cmd := exec.Command("curl", cmdVarArray...)
	out, err := exec.Command("/bin/bash", "-c", cmd.String()).CombinedOutput()
	if err != nil {
		fmt.Println(exec.Command("/bin/bash", "-c", cmd.String()).String())
		fmt.Println(string(out))
		log.Fatal(err)
	}
}

func addToCart(c *http.Client, asin string) (int, float64, bool) {
	fmt.Println("adding to cart...")
	jsonStr := fmt.Sprintf(`{"asin":"%s","csrf":"%s","action":"add"}`, asin, config.CSRF)
	req, err := http.NewRequest(http.MethodPost, "https://www.amazon.in/api/bifrost/bundles/v1/collections/midgard/bundles/Active?&x-client-id=ebook-cart", bytes.NewBufferString(jsonStr))
	if err != nil {
		panic(err)
	}

	setBrowserHeader(req)
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "application/json")
	req.Header.Set("accept-encoding", "gzip, deflate, br")
	req.Header.Set("accept-language", "en-US,en;q=0.9,gu;q=0.8")
	req.Header.Set("x-requested-with", "XMLHttpRequest")

	resp, err := c.Do(req)

	r, err := gzip.NewReader(resp.Body)
	if err != nil {
		log.Fatalf("unable to read gzip")
	}
	body, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalf("unable to read body")
	}

	type x struct {
		Resources struct {
			DetailedStatus string `json:"detailedStatus"`
			ReferenceData  struct {
				Bundle struct {
					Summary struct {
						TotalItemCount               float64 `json:"totalItemCount"`
						TotalPriceWithTaxAndDiscount struct {
							Amount float64 `json:"amount"`
						} `json:"totalPriceWithTaxAndDiscount"`
					} `json:"summary"`
				} `json:"bundle"`
			} `json:"referenceData"`
		} `json:"resources"`
	}
	m := x{}
	err = json.Unmarshal(body, &m)
	if err != nil {
		log.Fatal("Unable to marshal")
	}
	fmt.Println(m)
	details := m.Resources.ReferenceData.Bundle.Summary
	fmt.Println(asin, m.Resources.DetailedStatus)
	alreadyOwned := m.Resources.DetailedStatus == "alreadyOwned" || m.Resources.DetailedStatus == "ok"
	if m.Resources.DetailedStatus == "limitExceeded" {
		time.Sleep(5 * time.Second)
		checkout()
		return addToCart(c, asin)
	}
	return int(details.TotalItemCount), details.TotalPriceWithTaxAndDiscount.Amount, alreadyOwned
}

func init() {
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		panic(err)
	}
	_jar = jar
	fill("free.txt", _purchased)
	fill("invalid.txt", _invalid)

	yamlFile, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		fmt.Printf("Error reading YAML file: %s\n", err)
		return
	}

	err = yaml.Unmarshal(yamlFile, &config)
	config.Cookies = strings.TrimRight(config.Cookies, "\n")
	setCookies()
}

func fill(path string, m map[string]bool) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	r := csv.NewReader(strings.NewReader(string(b)))
	records, err := r.ReadAll()
	if err != nil {
		panic(err)
	}
	for _, record := range records {
		m[strings.Split(record[0], "#")[0]] = true
	}
}

func getNumberOfPages(c *http.Client) int {
	bts := getPage(c, base)
	ioutil.WriteFile("ssss.html", bts, 0777)
	d, err := goquery.NewDocumentFromReader(bytes.NewBuffer(bts))
	if err != nil {
		log.Fatal(err)
	}
	d = goquery.NewDocumentFromNode(d.Find(".a-pagination").Get(0).LastChild.PrevSibling.PrevSibling)
	htmlD, _ := d.Html()
	pages, _ := strconv.Atoi(htmlD)
	fmt.Println("Max pages found to be:", pages)
	return pages
}

func main() {
	c := &http.Client{
		Jar: _jar,
	}
	_client = c
	startPage := _startPage
	maxPagesToScroll := _maxPagesToScroll
	if len(os.Args) > 1 {
		base = os.Args[1]
	}
	pages := getNumberOfPages(c)
	if len(os.Args) > 2 {
		startPage, _ = strconv.Atoi(os.Args[2])
	}
	if len(os.Args) > 3 {
		maxPagesToScroll, _ = strconv.Atoi(os.Args[3])
	} else {
		maxPagesToScroll = pages
	}
	checkout()
	for i := startPage; i <= maxPagesToScroll; i++ {
		fmt.Println("Page: ", i)
		b := foo(c, int64(i))
		if err := parse2(b); err != nil {
			panic(err)
		}
	}
	fmt.Println("Found", len(_m), "new books")
	checkout()
	ioutil.WriteFile("downloaded.txt", _buf.Bytes(), 0777)
}

func parse2(b []byte) error {
	n, err := html.Parse(bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	goq := goquery.NewDocumentFromNode(n)
	goq.Find("div[data-asin]").Each(
		func(i int, s *goquery.Selection) {
			asin, exists := s.Attr("data-asin")
			if !exists || asin == "" {
				return
			}
			if _purchased[asin] || _invalid[asin] {
				return
			}
			b := book{
				asin: asin,
			}
			s.Find("a[href]").Each(
				func(j int, s *goquery.Selection) {
					href, exists := s.Attr("href")
					if !exists || !strings.Contains(href, "/") {
						return
					}
					b.href = "https://amazon.in" + href
					rating := ""
					ratingNodes := s.Find(".a-icon-alt")
					if ratingNodes.Size() > 0 {
						rating = ratingNodes.Text()
						b.stars = rating
					}
				},
			)
			z := `₹0 to buy`
			valid := false
			s.Find(`span[dir="auto"]`).Each(
				func(j int, s *goquery.Selection) {
					if !strings.Contains(s.Text(), z) {
						return
					}
					valid = true
					fmt.Println(asin, s.Text())
				},
			)
			if b.href != "" {
				_m[asin] = b
				book := b
				items, price, alreadyOwned := addToCart(_client, book.asin)
				if alreadyOwned {
					_buf.WriteString(book.asin + "\n")
				}
				if price > 0 {
					log.Fatalf("Non-free book found")
				}
				if items == 10 {
					checkout()
				}
			} else {
				fmt.Println("href is empty?", b.href, valid)
			}
		},
	)
	return nil
}

func getPage(c *http.Client, url string) []byte {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		panic(err)
	}

	setBrowserHeader(req)

	resp, err := c.Do(req)
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	// ioutil.WriteFile("page"+strconv.FormatInt(page, 10)+".html", b, 0777)
	return b
}

func foo(c *http.Client, page int64) (bytes []byte) {
	url := base
	if page > 1 {
		url = base + "&page=" + strconv.FormatInt(page, 10)
	}
	return getPage(c, url)
}

func print(title string, n *html.Node) {
	fmt.Println(title, n.Data, n.DataAtom.String(), n.Attr)
}
