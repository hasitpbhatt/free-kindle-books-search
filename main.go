package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

const _maxPagesToScroll = 400 // can't be more than 400

var (
	_jar       = &cookiejar.Jar{}
	_m         = map[string]book{}
	_purchased = map[string]bool{}
	_invalid   = map[string]bool{}
)

type book struct {
	asin  string
	href  string
	title string
	stars string
}

func init() {
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		panic(err)
	}
	_jar = jar
	fill("free.txt", _purchased)
	fill("invalid.txt", _invalid)
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
		m[record[0]] = true
	}
}

const base = "https://www.amazon.in/s?i=digital-text&bbn=1634753031&rh=n%3A1571277031%2Cn%3A1571278031%2Cn%3A1634753031%2Cp_36%3A-1%2Cp_n_feature_three_browse-bin%3A10837939031%7C10837942031%7C11301931031%2Cp_72%3A1318476031&dc&qid=1594215302&rnid=1318475031&ref=sr_nr_p_72_1"

func main() {
	c := &http.Client{
		Jar: _jar,
	}
	_ = c
	for i := 1; i <= _maxPagesToScroll; i++ {
		fmt.Printf("%d,", i)
		b := foo(c, int64(i))
		// b, err := ioutil.ReadFile("page" + strconv.FormatInt(int64(i), 10) + ".html")
		// if err != nil {
		// 	panic(err)
		// }
		if err := parse2(b); err != nil {
			panic(err)
		}
		// time.Sleep(4 * time.Second)
	}
	fmt.Println()
	for _, book := range _m {
		fmt.Println(book.title)
		fmt.Println(book.href, book.stars)
		fmt.Println("----------------------")
	}
}

func parse2(b []byte) error {
	n, err := html.Parse(bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Data == "div" {
			if book := getBook(n); book != nil {
				if !_purchased[book.asin] && !_invalid[book.asin] {
					_m[book.asin] = *book
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return nil
}

func getBook(n *html.Node) *book {
	var asin *string
	asin = getString(n, "data-asin")
	if asin == nil {
		return nil
	}
	c := n.FirstChild                                     // div [{ class sg-col-inner}]
	l := c.FirstChild.NextSibling                         // span [{ cel_widget_id MAIN-SEARCH_RESULTS} { class celwidget slot=MAIN template=SEARCH_RESULTS widgetId=search-results}]
	m := l.FirstChild.NextSibling                         // div [{ class s-include-content-margin s-border-bottom s-latency-cf-section}]
	p := m.FirstChild.NextSibling                         // div [{ class a-section a-spacing-medium}]
	g := p.FirstChild.NextSibling                         // empty row
	g = g.NextSibling.NextSibling                         // original row : div [{ class sg-row}]
	q := g.FirstChild.NextSibling.NextSibling.NextSibling // div [{ class sg-col-4-of-12 sg-col-8-of-16 sg-col-16-of-24 sg-col-12-of-20 sg-col-24-of-32 sg-col sg-col-28-of-36 sg-col-20-of-28}]
	// print("q", q)
	t := q.FirstChild // div [{ class sg-col-inner}]
	// print("t", t)
	k := t.FirstChild.NextSibling
	k1 := k.FirstChild.NextSibling
	k2 := k1.FirstChild
	k3 := k2.FirstChild.NextSibling // div [{ class a-section a-spacing-none}]
	k4 := k3.FirstChild.NextSibling.FirstChild.NextSibling
	href := getString(k4, "href")

	b := &book{
		asin:  *asin,
		href:  "https://amazon.in" + *href,
		title: k4.FirstChild.NextSibling.FirstChild.Data,
	}
	if k3.NextSibling.NextSibling != nil {
		k31 := k3.NextSibling.NextSibling // div [{ class a-section a-spacing-none a-spacing-top-micro}]
		k32 := k31.FirstChild.NextSibling.FirstChild.NextSibling
		stars := getString(k32, "aria-label")
		b.stars = *stars
	}
	return b
}

func getString(n *html.Node, name string) *string {
	for _, a := range n.Attr {
		if a.Key == name && a.Val != "" {
			return &a.Val
		}
	}
	return nil
}

func foo(c *http.Client, page int64) (bytes []byte) {
	url := base
	if page > 1 {
		url = base + "&page=" + strconv.FormatInt(page, 10)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.97 Safari/537.36")

	resp, err := c.Do(req)
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	// ioutil.WriteFile("page"+strconv.FormatInt(page, 10)+".html", b, 0777)
	return b
}

func print(title string, n *html.Node) {
	fmt.Println(title, n.Data, n.DataAtom.String(), n.Attr)
}

/* Autoadd cart: not working
for (var i=1;i<10;i++) {
  var x = window.open('https://amazon.in/dp/' + arr[i]);
  setTimeout(2000);
  console.log("x",x)
  var y = x.opener.document.getElementById('add-to-ebooks-cart-button');
  y.click();
  x.close();
}
*/

/* Open pages
for(var i=0;i<arr.length;i++) {
  if( (i%10) == 0) {
    alert("aa");
  }
  window.open("https://amazon.in/dp/" + arr[i], "_blank");
}
*/
