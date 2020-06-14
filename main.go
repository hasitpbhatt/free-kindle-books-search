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

const _maxPagesToScroll = 243 // can't be more than 400

var (
	_jar       = &cookiejar.Jar{}
	_m         = map[string]string{}
	_purchased = map[string]bool{}
	_invalid   = map[string]bool{}
)

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

const base = "https://www.amazon.in/s?i=digital-text&bbn=1634753031&rh=n%3A1571277031%2Cn%3A1571278031%2Cn%3A1634753031%2Cp_n_feature_three_browse-bin%3A11301931031%2Cp_36%3A-1%2Cp_72%3A1318476031&dc&fst=as%3Aoff&qid=1592140199&rnid=1318475031&ref=sr_nr_p_72_1"

func main() {
	c := &http.Client{
		Jar: _jar,
	}
	_ = c
	for i := 1; i <= _maxPagesToScroll; i++ {
		fmt.Printf("%d,", i)
		b := foo(c, int64(i))
		// fmt.Println(i)
		// b, err := ioutil.ReadFile("page" + strconv.FormatInt(int64(i), 10) + ".html")
		// if err != nil {
		// 	panic(err)
		// }
		if err := parse(b); err != nil {
			panic(err)
		}
		// time.Sleep(4 * time.Second)
	}
	fmt.Println()
	for k, v := range _m {
		if !_purchased[k] && !_invalid[k] {
			fmt.Println(k, fmt.Sprintf("https://amazon.in/dp/%s", k), v)
		}
	}
}

func parse(b []byte) error {
	n, err := html.Parse(bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Parent != nil && n.Data == "img" {
			parent := n.Parent
			if parent.Data == "div" && parent.Parent != nil {
				grandParent := parent.Parent
				if grandParent.Data == "a" {
					alt := ""
					for _, a := range n.Attr {
						if a.Key == "alt" {
							alt = a.Val
							break
						}
					}
					if alt != "" {
						getIDFromLink(grandParent, alt)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return nil
}

func getIDFromLink(n *html.Node, alt string) {
	for _, a := range n.Attr {
		if a.Key == "href" && strings.Contains(a.Val, "/dp/") {
			v := strings.Split(a.Val, "/ref")
			v = strings.Split(v[0], "/dp/")
			_m[v[1]] = alt
			break
		}
	}
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
