# live-search-crawler
SQLite crawler which collets live search rank of primary web portal services in Republic Of Korea.

### Usage
`go run main.go` will save live search ranking of both Naver and Daum into `live-search.sqlite` file on every 15 minutes.

### DB structure
  - Table summary
    |table name |columns| | | |
    |-----------|-------|-|-|-|
    |time       |time   | | | |
    |keyword    |keyword| | | |
    |naver      |tid    |kid|rank|state|
    |daum       |tid    |kid|rank|state|
