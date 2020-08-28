# Straw
[Telegraf] [1]를 기반으로 여러가지 프로젝트를 뜯어보고 조합하며 공부하는 사이드 프로젝트 입니다.

## 목표
모니터링에 필요한 데이터 수집과 종류에 대한 이해를 목표로 하고 있습니다.
만약 모니터링 용도로 사용하고자 하는 경우에는 [Telegraf] [1]를 사용하시기를 추천합니다.:)

## Metrics Type
  - Counter
  - Gage

## 테스트

    docker run --rm --name influxdb -p 8083:8083 -p 8086:8086 influxdb

    go run cmd/straw.go -config=./etc/straw.conf
    
    docker run --rm --name chronograf -p 8888:8888 chronograf


## 참고 링크

* [Telegraf][1]
* [Telegraf Docker][2]
* [Chronograf Docker][3]
* [Art of Monitoring][4]

[1]: https://github.com/influxdata/telegraf "Telegraf"
[2]: https://hub.docker.com/_/telegraf "Telegraf Docker"
[3]: https://hub.docker.com/_/chronograf "Chronograf Docker"
[4]: https://artofmonitoring.com/ "Art of Monitoring"
