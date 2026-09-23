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

## 후속 검토: Linux UDP 배치 I/O

> 상태: 설계 검토 항목이며 아직 구현되지 않았다. 현재 실행 코드는 `net.UDPConn.ReadFromUDP`와 `Conn.Write`를 패킷마다 호출한다.

응답을 받지 않는 UDP fire-and-forget 용도에서는 Linux의 `recvmmsg(2)`와 `sendmmsg(2)`를 이용해 한 번의 syscall로 여러 datagram을 처리할 수 있다. 적용 대상은 Linux UDP 멀티프로세스 모드이며 TCP/TLS 및 요청·응답형 UDP 세션은 대상에서 제외한다.

### 적용 조건

```toml
[runtime]
worker_processes = 4

[servers.udpproxy.udp]
max_requests = 1
max_responses = 0
```

현재 구현에서 `max_responses = 0`은 응답 수신을 비활성화한다는 뜻이 아니라 응답 횟수를 제한하지 않는다는 뜻이다. `max_requests = 1`을 함께 설정해야 세션과 응답 수신 고루틴을 만들지 않는 fire-and-forget 경로를 사용한다.

향후 배치 크기를 설정으로 노출한다면 다음 형태를 사용한다. 아래 `io_batch_size`는 제안된 설정이며 현재는 사용할 수 없다.

```toml
[servers.udpproxy.udp]
max_requests = 1
max_responses = 0
io_batch_size = 32
```

초기 기본값은 32를 사용하고 1로 설정하면 기존 패킷 단위 처리와 동일하게 동작하도록 한다. 운영 환경에서는 16, 32, 64를 비교해 CPU 사용률, 지연 시간 및 drop 수를 기준으로 선택한다.

### 수신 경로

워커별 `SO_REUSEPORT` UDP 소켓과 CPU affinity 구조는 유지한다. 각 워커의 수신 루프만 다음과 같이 변경한다.

```text
SO_REUSEPORT UDP socket
        │
        ▼
recvmmsg(batchSize)
        │
        ├─ 접근 제어
        ├─ 패킷별 백엔드 선택
        └─ 백엔드 소켓별 송신 묶음 생성
```

- `recvmmsg`에 전달할 message descriptor, source address, payload buffer 배열을 워커 시작 시 미리 할당하고 반복해서 재사용한다.
- 기존 최대 UDP payload인 65,507바이트를 계속 지원해야 한다. 메모리 사용량은 대략 `worker_processes × io_batch_size × 65,507`바이트에 descriptor 및 송신 큐 메모리가 추가된다.
- listener의 nonblocking 동작과 Go runtime poller를 유지한다. `EAGAIN`이면 poller가 다음 readable 이벤트를 기다리게 하고, `EINTR`은 같은 batch를 다시 시도한다.
- `MSG_TRUNC`가 확인된 datagram은 전달하지 않고 별도 drop 메트릭을 증가시킨다.
- 한 flow 내부의 입력 순서는 유지하되 UDP 자체는 전달 순서를 보장하지 않는다는 전제를 유지한다.

### 송신 경로

수신 batch의 각 datagram에 대해 기존 balance 정책으로 백엔드를 선택한 후, 같은 backend socket으로 전송할 datagram을 묶는다.

```text
received batch
    │
    ├─ backend A ── sendmmsg(socket A)
    ├─ backend B ── sendmmsg(socket B)
    └─ backend C ── sendmmsg(socket C)
```

- fire-and-forget connection pool의 연결된 UDP socket을 계속 재사용한다.
- 같은 backend로 선택된 datagram만 하나의 `sendmmsg` 호출에 넣는다. 백엔드 수가 많거나 선택 결과가 고르게 분산되면 송신 batch가 작아져 성능 이득도 감소한다.
- `sendmmsg`가 일부 datagram만 전송한 경우 반환된 개수 이후의 message만 재시도해 중복 송신을 방지한다.
- `EAGAIN` 발생 시 무제한 대기하지 않는다. 제한된 worker-local pending queue를 사용하고 큐가 가득 차면 명시적인 drop 정책과 메트릭을 적용한다.
- 패킷별 임시 slice와 goroutine 생성을 피하고, batch가 처리된 뒤 payload buffer를 재사용한다.

### Scheduler 및 통계 처리

소켓 syscall만 배치화하면 기존 scheduler와 통계 채널이 다음 병목이 될 수 있다. 배치 I/O와 함께 아래 변경을 검토한다.

- 패킷마다 동기 채널을 왕복하는 대신 한 batch의 backend 선택 요청을 한 번에 전달한다.
- backend별 Tx byte와 packet 수를 batch 내부에서 합산한 뒤 통계 채널에 한 번만 전달한다.
- 수신 packet, 송신 packet, syscall, partial send, queue drop, truncated packet 수를 워커별 counter로 관리한다.
- 부모 프로세스는 기존 IPC stats 메시지에 워커 counter를 포함해 합산한다.
- roundrobin 등 기존 balance 의미가 batch 경계 때문에 달라지지 않도록 패킷 순서대로 backend를 결정한다.

### 구현 순서

1. 기존 `net.UDPConn` 경로를 유지한 상태에서 Linux 전용 `recvmmsg` 수신 구현과 단위 테스트를 추가한다.
2. `io_batch_size = 1`과 기존 구현의 동작 및 패킷 전달 결과가 같은지 확인한다.
3. 수신 batch만 활성화해 syscall 수, CPU 사용률, PPS 및 drop을 비교한다.
4. backend socket별 `sendmmsg` 송신 batch와 partial-send 처리를 추가한다.
5. scheduler 선택과 통계 업데이트를 batch 단위로 변경한다.
6. 설정으로 기존 경로와 batch 경로를 선택할 수 있게 한 뒤 장시간 부하 및 graceful shutdown을 검증한다.
7. 충분한 결과가 확인된 후에만 batch 경로를 기본값으로 변경한다.

### 예상 성능과 검증 기준

4워커, 작은 UDP 패킷, 다수의 source port, 충분한 CPU 및 NIC queue, 응답 없는 fire-and-forget 조건의 1차 예상치는 다음과 같다. 실제 성능은 CPU, NIC, 백엔드 수, 패킷 크기와 scheduler 비용에 따라 달라지므로 보장값이 아니다.

| 처리 방식 | 예상 입력 처리량 | 예상 NIC 전체 처리량 |
|---|---:|---:|
| 현재 패킷 단위 `net.UDPConn` | 600K~1M PPS | 1.2M~2M PPS |
| `recvmmsg` 수신 batch | 850K~1.2M PPS | 1.7M~2.4M PPS |
| `recvmmsg` + `sendmmsg` | 1M~1.4M PPS | 2M~2.8M PPS |

성능 검증에서는 평균 PPS뿐 아니라 다음 항목을 함께 확인한다.

- p50/p95/p99 전달 지연 시간
- 워커별 패킷 분배 편차
- syscall당 평균 datagram 수
- UDP socket receive/send buffer drop
- pending queue drop과 partial send 횟수
- 워커 CPU 사용률과 scheduler/stat 처리 비중
- backend별 전송 순서와 balance 분포
- 워커 재시작 및 graceful shutdown 중 패킷 손실 범위

하나의 source IP/port로 구성된 단일 UDP flow는 `SO_REUSEPORT` 해시에 의해 한 워커로 고정될 수 있다. 4워커 성능을 검증할 때는 충분히 많은 source port 또는 source IP를 사용해 워커와 NIC queue에 트래픽이 고르게 분산되도록 해야 한다.
