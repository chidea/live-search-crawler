package main

import (
  _"fmt"
  "net/http"
  "golang.org/x/net/html"
  _"io/ioutil"
  "log"
  "bytes"
  "strconv"
  "time"
  "database/sql"
  _ "github.com/mattn/go-sqlite3"
)
var nstmt, dstmt, kstmt, ksstmt, tstmt, tsstmt *sql.Stmt
func main(){
  db, err := sql.Open("sqlite3", "./live-search.db")
  if err != nil {
    log.Fatal(err)
    return
  }
  defer db.Close()
  if ! initDB(db) { return }
  for {
    log.Println(time.Now())
    _, err := parse(db, "Naver", "http://www.naver.com/include/realrank.html.09", parseNaver, nstmt)
    if err != nil {
      time.Sleep(1*time.Minute)
      log.Fatal(err)
      continue
    }
    _, err = parse(db, "Daum", "http://www.daum.net", parseDaum, dstmt)
    if err != nil {
      time.Sleep(1*time.Minute)
      log.Fatal(err)
      continue
    }
    time.Sleep(15*time.Minute)
  }
}

func initDB(db *sql.DB) bool{ // return : is success
  _, err := db.Exec(`pragma journal_mode = WAL`)
  if err != nil {
    log.Fatal(err)
    return false
  }
  _, err = db.Exec(`create table if not exists time (time integer not null primary key)`)
  if err != nil {
    log.Fatal(err)
    return false
  }
  _, err = db.Exec(`create table if not exists naver (tid integer not null, kid integer not null, rank integer not null, state text not null, foreign key(tid) references time(time), foreign key(kid) references keyword(rowid))`)
  if err != nil {
    log.Fatal(err)
    return false
  }
  _, err = db.Exec(`create table if not exists daum (tid integer not null, kid integer not null, rank integer not null, state text not null, foreign key(tid) references time(time), foreign key(kid) references keyword(rowid))`)
  if err != nil {
    log.Fatal(err)
    return false
  }
  _, err = db.Exec(`create table if not exists keyword (keyword text not null primary key)`)
  if err != nil {
    log.Fatal(err)
    return false
  }
  tstmt, err = db.Prepare("insert or ignore into time values (?)")
  if err != nil {
    log.Fatal(err)
    return false
  }
  tsstmt, err = db.Prepare("select rowid from time where time=?")
  if err != nil {
    log.Fatal(err)
    return false
  }
  kstmt, err = db.Prepare("insert or ignore into keyword values (?)")
  if err != nil {
    log.Fatal(err)
    return false
  }
  ksstmt, err = db.Prepare("select rowid from keyword where keyword=?")
  if err != nil {
    log.Fatal(err)
    return false
  }
  nstmt, err = db.Prepare("insert into naver values (?, ?, ?, ?)")
  if err != nil {
    log.Fatal(err)
    return false
  }
  dstmt, err = db.Prepare("insert into daum values (?, ?, ?, ?)")
  if err != nil {
    log.Fatal(err)
    return false
  }
  return true
}
func rollDB(tx *sql.Tx, e error) error{
  _ = tx.Rollback()
  return e
}
  
func tranDB(db *sql.DB, stmt *sql.Stmt, t time.Time, rst [10]rank, name string) error{
  tx, err := db.Begin()
  if err != nil { return err }
  
  _, err = tx.Stmt(tstmt).Exec(t.Unix())
  if err != nil { return rollDB(tx, err) }
  var tid int
  err = tx.Stmt(tsstmt).QueryRow(t.Unix()).Scan(&tid)
  if err != nil { return rollDB(tx, err) }
  log.Println("Time", t, "inserted as #", tid)
  
  
  for i, v := range rst{
    _, err = tx.Stmt(kstmt).Exec(v.Keyword)
    if err != nil { return rollDB(tx, err) }
    var kid int
    err = tx.Stmt(ksstmt).QueryRow(v.Keyword).Scan(&kid)
    if err != nil { return rollDB(tx, err) }
    log.Println("Keyword", v.Keyword, "inserted as #", kid)

    rrst, err := tx.Stmt(stmt).Exec(tid, kid, i+1, v.State)
    if err != nil { return rollDB(tx, err) }
    rid, err := rrst.LastInsertId()
    if err != nil { return rollDB(tx, err) }
    log.Println(name, "rank #", i+1, "(", v.State, ") inserted as #", rid)
  }

  err = tx.Commit()
  if err != nil { return rollDB(tx, err) }
  return nil
}

func parse(db *sql.DB, name, url string, parseFn func(*http.Response) ([10]rank, error), stmt *sql.Stmt) ([10]rank, error){
  t := time.Now()
  r, err := http.Get(url)
  if r != nil {
    defer r.Body.Close()
  }
  var rst [10]rank
  if err != nil {
    log.Println("Cannot connect to", name)
    return rst, err
  }
  rst, err = parseFn(r)
  log.Println("#", name,  "#", t, "#")
  if err != nil {
    log.Fatal("Cannot get result of ", name, " error:", err)
    return rst, err
  }
  err = tranDB(db, stmt, t, rst, name)
  if err != nil {
    log.Fatal(err) 
    return rst, err
  }
  return rst, nil
}

type rank struct {
  Keyword string
  State string
}

func parseDaum(r *http.Response) ([10]rank, error){
  var rst [10]rank
  var crt *rank
  depth := 0
  passDepth:= -1
  z := html.NewTokenizer(r.Body)
  for {
    tt := z.Next()
    switch tt {
    case html.ErrorToken:
      return rst, nil;
    case html.TextToken:
      if depth > 5 && passDepth<0 { // (crt == &rst[0] && depth > 6 ) || ( crt != &rst[0] && depth > 5) {
        t := string(bytes.TrimSpace(z.Text()))
        if t == "" { continue }
        if crt.Keyword == ""{
          crt.Keyword = t
        }else if crt.State == "" {
          if len(t) == 12 {
            crt.State = t
          }else{
            crt.State = t[4:]
          }
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
  return rst, nil
}

func parseNaver(r *http.Response) ([10]rank, error){
  var rst [10]rank
  var crt *rank
  depth := 0
  z := html.NewTokenizer(r.Body)
  for {
    tt := z.Next()
    switch tt {
    case html.ErrorToken:
      return rst, nil;
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
            return rst, nil
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
  return rst, nil
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
