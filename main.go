package main

import (
  "fmt"
  "net/http"
  "golang.org/x/net/html"
  _"io/ioutil"
  "log"
  "bytes"
  "strconv"
  "time"
)

func main(){
  fmt.Println(time.Now())
  parse("Naver", "http://www.naver.com/include/realrank.html.09", parseNaver)
  parse("Daum", "http://www.daum.net", parseDaum)
}

func parse(name, url string, parseFn func(*http.Response) [10]rank) [10]rank{
  r, _ := http.Get(url)
  var rst [10]rank
  if r == nil {
    log.Println("Cannot connect to", name)
  } else{
    defer r.Body.Close()
    //b, _ := ioutil.ReadAll(r.Body)
    //fmt.Println(string(b))
    rst = parseFn(r)
    fmt.Println("#", name,  "#")
    for _, v := range rst{
      fmt.Printf("%-50s\t%s\n", v.Keyword, v.State)
    }
  }
  return rst
}

type rank struct {
  Keyword string
  State string
}

func parseDaum(r *http.Response) [10]rank{
  var rst [10]rank
  var crt *rank
  depth := 0
  passDepth:= -1
  z := html.NewTokenizer(r.Body)
  for {
    tt := z.Next()
    switch tt {
    case html.ErrorToken:
      return rst;
    case html.TextToken:
      if depth > 5 && passDepth<0 { // (crt == &rst[0] && depth > 6 ) || ( crt != &rst[0] && depth > 5) {
        t := string(bytes.TrimSpace(z.Text()))
        if t == "" { continue }
        if crt.Keyword == ""{
          crt.Keyword = t
        }else if crt.State == "" {
          crt.State = t[4:]
        }else {
          crt.State += " " + t
        }
      }
    case html.EndTagToken:
      if depth > 0 {
        if passDepth >= 0 {
          passDepth--
        }else{
          depth--
        }
      }
    case html.StartTagToken:
      if depth == 0 {
        tn, isTa := z.TagName()
        if bytes.Compare(tn, []byte("ol")) == 0 && isTa{
          if bytes.Compare(get_attr("id", z), []byte("realTimeSearchWord")) == 0 {
            depth++
          }
        }
      } else if depth == 3 {
        cls := get_attr("class", z)
        if bytes.HasPrefix(cls, []byte("rank_cont realtime_")) {
          n, _ := strconv.Atoi(string(cls[23:]))
          crt = &rst[n-1]
          depth++
          passDepth = 0
        }
      } else if depth == 5 {
          cls := get_attr("class", z )
          if !bytes.HasPrefix(cls, []byte("ico_daum")) {
            depth ++
          }else {
            passDepth ++
          }
      } else if passDepth < 0{
        depth ++
      } else if passDepth >= 0{
        passDepth ++
      }
    }
  }
  return rst
}

func parseNaver(r *http.Response) [10]rank{
  var rst [10]rank
  var crt *rank
  depth := 0
  z := html.NewTokenizer(r.Body)
  for {
    tt := z.Next()
    switch tt {
    case html.ErrorToken:
      return rst;
    case html.TextToken:
      if depth > 2 {
        t := string(z.Text())
        if crt.State == ""{
          crt.State = t
        }else {
          crt.State += " " + t
        }
      }
    case html.EndTagToken:
      if depth > 0 {
        depth-- 
      }
    case html.StartTagToken:
      tn, _ := z.TagName()
      /*tn, isTa := z.TagName()
      if depth == 0 {
        if bytes.Compare(tn, []byte("ol")) == 0 && isTa{
          for k, v, isTam  := z.TagAttr(); ; k,v,isTam = z.TagAttr() {
            if bytes.Compare(k, []byte("id")) == 0 && bytes.Compare(v, []byte("realrank")) == 0 {
              depth++
            }
            if !isTam {break}
          }
        }
      } else {*/
      if bytes.Compare(tn, []byte("li")) == 0 {
        var rank int
        for k, v, isTam  := z.TagAttr(); ; k,v,isTam = z.TagAttr() {
          if bytes.Compare(k, []byte("id")) == 0 { // ignore last #lastrank item which is duplication of first rank
            return rst
          }else if bytes.Compare(k, []byte("value")) == 0 {
            rank, _ = strconv.Atoi(string(v))
          }
          if !isTam {break}
        }
        depth++
        crt = &rst[rank-1]
      }else{
        depth++
        if bytes.Compare(tn, []byte("a")) == 0 {
          crt.Keyword = string( get_attr("title", z))
        }
      }
    }
  }
  return rst
}

func get_attr(name string, z *html.Tokenizer) []byte{
  for k, v, isTam  := z.TagAttr(); ; k,v,isTam = z.TagAttr() {
    if bytes.Compare(k, []byte(name)) == 0 { // ignore last #lastrank item which is duplication of first rank
      return v
    }
    if !isTam {break}
  }
  return nil
}
