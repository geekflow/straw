# straw
Telegraf를 기반으로 뜯어보며 연습하는 사이드 프로젝트.

## 테스트

    docker run --rm --name influxdb -p 8083:8083 -p 8086:8086 influxdb

    go run cmd/straw.go -config=./etc/straw.conf
    
    docker run --rm --name chronograf -p 8888:8888 chronograf


## 참고 링크

* [Telegraf] [1]
* [Telegraf Docker][2]
* [Chronograf Docker][2]

[1]: https://github.com/influxdata/telegraf "Telegraf"
[2]: https://hub.docker.com/_/telegraf "Telegraf Docker"
[3]: https://hub.docker.com/_/chronograf "Chronograf Docker"
