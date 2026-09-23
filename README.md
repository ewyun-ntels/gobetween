[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://vshymanskyy.github.io/StandWithUkraine)

<img src="/logo.png?raw=true" alt="gobetween" width="256px" />

[![Tag](https://img.shields.io/github/tag/yyyar/gobetween.svg)](https://github.com/yyyar/gobetween/releases/latest)
[![Build Status](https://travis-ci.org/yyyar/gobetween.svg?branch=master)](https://travis-ci.org/yyyar/gobetween)
[![Go Report Card](https://goreportcard.com/badge/github.com/yyyar/gobetween)](https://goreportcard.com/report/github.com/yyyar/gobetween)
[![Docs](https://img.shields.io/badge/docs-current-brightgreen.svg)](https://github.com/yyyar/gobetween/wiki)
[![Docker](https://img.shields.io/docker/pulls/yyyar/gobetween.svg)](https://hub.docker.com/r/yyyar/gobetween/)
[![Telegram](https://img.shields.io/badge/telegram-chat-blue.svg)](https://t.me/joinchat/GdlUlg_gRfchk1BORU82PA)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](/LICENSE)


**gobetween** -  modern & minimalistic load balancer and reverse-proxy for the :cloud: Cloud era.

**Current status**: *Maintenance mode, accepting PRs*. Currently in use in several highly loaded production environments.

## Features

* [Fast L4 Load Balancing](https://github.com/yyyar/gobetween/wiki)
  * **TCP** - with optional [The PROXY Protocol](https://github.com/yyyar/gobetween/wiki/Proxy-Protocol) support
  * **TLS** - [TLS Termination](https://github.com/yyyar/gobetween/wiki/Protocols#tls) + [ACME](https://github.com/yyyar/gobetween/wiki/Protocols#tls) & [TLS Proxy](https://github.com/yyyar/gobetween/wiki/Tls-Proxying)
  * **UDP** - with optional virtual sessions and transparent mode


* [Clear & Flexible Configuration](https://github.com/yyyar/gobetween/wiki/Configuration) with [TOML](config/gobetween.toml) or [JSON](config/gobetween.json)
  * **File** - read configuration from the file
  * **URL** - query URL by HTTP and get configuration from the response body 
  * **Consul** - query Consul key-value storage API for configuration

* [Management REST API](https://github.com/yyyar/gobetween/wiki/REST-API)
  * **System Information** - general server info
  * **Configuration** - dump current config 
  * **Servers** - list, create & delete
  * **Stats & Metrics** - for servers and backends including rx/tx, status, active connections & etc.
 
* [Discovery](https://github.com/yyyar/gobetween/wiki/Discovery)
  * **Static** - hardcode backends list in the config file
  * **Docker** - query backends from Docker / Swarm API filtered by label
  * **Exec** - execute an arbitrary program and get backends from its stdout
  * **JSON** - query arbitrary http url and pick backends from response json (of any structure)
  * **Plaintext** - query arbitrary http and parse backends from response text with customized regexp
  * **SRV** - query DNS server and get backends from SRV records
  * **Consul** - query Consul Services API for backends 
  * **LXD** - query backends from LXD

* [Healthchecks](https://github.com/yyyar/gobetween/wiki/Healthchecks)
  * **Ping** - simple TCP ping healthcheck
  * **Exec** - execute arbitrary program passing host & port as options, and read healthcheck status from the stdout
  * **Probe** - send specific bytes to backend (udp, tcp or tls) and expect a correct answer (bytes or regexp)

* [Balancing Strategies](https://github.com/yyyar/gobetween/wiki/Balancing) (with [SNI](https://github.com/yyyar/gobetween/wiki/Server-Name-Indication) and [MaxConnections](https://github.com/yyyar/gobetween/wiki/Balancing#limiting-number-of-active-connections-to-a-backend-since-082) support)
  * **Weight** - select backend from pool based relative weights of backends
  * **Roundrobin** - simple elect backend from pool in circular order
  * **Iphash** - route client to the same backend based on client ip hash
  * **Iphash1** - same as iphash but backend removal consistent (clients remain connecting to the same backend, even if some other backends down)
  * **Leastconn** - select backend with least active connections
  * **Leastbandwidth** -  backends with least bandwidth

* Integrates seamlessly with Docker and with any custom system (thanks to Exec discovery and healthchecks)

* Single binary distribution


## Architecture
<img src="/architecture.png?raw=true" alt="architecture" />

## Usage

* Install with snap: https://snapcraft.io/gobetween
* [Other Installation Options](https://github.com/yyyar/gobetween/wiki/Installation)
* [Read Configuration Reference](https://github.com/yyyar/gobetween/wiki)
* Execute `gobetween --help` for full help on all available commands and options.

## Hacking

* Install Go 1.24+ https://golang.org/
* `$ git clone git@github.com:yyyar/gobetween.git`
* `$ make`
* `$ make run`

### Debug and Test
Run several web servers for tests in different terminals:

* `$ python -m SimpleHTTPServer 8000`
* `$ python -m SimpleHTTPServer 8001`

Instead of Python's internal HTTP module, you can also use a single binary (Go based) webserver like:
https://github.com/udhos/gowebhello

**gowebhello** has support for SSL sertificates as well (**HTTPS** mode), in case you want to do quick demos
of the **TLS+SNI** capabilities of gobetween.

Put `localhost:8000` and `localhost:8001` to `static_list` of static discovery in config file, then try it:

* `$ gobetween -c gobetween.toml`

* `$ curl http://localhost:3000`

## Performance
It's Fast! See [Performance Testing](https://github.com/yyyar/gobetween/wiki/Performance-tests)

## The Name
It's a play on words: gobetween ("go between"). 

Also, it's written in Go, and it's a proxy so it's something that stays between 2 parties :smile:

## License
MIT. See LICENSE file for more details.

## Authors & Maintainers
- [Yaroslav Pogrebnyak](http://pogrebnyak.info)
- [Nick Doikov](https://github.com/nickdoikov)
- [Ievgen Ponomarenko](https://github.com/kikom)
- [Illarion Kovalchuk](https://github.com/illarion)

## All Contributors
- See [AUTHORS](AUTHORS)

## Community
- Join gobetween Telegram group [here](https://t.me/joinchat/GdlUlg_gRfchk1BORU82PA).

## Logo
Logo by [Max Demchenko](https://www.linkedin.com/in/max-demchenko-116170112)

## 개발 내역: UDP 멀티프로세스 런타임

Linux에서 하나의 UDP bind 주소를 여러 워커 프로세스가 `SO_REUSEPORT`로 공유하는 실행 모드를 추가했다. 이 모드는 UDP 처리량 확장을 위한 기능이며 TCP/TLS 서버는 지원하지 않는다.

### 설정 예시

```toml
[runtime]
worker_processes = 4
restart_workers = true      # 생략 시 true
restart_backoff = "1s"      # 생략 시 1s
shutdown_timeout = "10s"   # 생략 시 10s

[metrics]
enabled = true
bind = ":9284"

[defaults]
max_connections = 0
client_idle_timeout = "0"
backend_idle_timeout = "0"
backend_connection_timeout = "0"

[servers.udpproxy]
bind = "0.0.0.0:4000"
protocol = "udp"
balance = "roundrobin"

  [servers.udpproxy.udp]
  max_responses = 1

  [servers.udpproxy.discovery]
  kind = "srv"
  interval = "10s"
  timeout = "2s"
  srv_lookup_pattern = "_backend._udp.my-headless.default.svc.cluster.local."
  srv_lookup_server = "system"
  srv_dns_protocol = "udp"
```

`[api]` 설정은 넣지 않아도 되며 기본값은 비활성이다. 이 UDP 멀티프로세스 런타임에서는 REST API와 profiler를 실행하지 않는다.

### 동작 방식

- 부모 프로세스가 설정 파일 하나로 동일한 UDP 워커들을 실행하고 감시한다.
- 워커별 CPU 집합은 현재 프로세스에 허용된 CPU를 균등 분배해 자동 계산한다. 워커는 CPU affinity를 먼저 적용한 뒤 재실행되며 `GOMAXPROCS`도 할당 CPU 수로 자동 설정된다.
- 워커 수가 사용 가능한 CPU 수보다 많으면 CPU를 순환 배정하므로 일부 워커가 같은 CPU를 공유한다.
- 부모와 워커는 상속된 Unix socketpair(FD 3)에서 길이 헤더가 붙은 JSON 메시지로 ready, heartbeat, stats, shutdown 상태를 교환한다.
- 워커 비정상 종료 시 기본적으로 재시작하며, 부모 종료 시 모든 워커에 graceful shutdown을 요청한 뒤 제한 시간을 넘긴 워커를 종료한다.
- 각 워커가 동일한 SRV discovery를 독립적으로 수행한다. `srv_lookup_server = "system"` 또는 해당 항목 생략 시 `/etc/resolv.conf`의 Kubernetes DNS를 사용하며, 큰 UDP DNS 응답이 잘리면 TCP로 다시 질의한다.
- discovery 갱신 주기에 워커별 지연을 조금 추가해 DNS 질의가 동시에 몰리는 현상을 줄였다.
- 메트릭 HTTP 서버는 부모만 열고 워커 통계를 합산한다. 기존 server/backend 메트릭 외에 `gobetween_worker_up`, `gobetween_worker_restarts_total`을 제공한다.
- pidfile은 부모 프로세스만 기록한다.

`[runtime]`을 생략하거나 `worker_processes`가 0이면 기존 단일 프로세스 모드로 실행된다. 멀티프로세스 모드에서 TCP/TLS 서버, REST API, profiler 설정을 활성화하면 시작 단계에서 오류로 종료한다.
