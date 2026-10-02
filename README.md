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

### 성능 검증 기준

현재 패킷 단위 처리의 PPS와 배치 I/O 전환 효과는 아직 측정하지 않았다. 이전의 600K~1.4M PPS 예상치는 근거가 충분하지 않아 삭제했다. 배치 I/O는 여전히 검토 항목이며 아래 커널 RR 개발과는 별개이다. 입력 300,000 PPS를 4워커에 균등 분배하면 산술 평균은 75,000 PPS/워커이지만, 실제 처리 가능한 성능을 뜻하지 않는다.

성능 검증에서는 평균 PPS뿐 아니라 다음 항목을 함께 확인한다.

- p50/p95/p99 전달 지연 시간
- 워커별 패킷 분배 편차
- syscall당 평균 datagram 수
- UDP socket receive/send buffer drop
- pending queue drop과 partial send 횟수
- 워커 CPU 사용률과 scheduler/stat 처리 비중
- backend별 전송 순서와 balance 분포
- 워커 재시작 및 graceful shutdown 중 패킷 손실 범위

하나의 source IP/port로 구성된 단일 UDP flow는 기본 `SO_REUSEPORT` hash 모드에서 한 워커로 고정될 수 있다. 아래 RR 모드는 같은 flow도 여러 수신 워커에 분배하지만 NIC RSS/IRQ 분배까지 바꾸지는 않는다. 성능 검증은 단일 flow와 다중 flow를 구분하고 워커 CPU뿐 아니라 NIC queue와 softirq도 함께 확인한다.

## 개발 내역: UDP 수신 워커 커널 RR 분배 (2026-10-02)

기존 CPU affinity 기반 UDP 멀티프로세스에 서버별 수신 워커 분배 정책을 추가했다. 모든 워커는 동일한 설정과 백엔드 목록을 사용한다. 부모는 패킷을 중계하지 않으며 패킷별 IPC도 없다.

[Notion 개발 노트 — UDP 멀티프로세스 및 커널 RR 분배](https://app.notion.com/p/3ed539f17105813e9769f34cbdd9d4b4)는 기존 NGINX 개발 노트와 같은 부모 페이지 아래에 생성했다. 작업 순서는 개발·검증 → Notion 기록 → 이 README 정리이다.

### 설정

```toml
[runtime]
worker_processes = 4
restart_workers = true
restart_backoff = "1s"
shutdown_timeout = "10s"

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
  reuse_port_distribution = "rr"
  max_requests = 1
  max_responses = 0

  [servers.udpproxy.discovery]
  kind = "static"
  static_list = [
    "192.168.10.101:5000",
    "192.168.10.102:5000",
    "192.168.10.103:5000"
  ]
```

- `reuse_port_distribution` 생략 또는 `"hash"`: 기존 커널 hash 경로. BPF 리소스나 로드 권한을 추가로 요구하지 않는다.
- `"rr"`: 준비된 수신 워커에 패킷 단위 순환 분배. 같은 source IP/port도 분산된다.
- `balance = "roundrobin"`은 백엔드 선택, `reuse_port_distribution = "rr"`은 수신 워커 선택이다. 서로 다른 단계이다.
- SO_REUSEPORT는 기존 멀티프로세스 런타임이 자동 설정하므로 별도 `reuse_port = true`를 추가하지 않는다.
- `max_responses = 0`은 응답 수 제한 없음이다. 위처럼 `max_requests = 1`을 함께 지정해야 기존 단방향 fire-and-forget 경로를 사용한다.
- 요청/응답도 RR을 사용할 수 있다. `max_requests = 1`을 제거하고 예를 들어 `max_responses = 1`을 사용한다. 검증한 일반 비투명 UDP에서는 응답이 전달을 담당한 워커의 백엔드 소켓으로 돌아온다.
- RR은 메시지가 독립적이고 순서·세션·백엔드/source-port 고정이 필요 없을 때만 사용한다. 워커별 세션과 backend 선택은 공유하지 않으며 처리/응답 순서는 보장하지 않는다.
- Headless Service는 앞의 SRV discovery 예시를 그대로 사용할 수 있다. 워커마다 독립 갱신하므로 순간적인 discovery/healthcheck 상태 차이는 있을 수 있다. 이번 시험은 Kubernetes 배포 검증을 포함하지 않았다.
- `[api]` 생략 시 기본 false이며 이 UDP 런타임에서는 REST API/profiler를 사용하지 않는다. 배치 I/O는 구현하지 않았다.

### 구현 방식과 워커 수명주기

- `src/udpdispatch`에 Linux SK_REUSEPORT eBPF controller와 상속 listener 검증을 추가했다. `github.com/cilium/ebpf v0.17.3`의 Go asm을 사용하므로 clang/CGo나 별도 BPF 오브젝트 생성이 필요 없다.
- RR bind마다 독립된 64비트 atomic fetch-add 카운터, 준비된 워커 목록, ReusePortSockArray를 사용한다. 워커 목록은 Hash map의 불변 스냅샷으로 교체한다. 서로 다른 서버의 RR 순서는 독립적이다.
- 부모는 BPF 정책을 유지하는 제어용 anchor 소켓만 계속 소유한다. anchor는 분배 대상이 아니고 부모가 패킷을 읽지 않는다. 모든 워커 종료 후에도 같은 그룹을 유지한다.
- 부모가 생성한 워커 listener를 FD 4 이후로 상속한다. FD 3은 기존 Unix socketpair IPC이다. 자식의 ready 이후에만 listener를 선택 map에 등록한다.
- 등록 후 부모의 워커 listener 복제 FD를 닫아 자식 종료 시 소켓이 부모 때문에 살아남는 문제를 방지한다.
- IPC 단절, 프로세스 종료, watchdog 및 shutdown 시 워커를 RR 대상에서 제외하고 재시작 워커는 새 ready 후 재등록한다. 오래된 generation 이벤트와 중복 ready도 방어한다.
- 선택 도중 소켓이 닫히면 다른 활성 소켓을 제한된 횟수만큼 재시도한다. 사용 가능한 워커가 없으면 drop이며 hash로 조용히 대체하지 않는다.
- 부모의 `File.Fd()`가 자식과 공유한 소켓을 blocking 모드로 바꾸는 종료 지연 문제를 발견해 `RawConn.Control`로 등록하도록 수정했다. O_NONBLOCK 유지와 강제 kill 없는 종료를 회귀 테스트로 검증했다.

### 실행 조건과 제한

- 2개 이상 RR 워커는 Linux의 SK_REUSEPORT, ReusePortSockArray 및 BPF atomic fetch-add 지원과 BPF 로드 권한이 필요하다. 권한/커널 지원 부족 시 시작 오류로 종료한다.
- 실제 커널 검증은 WSL2 Linux `6.18.40.1-microsoft-standard-WSL2`, Go `1.26.5`, root로 수행했다. 일반 사용자 권한 부족 실패도 별도로 확인했다. 운영 환경의 capability와 보안 정책은 별도 확인해야 한다.
- RR 워커 상한은 256이며, 2개 이상 RR에서는 고정 bind port가 필요하다. 같은 resolved endpoint를 여러 서버가 RR/hash로 중복 사용하는 설정은 거부한다.
- `worker_processes = 1`의 RR은 BPF 없이 기존 listener를 사용한다. RR 설정은 `worker_processes > 0`이 필요하며 기존 단일 프로세스 모드에 조용히 적용하지 않는다.
- 자식 설정과 상속 listener의 서버명/bind/정책을 대조한다. 설정 변경 시 자식만 재시작하지 말고 부모를 함께 재시작한다.
- 다른 프로그램이 동일 reuseport 그룹에 참여해 정책을 바꾸는 구성과 transparent UDP는 이번 검증 범위가 아니다.
- NIC RSS/IRQ, CPU 부족, socket buffer 포화와 백엔드 병목은 RR만으로 해결되지 않는다. 원자 카운터 비용과 처리량/지연 변화는 부하 측정 전까지 미확정이다.

### 검증 결과

- 원본 모듈의 `config`/`udpdispatch` race 테스트: 설정, 고정 flow의 정확한 4워커 순환, IPv6, 0/1/1472/4096/65507바이트 datagram, 동시 송신, close/제외/재등록, 서버별 카운터 독립성, 준비된 워커가 없을 때 모든 worker/anchor로 전달하지 않는 동작을 확인했다.
- supervisor race 테스트: generation 검사, 중복 ready, IPC 종료 시 제외, BPF map 오류 전파, 부분 시작 실패 cleanup을 확인했다.
- 실제 프로세스 통합 테스트: 4워커 RR 단방향과 전체 워커 강제 종료 후 자동 복구, RR 요청/응답, 기존 hash 단방향, RR 1워커를 확인했다. 케이스별 144개 datagram의 payload/분배 및 강제 kill 없는 graceful shutdown을 검사했다.
- 변경 패키지 `go vet`, 전체 소스 `go test -vet=off ./...`, Windows 교차 빌드를 통과했다. Windows에서 RR 실행을 검증한 것은 아니다.
- 기존 `balance/weight.go`의 `Warn` 포맷 vet 오류는 이번 작업과 별개라 수정하지 않았다.

**당시 빌드 검증의 제약 — 후속 replace로 Linux 빌드 해결:** 기존 `github.com/eric-lindau/udpfacade` 저장소/모듈을 당시 환경에서 내려받을 수 없었다(404). 그 시점에는 실제 `go.mod`의 이 의존성을 변경하지 않았고, 독립적인 `config`/`udpdispatch`를 제외한 전체 테스트·통합 바이너리·Windows 빌드는 임시 modfile과 transparent UDP를 거부하는 테스트 전용 shim으로 수행했다. 이후 실제 라이브러리 사본을 replace로 연결한 정상 Linux 빌드와 재검증 결과는 맨 아래 「개발 내역: udpfacade 의존성 복구 및 정상 Linux 검증」에 기록한다. 당시 shim 바이너리를 운영용으로 사용하지 않는다.

아래는 최초 RR 검증의 재현 명령이다. 당시에는 위에서 설명한 임시 modfile을 필요한 빌드/테스트에 사용했다. 현재는 맨 아래의 replace 기반 검증 명령을 사용한다.

```bash
cd /home/bigwo/nTels/debug/Poc/gobetween/src
go test -race ./config ./udpdispatch ./multiprocess
sudo env GOBETWEEN_REQUIRE_BPF=1 go test -race -count=1 ./udpdispatch
go vet ./config ./udpdispatch ./multiprocess ./server/udp ./manager
go test -vet=off ./...

cd ..
go build -o /tmp/gobetween-rr .
sudo env GOBETWEEN_TEST_BINARY=/tmp/gobetween-rr go test -count=1 -run TestUDPWorkerIntegration -v ./test
```

`GOBETWEEN_REQUIRE_BPF=1`은 일반 사용자라도 BPF 초기화를 시도해 권한/로드 실패를 테스트 실패로 처리한다(시험 주소를 사용할 수 없으면 해당 주소 테스트는 skip 가능). `GOBETWEEN_TEST_BINARY`를 지정하지 않으면 실제 프로세스 통합 테스트는 skip한다.

운영 진입 전에 배포 환경의 BPF 권한과 실제 입력 300,000 PPS에서 손실·지연·CPU/워커 분포를 추가 확인해야 한다. 의존성 대체 후 정상 Linux 빌드는 아래 후속 검증에서 확인했다. 기능 테스트 결과로 PPS나 성능 향상률을 추정하지 않는다.

## 상용 OpenShift 환경 설정 조사 (2026-10-02)

> 상태: 사용자가 제공한 상용 환경 명령 출력을 검토한 기록이다. gobetween 파드는 아직 생성하지 않았으며, 해당 클러스터에서 프로그램 실행·BPF 로드·CPU 할당·성능 검증을 수행한 결과가 아니다. 이 조사 기록 작성 당시에는 CPU 배치 코드나 배포 차트를 수정하지 않았다. 이후 코드 구현은 맨 아래의 「개발 내역: 물리 코어 기반 워커 CPU 배치」를 참고한다.

### 확인한 플랫폼 및 커널

| 항목 | 확인값 |
| --- | --- |
| OpenShift Client / Server | `4.22.8` / `4.22.8` |
| Kubernetes | `v1.35.6` |
| Kustomize | `v5.7.1` |
| 조사 노드 | `bdtb-auth41a-policy-wk02.ocp41.skt.local` |
| OS | Red Hat Enterprise Linux CoreOS `9.8.20260727-0` |
| 커널 / kernel-core RPM | `5.14.0-687.31.1.el9_8.x86_64` |
| CPU 모델 | AMD EPYC 9655P 96-Core Processor |
| 소켓 / 물리 코어 / SMT | 1개 / 96개 / 코어당 2개 스레드 |
| 논리 CPU / NUMA | 192개 (`0-191`) / NUMA 노드 1개 |

이 값은 조사한 노드의 결과이며 다른 배포 대상 노드에도 동일한 설정이 적용됐는지는 별도 확인해야 한다.

확인된 BPF 관련 설정은 다음과 같다.

```text
kernel.bpf_stats_enabled = 0
kernel.unprivileged_bpf_disabled = 2
net.core.bpf_jit_enable = 1
net.core.bpf_jit_harden = 1
net.core.bpf_jit_kallsyms = 1
net.core.bpf_jit_limit = 528482304
```

- JIT는 활성화돼 있으며 비특권 BPF 사용은 제한돼 있다. 이 출력만으로 gobetween RR 프로그램의 로드 성공을 확정할 수는 없다.
- `/boot/config-$(uname -r)`와 `/proc/config.gz`를 찾지 못했다. 커널 설정 파일이 보이지 않는다는 사실이 BPF 미지원이라는 뜻은 아니다.
- `bpftool`은 설치돼 있지 않았다. 현재 구현은 Go 라이브러리에서 BPF map과 프로그램을 직접 생성하므로 `bpftool`은 실행 필수 의존성이 아니다.
- upstream Linux 5.14에는 SK_REUSEPORT와 필요한 BPF 기능이 있지만, 실제 RHCOS 커널의 기능·정책 조합은 파드에서 현재 프로그램을 로드해 확인해야 한다. 로컬 WSL 시험 결과를 상용 환경 검증으로 대신하지 않는다.

### BPF 권한 및 SCC 조사

2개 이상 워커의 `reuse_port_distribution = "rr"`은 부모 프로세스가 BPF를 로드한다. `"hash"` 및 RR 1워커 경로에는 이 BPF 권한 요구가 없다.

upstream Linux 5.14의 SK_REUSEPORT 프로그램 로드에는 `CAP_BPF`가 핵심 권한이다. 이 프로그램 유형 때문에 `NET_ADMIN`이 반드시 필요하다고 보지는 않는다. `SYS_ADMIN`도 광범위한 대체 권한이 될 수 있지만 최소 권한 설계의 우선 선택은 아니다. root UID만으로 필요한 capability를 보장하지 않는다. 근거: [Linux 5.14 BPF 프로그램 권한 검사](https://github.com/torvalds/linux/blob/v5.14/kernel/bpf/syscall.c#L1965-L1993), [bpf_capable 검사](https://github.com/torvalds/linux/blob/v5.14/include/linux/capability.h#L244-L247).

사용자가 제공한 `oc get scc` 목록 중 관련 항목을 요약하면 다음과 같다. 표의 CAPS는 목록에 표시된 값이며 실제 파드 권한의 확정값이 아니다.

| SCC | PRIV | 목록에 표시된 CAPS | 판단 |
| --- | --- | --- | --- |
| `restricted-v2`, `restricted-v3` | false | `NET_BIND_SERVICE` | 기본 정책 그대로는 RR용 BPF 권한을 확보하지 못함 |
| `cnag-ipmdn-if`, `cnag-camaraipmdn-if`, `cnag-perf-hostnet` | false | `[]` | 전체 YAML과 사용 권한 확인 필요 |
| `upm-backend-services-scc` | false | `[]` | 전체 YAML과 사용 권한 확인 필요 |
| `upm-common-scc`, `upm-fluent-bit` | true | `[]` | privileged 컨테이너 허용과 실제 파드의 privileged 사용은 구별해야 함 |
| `s1` | false | `SYS_ADMIN`, `SYS_RESOURCE`, `NET_ADMIN` 등 | 광범위한 권한을 허용하므로 그대로 재사용하는 것을 기본안으로 삼지 않음 |
| `privileged` | true | `*` | 권한 범위가 넓으므로 최소 권한 구성의 기본안이 아님 |

`PRIV=true`는 privileged 컨테이너를 허용한다는 의미이지 그 SCC의 모든 파드가 자동으로 privileged라는 뜻이 아니다. CAPS가 빈 목록인 경우에도 `defaultAddCapabilities`, `requiredDropCapabilities`, seccomp, user namespace 등은 전체 YAML에서 확인해야 한다. SCC가 존재하는 것과 대상 ServiceAccount가 사용할 수 있는 것은 별개이다. [OpenShift 4.22 SCC 공식 문서](https://docs.redhat.com/en/documentation/openshift_container_platform/4.22/html/authentication_and_authorization/managing-pod-security-policies)

현재 미확인 사항은 대상 namespace/ServiceAccount, SCC 전체 YAML, RBAC와 SCC 직접 사용자·그룹 지정, 최종 적용 SCC, capability, seccomp 및 SELinux 정책이다. 특히 `restricted-v3`는 파드 user namespace를 요구하므로 namespace 안의 root 권한을 호스트 초기 user namespace의 BPF 권한과 동일하게 취급하면 안 된다.

배포안은 관리자가 승인한 전용 ServiceAccount에 최소 권한 SCC를 연결하는 방향으로 검토한다. 실제 로드에 필요한 capability와 `bpf` syscall 허용 여부를 확인한 뒤 확정하며, 무조건 privileged로 실행하거나 노드의 `unprivileged_bpf_disabled`를 완화하는 방식은 기본안으로 삼지 않는다. 현재 코드의 `rlimit.RemoveMemlock()` 초기화도 시험 대상이며, 메모리 회계 방식과 rlimit 처리 결과에 따라 추가 권한 필요 여부를 확인한다.

### CPU Manager 및 NUMA 설정

노드에서 `chroot /host` 후 `/etc/kubernetes/kubelet.conf`와 `/var/lib/kubelet/cpu_manager_state`를 확인한 결과이다.

```yaml
cpuManagerPolicy: static
cpuManagerPolicyOptions:
  full-pcpus-only: "true"
cpuManagerReconcilePeriod: 5s
reservedSystemCPUs: 0-7
topologyManagerPolicy: single-numa-node
```

```json
{"policyName":"static","defaultCpuSet":"0-191","checksum":3268025208}
```

- `static` 정책의 전용 CPU 할당 대상은 Guaranteed QoS와 정수 CPU 요청 조건을 만족하는 컨테이너이다. CPU 단위는 물리 코어가 아니라 논리 CPU이다.
- `full-pcpus-only=true`이므로 SMT 2개인 이 노드에서는 전용 할당에 완전한 물리 코어 단위가 필요하다. 홀수 CPU 요청 등 SMT 정렬을 만족하지 못하면 `SMTAlignmentError`로 admission이 실패할 수 있다.
- `single-numa-node`는 관련 자원 할당의 NUMA 정렬 정책이다. 조사 노드는 NUMA 1개지만 자원 부족에 따른 배치 실패 가능성까지 없어지는 것은 아니다.
- checkpoint에 `entries`가 없으므로 확인 시점의 전용 CPU 할당 기록은 보이지 않는다. `defaultCpuSet=0-191`을 192개 CPU가 모두 유휴 상태라는 뜻으로 해석하지 않는다.
- 공유 CPU 집합에 reserved CPU가 표시될 수 있다. 이 출력만으로 `reservedSystemCPUs`가 무시됐다고 판단하거나 사용 가능한 전용 CPU 수를 단순히 `192-8`로 확정하지 않는다. 예약 CPU의 실제 SMT 형제 관계도 확인해야 한다.

정책 근거: [Kubernetes CPU Manager 및 full-pcpus-only](https://kubernetes.io/docs/concepts/resource-management/resource-managers/#full-pcpus-only), [Kubernetes v1.35.6 static 정책 구현](https://github.com/kubernetes/kubernetes/blob/v1.35.6/pkg/kubelet/cm/cpumanager/policy_static.go).

### 4워커 배포 시 CPU 자원과 현재 코드의 관계

| gobetween 컨테이너 CPU 요청·제한 | 이 노드의 전용 할당 조건 충족 시 | 4워커에 대한 의미 |
| --- | --- | --- |
| CPU 4개 | 논리 CPU 4개 = 물리 코어 2개 | 4워커를 서로 다른 물리 코어에 배치할 수 없음 |
| CPU 8개 | 논리 CPU 8개 = 물리 코어 4개 | 물리 코어별 1워커 배치에 필요한 자원은 확보하지만 현재 코드만으로 그 배치를 보장하지 않음 |

배포 차트에서는 gobetween 컨테이너에 CPU 8개를 요청·제한하고 Pod가 Guaranteed가 되도록 CPU와 메모리의 request/limit을 맞추는 구성을 검토한다. 다른 컨테이너가 있으면 그 컨테이너들도 QoS 조건을 만족해야 하며 `full-pcpus-only`의 admission 조건도 확인해야 한다. 아래 메모리 1Gi는 형식 예시이며 실측 기반의 용량 권고가 아니다.

```yaml
# gobetween 컨테이너의 resources 예시이며 완성된 배포 차트가 아니다.
resources:
  requests:
    cpu: "8"
    memory: "1Gi"
  limits:
    cpu: "8"
    memory: "1Gi"
```

현재 `src/multiprocess/platform_linux.go`는 `sched_getaffinity`로 프로세스가 사용할 수 있는 논리 CPU를 조회한다. `src/multiprocess/cpu.go`의 `splitCPUs`는 그 CPU ID를 숫자순으로 정렬해 워커 수대로 나눈다. 자식은 나눠 받은 mask에 affinity를 설정하고 `GOMAXPROCS`를 해당 논리 CPU 개수로 자동 설정한다.

즉, 현재도 CPU affinity는 적용되지만 물리 코어의 SMT 형제 관계는 읽지 않는다. CPU 8개/4워커일 때 워커당 논리 CPU 2개와 `GOMAXPROCS=2`가 설정돼도, 서로 다른 워커가 같은 물리 코어의 SMT 스레드를 나눠 갖는 배치가 가능하다. 실제 SMT CPU ID 쌍은 아직 제공되지 않았으므로 `CPU N`과 `CPU N+96`이 형제라고 가정하지 않는다.

또한 현재 분할 코드는 CPU quota를 기준으로 계산하지 않는다. 공유 CPU pool에 quota만 설정한 구성과 Guaranteed 전용 CPU 할당 구성을 구별해야 한다. 부모 프로세스도 같은 컨테이너 CPU 집합을 사용하므로 워커 affinity를 설정했다고 부모·커널 IRQ까지 완전히 격리되는 것은 아니다.

### 조사 당시 CPU 배치 최적화 제안 — 후속 구현은 아래 개발 내역 참고

4워커를 서로 다른 물리 코어에 배치하려면 배포 자원 설정과 코드의 topology 인식이 모두 필요하다. RR 동작 자체에 이 수정이 필수인 것은 아니며 처리량 개선 효과는 아직 측정하지 않았다.

1. 기존 affinity 조회로 허용된 CPU 집합을 먼저 얻는다.
2. `/sys/devices/system/cpu/cpuN/topology/core_cpus_list` 또는 호환 경로 `thread_siblings_list`를 읽어 물리 코어별 SMT 그룹을 구성한다. 범위형 CPU 목록 파싱을 포함하고 허용 집합 밖의 CPU는 할당하지 않는다.
3. 완전한 SMT 그룹을 분리하지 않고 물리 코어 그룹 단위로 워커에 나눈다. 여러 코어를 받는 경우에도 그룹 단위를 유지한다.
4. 자식 affinity 및 `GOMAXPROCS` 자동 설정 흐름은 유지한다. topology 읽기 실패나 물리 코어 부족은 명시적인 오류로 처리한다.
5. 기존 논리 CPU 분할을 기본값으로 유지하고 아래처럼 별도 정책을 선택하는 방안을 검토한다.

```toml
# 조사 당시 제안 예시: core는 채택하지 않았으며 실제 지원 값은 아래의 physical이다.
[runtime]
worker_processes = 4
worker_cpu_policy = "core"
```

수정 대상은 CPU 분할·Linux topology 조회·설정 검증·부모의 정책 선택·로그와 테스트이다. UDP 전달 및 커널 RR 알고리즘을 변경할 필요는 없다. Linux topology 경로 근거: [CPU sysfs ABI](https://www.kernel.org/doc/Documentation/ABI/stable/sysfs-devices-system-cpu).

### 남은 확인 절차

아래는 조사용 명령이며 이번 문서 작업에서 상용 클러스터에 실행하지 않았다. 기존 노드 debug 셸에서 `chroot /host`를 완료했다면 topology를 다음처럼 확인할 수 있다.

```bash
LC_ALL=C lscpu -e=CPU,CORE,SOCKET,NODE,ONLINE
for cpu in 0 1 2 3 4 5 6 7; do
  printf 'cpu%s SMT siblings: ' "$cpu"
  cat "/sys/devices/system/cpu/cpu${cpu}/topology/thread_siblings_list"
done
```

관리용 셸에서는 SCC 상세 설정과 관련 선언을 확인한다. ServiceAccount 접근은 RBAC뿐 아니라 SCC의 `users`/`groups` 지정도 함께 확인한다.

```bash
oc get scc cnag-ipmdn-if -o yaml
oc get scc upm-common-scc -o yaml
oc get scc s1 -o yaml
oc get kubeletconfig -o yaml
```

이후 승인된 시험 파드를 생성한 뒤에만 다음 항목을 검증한다. 파드 생성 및 SCC/RBAC 변경은 별도 배포 작업이며 이번 조사에서 수행하지 않았다.

- 파드의 실제 적용 SCC, `status.qosClass=Guaranteed`, 컨테이너 CPU mask와 SMT 형제 관계.
- 부모 프로세스의 effective capability, seccomp/user namespace/SELinux 조건과 실제 RR BPF 로드 성공.
- 4워커의 시작 로그에 기록된 CPU mask, RR 분배 및 재시작·종료 동작.
- Headless/SRV discovery, UDP 통신과 백엔드 응답 여부에 맞는 설정.
- 실제 `udpfacade` 사본의 replace를 사용한 정상 Linux 빌드는 아래 후속 검증에서 확인했다. 앞 절의 테스트 전용 shim 빌드는 배포 근거로 사용하지 않는다.
- 실제 부하 환경에서 패킷 손실·지연·CPU·NIC queue/softirq 분포. 목표 총 300,000 PPS의 산술 평균 75,000 PPS/워커는 처리 성능 보장이 아니다.

조사 당시 결론은 **CPU 전용 할당 정책은 확인됐지만 파드의 RR 실행 권한과 실제 CPU 배치는 미검증**이라는 것이다. CPU 8개와 물리 코어 인식 분할의 후속 코드 구현은 아래에 기록한다. 배포 차트 적용과 상용 파드 검증은 여전히 남아 있다.

## 개발 내역: 물리 코어 기반 워커 CPU 배치 (2026-10-02)

### 추가 설정과 동작

```toml
[runtime]
worker_processes = 4
worker_cpu_policy = "physical"
```

- `worker_cpu_policy` 생략 또는 `"logical"`: 기존 논리 CPU 번호순 분할을 유지한다. sysfs를 읽지 않으며 기존 CPU 수 초과 시 공유 동작도 유지한다.
- `"physical"`: 현재 affinity로 허용된 CPU를 먼저 조회하고, 물리 코어의 SMT 그룹을 분리하지 않은 채 워커에 배정한다. CPU 번호 간격을 `+48` 또는 `+96`으로 가정하지 않는다.
- Linux의 `core_cpus_list`를 우선 읽고, 파일이 없을 때만 `thread_siblings_list`로 대체한다. CPU 범위 목록을 지원하며 권한 오류나 잘못된 내용은 무시하지 않는다.
- 전체 SMT 그룹이 허용 CPU 집합 안에 있어야 한다. 부분 그룹, topology 누락·불일치, 물리 코어 수보다 많은 워커 요청은 시작 오류이다. 사용 가능한 일부 그룹만 조용히 선택하거나 logical로 대체하지 않는다.
- 완전한 그룹을 물리 코어 개수 기준으로 균등 배정한다. 코어가 워커보다 많으면 워커가 여러 코어를 받는다. 워커당 1코어를 원하면 컨테이너에 그에 맞는 CPU 집합을 확보해야 한다.
- 기존 자식 affinity와 `GOMAXPROCS` 자동 설정은 그대로 재사용한다. SMT 2·논리 CPU 8개·4워커이면 워커당 물리 코어 1개/논리 CPU 2개/`GOMAXPROCS=2`이다.
- 허용 집합 조회와 topology 계산은 부모 시작 시 수행한다. 재시작 자식은 부모가 계산한 동일한 mask를 재사용한다. 실행 중 CPU 집합 변경을 자동 재분배하지 않으므로 자원 또는 정책 변경 시 부모까지 재시작한다.
- 부모 로그에 정책·허용 CPU·워커 수를 추가했고 기존 워커별 CPU mask/GOMAXPROCS 로그를 유지한다.
- `physical`은 `worker_processes > 0`이 필요하다. 오타인 `phygical`, 이전 제안인 `core` 등은 거부한다. 정책은 TOML/JSON 모두 지원한다.

변경은 설정 검증, CPU 분할 및 Linux topology 조회에 한정했다. UDP 수신·전달, 커널 RR, IPC 및 워커 수명주기 알고리즘은 이번 변경에서 수정하지 않았다. 기존 1024비트 affinity mask의 CPU ID 지원 범위(`0-1023`)는 유지한다. CPU quota 판독, IRQ 배치, 부모 전용 코어, 부하 기반 자동 워커 증설은 추가하지 않았다.

### 상용 환경 추가 확인

사용자가 후속으로 제공한 출력에서 아래 항목을 확인했다. 서로 다른 노드의 결과를 섞지 않는다.

| 노드 | 확인 대상 | 확인 내용 |
| --- | --- | --- |
| `bdtb-auth41a-policy-wk02.ocp41.skt.local` | 노드 직접 조사 | 앞 절의 EPYC 9655P·96코어·SMT 2·192논리 CPU와 CPU Manager 정책 |
| `bdtb-auth41a-chrg-wk02.ocp41.skt.local` | `upm-app/tcp-bridge-0-867b5999b8-vjvkp` | 컨테이너 안에서 CPU 0의 `core_cpus_list=0,96` 조회 성공. 이 노드의 CPU Manager 정책은 아직 미확인 |
| `bdtb-auth41a-cems-wk01.ocp41.skt.local` | `upm-ems/was-pharos-86c8c686d8-x8fjk` 및 노드 직접 조사 | EPYC 9454P·48코어·SMT 2·96논리 CPU·NUMA 1개. 파드 셸의 허용 CPU `0-95`, CPU 0~7의 SMT 형제 `N,N+48` 확인 |

`cems-wk01`에서도 `cpuManagerPolicy=static`, `full-pcpus-only=true`, `reservedSystemCPUs=0-7`, `topologyManagerPolicy=single-numa-node`, reconcile 5s를 확인했다. checkpoint는 `policyName=static`, `defaultCpuSet=0-95`이며 `entries`는 보이지 않았다. 전체 허용 CPU 목록이나 checkpoint를 유휴 CPU 수 또는 해당 파드의 전용 CPU 할당 증거로 취급하지 않는다.

두 조사 노드 모두 SMT 2이므로 Guaranteed 및 전용 할당 조건을 충족하는 CPU 8개 요청은 물리 코어 4개에 해당한다. 실제 gobetween 배포 노드·CPU 집합·SCC/BPF 권한은 별도로 확인해야 한다.

### 이번 변경의 검증과 남은 제약

- 원본 모듈의 `go test ./config`와 `go vet ./config`: 기본값, logical/physical, 잘못된 값, worker 수 조건 및 TOML/JSON 디코딩을 검증했다.
- CPU 관련 소스 파일만 지정한 독립 테스트: `+48`/`+96` 형태의 SMT 2, SMT 4, SMT 없는 CPU, 비연속 CPU 번호, 여러 코어의 균등 분할, 1워커, 부분/불일치 그룹, 코어 부족, 기존 logical 동작을 검증했다.
- Linux 테스트 바이너리를 Go 1.26.5로 교차 빌드하고 로컬 WSL2에서 실행했다. 범위 파싱, sysfs 파일 우선순위·fallback·오류 처리와 실제 topology/테스트 스레드 affinity 설정·원래 mask 복원을 확인했다. WSL에서는 논리 CPU 16개와 첫 물리 코어 그룹 `0,1`을 읽었으며 상용 노드 시험이 아니다.
- 실제 affinity 시험은 `GOBETWEEN_TEST_CPU_AFFINITY=1`로 명시적으로 활성화한다. 기본 단위 테스트는 실제 호스트의 전체 SMT 그룹 가용성을 요구하지 않는다.
- CPU 변경 당시 전체 `multiprocess` 테스트는 `udpfacade` 다운로드 문제로 시작하지 못했고 테스트 전용 shim도 사용하지 않았다. 이후 아래의 replace 적용으로 supervisor 설정 검증을 포함한 전체 `multiprocess` 테스트를 완료했다.
- 최초 RR 통합 시험은 CPU 변경 전 결과이다. CPU 변경 후 정상 Linux 바이너리 및 실제 4프로세스 기동·재시작·스레드 affinity 재검증 결과는 아래 후속 기록을 참고한다.
- 배포 차트·SCC·상용 클러스터는 변경하지 않았다. 물리 코어 분리는 워커 사이의 배치 정책이며 부모·OS·IRQ까지 격리한다는 의미가 아니다. 성능 향상률이나 PPS는 측정하지 않았다.

현재 replace 설정과 Go 1.24 이상을 사용하는 Linux 환경의 CPU 검증 명령은 다음과 같다.

```bash
cd /home/bigwo/nTels/debug/Poc/gobetween/src
go test ./config
go vet ./config
go test ./multiprocess
GOBETWEEN_TEST_CPU_AFFINITY=1 go test -count=1 -run TestLivePhysicalCPUAllocation -v ./multiprocess
```

의존성 문제와 분리하여 CPU 로직만 확인하려면 Linux에서 파일 목록을 지정해 테스트할 수 있다.

```bash
cd /home/bigwo/nTels/debug/Poc/gobetween/src/multiprocess
GO111MODULE=off GOBETWEEN_TEST_CPU_AFFINITY=1 go test -v \
  cpu.go cpu_physical.go cpu_physical_test.go \
  cpu_topology_linux.go cpu_topology_linux_test.go
```

## 개발 내역: udpfacade 의존성 복구 및 정상 Linux 검증 (2026-10-02)

### 실제 라이브러리 사본 연결

다운로드할 수 없던 원본 저장소 대신 [illarion/udpfacade](https://github.com/illarion/udpfacade)의 다운로드 가능한 고정 버전을 연결했다. 소스 import와 기존 require 항목은 유지하며 루트 `go.mod`와 `src/go.mod` 양쪽에 다음 replace를 추가했다. 루트 빌드에서는 의존 모듈 내부의 replace만으로는 적용되지 않으므로 두 진입점을 모두 설정한다.

```go
replace github.com/eric-lindau/udpfacade => github.com/illarion/udpfacade v0.0.0-20190425230512-031998cc71fa
```

두 `go.sum`에 실제 다운로드로 확인한 사본 버전과 go.mod checksum을 추가했다. 두 모듈에서 `go mod verify`를 통과했다. 이번 빌드와 시험에는 테스트 전용 shim, 임시 modfile 또는 기능 제거를 사용하지 않았다.

사본의 기본 브랜치는 2019년 4월 커밋 `031998cc71fa`이다. 2019년 6월의 기존 고정 커밋은 GitHub API/raw 경로에서는 조회되지만 Go 다운로드에서는 `unknown revision`이 발생하므로 그대로 대체 경로에 지정하지 않았다. [두 버전의 실제 소스 차이](https://github.com/illarion/udpfacade/compare/031998cc71fa...d8c1c27add16)는 Windows용 빌드 분리 및 대체 구현 추가이며, 비Windows UDP 구현 자체는 같다.

transparent UDP 코드는 실제 라이브러리로 유지했다. 다만 이 라이브러리는 IPv4용이고 저장소는 archived 상태이다. transparent 통신·원본 주소 보존·응답 경로·SCC/raw socket 권한을 이번에 동작 검증한 것은 아니다. 특히 기존 transparent 경로는 일반 요청/응답 경로와 달리 gobetween의 응답 수신 루틴을 시작하지 않는다. 단순히 라이브러리를 연결했다고 transparent의 상용 동작까지 검증됐다고 보지 않는다.

이 사본 버전에는 Windows 대체 구현이 없으므로 Linux 대상 복구이다. Windows 빌드 지원까지 복구했다고 주장하지 않는다. 향후 Windows 지원이 필요하면 OS별 빌드 분리와 명시적 미지원 오류 처리를 별도로 복구해야 한다.

### 정상 빌드 결과

- WSL2 Linux `6.18.40.1-microsoft-standard-WSL2`, Go `1.24.6`에서 기존 Makefile의 `build-static`을 실행했다. Makefile 자체는 변경하지 않았다.
- 명령: `make build-static NAME=gobetween-udp-linux-amd64`.
- 출력: `bin/gobetween-udp-linux-amd64`, ELF 64-bit x86-64, 정적 링크, stripped. 기존 바이너리를 덮어쓰지 않도록 새 이름을 사용했다.
- SHA-256: `3cb1db8f05be8e3b7d471911db2b47aafab22d4109b480919e064b921b2c524f`.
- 루트 모듈과 src 모듈 모두 실제 사본 의존성으로 빌드·테스트했다. Docker 이미지나 상용 파드를 빌드/배포한 결과는 아니다.

### 완료한 재검증

- 전체 src의 `go test -vet=off -count=1 ./...` 및 `go test -race -vet=off -count=1 ./...` 통과.
- `config`/`multiprocess` race 테스트와 실제 topology/affinity 시험 통과. 이전에 실행하지 못했던 supervisor 정책 검증도 포함한다.
- 변경 패키지 `go vet ./config ./multiprocess ./udpdispatch ./server/udp ./manager` 통과.
- 실제 커널 RR race 시험: 고정 flow, IPv6, datagram 길이 경계, 소켓 제외·재등록, 동시 송신, 활성 워커가 없을 때 drop, 서버별 카운터 독립성 통과. root 실행에서는 일반 사용자 권한 부족 시험을 skip하며, 일반 사용자로 그 부정 시험을 별도 실행해 통과했다.
- 루트의 기존 밸런싱·최대 연결 테스트를 실제 의존성과 race 모드로 실행해 통과했다.
- 프로세스 통합 시험 7개 통과: logical RR 단방향/전체 워커 재시작, logical RR 요청·응답, logical hash 단방향, RR 1워커, physical RR 단방향/전체 워커 재시작, physical RR 요청·응답, physical hash 단방향.
- 통합 시험에서 payload, 백엔드/source-port 분포, 부모 메트릭, 전체 자식 강제 종료 후 자동 복구 및 강제 kill 없는 graceful shutdown을 확인했다.
- physical 케이스는 `/proc/PID/task/*/status`에서 실제 자식 스레드의 CPU mask를 읽고 부모 로그와 비교한다. sysfs로 전체 SMT 그룹 배정과 워커 간 물리 코어 비공유를 검사하며, 재시작 전후 동일한 mask가 유지되는지도 검사한다.
- WSL에서 CPU `0-1`, `2-3`, `4-5`, `6-7`이 완전한 SMT 그룹임을 확인한 뒤 시험 프로세스를 CPU `0-7`로 제한해 8논리 CPU·4워커 조건도 재검증했다. 상용 노드의 CPU ID 배치를 이 숫자로 가정하지 않는다.

기본 vet를 포함한 전체 src 테스트는 기존 `balance/weight.go:49,54`의 `Warn` 호출에 `%v`를 사용한 경고로 실패한다. 이번 의존성 복구와 무관한 기존 코드는 수정하지 않았으며 전체 시험에는 `-vet=off`를 사용했다. 테스트 실패를 모두 없앴다거나 전체 vet가 통과했다고 표현하지 않는다.

### 재현 명령 및 남은 작업

다음 명령은 현재 저장된 replace 설정으로 실행하며 임시 modfile이 필요 없다. root가 필요한 BPF/프로세스 시험은 일반 사용자로 테스트 바이너리를 빌드한 뒤 sudo로 실행한다.

```bash
cd /home/bigwo/nTels/debug/Poc/gobetween/src
go mod verify
GOBETWEEN_TEST_CPU_AFFINITY=1 go test -race -count=1 ./config ./multiprocess
go vet ./config ./multiprocess ./udpdispatch ./server/udp ./manager
go test -race -vet=off -count=1 ./...

validation_dir=$(mktemp -d /tmp/gobetween-validation.XXXXXX)
go test -race -c -o "$validation_dir/udpdispatch.test" ./udpdispatch
sudo env GOBETWEEN_REQUIRE_BPF=1 "$validation_dir/udpdispatch.test" -test.v
"$validation_dir/udpdispatch.test" -test.v -test.run=TestRRPermissionFailure

cd ..
go mod verify
make build-static NAME=gobetween-udp-linux-amd64
go test -race -c -o "$validation_dir/integration.test" ./test
"$validation_dir/integration.test" -test.v -test.skip=TestUDPWorkerIntegration
sudo env GOBETWEEN_TEST_BINARY="$PWD/bin/gobetween-udp-linux-amd64" \
  "$validation_dir/integration.test" -test.v -test.run=TestUDPWorkerIntegration
```

CPU를 제한하는 시험에서는 대상 CPU가 완전한 SMT 그룹임을 확인한 경우에만 `taskset`을 사용한다. 위 root 시험은 전용 로컬 시험 환경에서 수행하며 상용 환경의 권한을 승인 없이 완화하지 않는다.

남은 작업은 운영용 빌드/이미지 절차 정리, Dockerfile·Helm chart 준비, SCC/ServiceAccount 권한 확정 및 실제 상용 파드의 기동·Headless discovery·CPU 할당·RR/통신·종료 검증이다. 실제 부하 성능, 패킷 손실과 지연, transparent UDP 통신은 여전히 미검증이다. 정상 바이너리 빌드 성공은 운영 인증이나 목표 PPS 보장을 의미하지 않는다.

## 개발 내역: Makefile·Docker 이미지·UDP Helm chart (2026-10-02)

운영 준비 파일과 로컬 기능 검증을 추가했다. 상용 클러스터 배포, SCC/RBAC 변경, 레지스트리 업로드는 수행하지 않았다.

### 빌드 및 테스트

- `make test`, `make test-race`, `make vet`가 루트와 중첩 `src` Go 모듈을 모두 검사하도록 변경했다.
- `make deps`도 두 모듈의 의존성을 다운로드한다. `make help`에 주요 명령을 정리했다.
- 기존 vet 경고 3개를 동작 변경 없이 정리했다: `balance/weight.go`의 `Warn`을 `Warnf`로 수정하고,
  TCP의 `TcpContext` 초기화에 필드 이름을 명시했다. TCP 기능 확대/알고리즘 변경은 없다.
- `build-static`은 기본 Linux/amd64, CGO 비활성으로 빌드하며 `GOARCH`로 아키텍처를 지정할 수 있다.
- `make image`는 `IMAGE_REPOSITORY:IMAGE_TAG`를 로컬에 빌드하며 push하지 않는다.
  기본값은 `gobetween-local:udp-dev`다. Docker 별칭도 이 값을 사용하며 `docker-tagged`는 VERSION 태그를 사용한다.
- `make run`, `make docker-run`의 기본 설정을 신규 UDP 전용 `config/gobetween-udp.toml`로 지정했다.
  `CONFIG_FILE`로 다른 파일을 명시할 수 있다. 예시 SRV 이름은 실제 Headless Service 이름으로 바꿔야 한다.
  이 로컬 예시는 추가 BPF 권한이 필요 없는 `hash`를 명시하며 RR과 혼동하지 않는다.
- 기존 다중 OS용 `dist`/snap 절차는 이번 검증 대상이 아니다. 현재 mirror 의존성은 Windows 빌드를 지원하지 않는다.

```sh
cd /home/bigwo/nTels/debug/Poc/gobetween
make deps
make test test-race vet
make build-static NAME=gobetween-udp-linux-amd64
make image
make chart-lint chart-test
make image-test   # 로컬 Linux Docker + Helm; 기본적으로 hash 시험만 수행
```

### 이미지

Dockerfile을 Go builder + scratch 실행 이미지로 정리했다. Go builder는 실제 빌드에 사용한
`golang:1.26.5-alpine` manifest digest를 고정했으며 root/src 모듈 다운로드 캐시를 분리했다.
정적 바이너리와 CA 인증서만 최종 이미지로 복사한다. `.dockerignore`에서 Git/빌드 산출물/차트/테스트 launcher를 제외한다.
기본 USER는 `65532:0`이며 OpenShift가 임의 UID로 변경할 수 있다. 실행 파일은 읽기/실행 가능하고
로그는 stdout, pidfile은 사용하지 않는다. `/gobetween`을 exec-form ENTRYPOINT로 실행해 부모가 PID 1이 되며 SIGTERM을 처리한다.
테스트용 launcher, shell, setuid wrapper 또는 file capability는 운영 이미지에 넣지 않았다.

### Helm chart

신규 경로: `charts/gobetween`. 상세 사용법과 한계는 해당 디렉터리의 README에 정리했다.

- 기본 Pod 1개, 워커 4개, `physical`, CPU request/limit 각각 8, memory 각각 1Gi.
  조사한 SMT=2 노드에서 완전한 CPU 그룹이 할당되면 물리 코어 4개를 워커별로 분리한다.
  CPU Manager 정책은 클러스터가 제공해야 하며 memory 값은 미측정 초기 예시다.
- 서버 하나의 UDP 설정, 단방향 `max_requests=1/max_responses=0`, 기존 Headless Service의 SRV discovery를 생성한다.
  API/profiler/transparent를 설정하지 않는다. 응답 모드로 바꿀 때 idle timeout은 서버 설정에도 직접 생성한다.
- UDP Service와 내부 metrics TCP Service, ConfigMap, ServiceAccount를 제공한다.
  기존 backend Headless Service는 생성하지 않는다. 실제 서비스명/네임스페이스/named UDP port는 배포 시 지정해야 한다.
- RR + 2워커 이상에만 BPF capability를 요청하고 기본 hash/1워커 분기에는 추가하지 않는다.
  임의 UID를 위해 `runAsUser`는 지정하지 않고 root filesystem/config는 읽기 전용이다.
- 승인된 `openshift.sccName`을 입력할 때만 해당 SCC 하나의 `use` Role/RoleBinding을 생성한다.
  관리자 검토용 `admin/scc-rr-example.yaml`은 차트 템플릿 밖에 두어 자동 생성/적용하지 않는다.
- startup/liveness probe는 부모 metrics listener만 검사한다. worker/backend readiness probe는 아직 없으므로
  Pod Ready가 실제 UDP 전달 준비 완료를 뜻하지 않는다. 배포 후 worker_up/discovery/실제 통신 검사가 필요하다.
- ConfigMap checksum으로 설정 변경 시 재기동한다. 기본 Recreate 업데이트는 통신 중단과 UDP 손실 가능성이 있다.

### 실제 로컬 검증 결과

WSL Linux, Go 1.24.6, Docker 28.0.1, Helm 3.17.0 환경에서 수행했다. 이미지 바이너리는 Go 1.26.5로 빌드했다.

- `make test`, `make test-race`, `make vet`: 두 모듈 전체 통과. `-vet=off` 없이 실행했다.
  기본 시험에서 opt-in BPF/프로세스/이미지 통합 항목은 활성화하지 않은 경우 skip한다.
- `helm lint --strict` 및 차트 회귀 시험 19개 통과: TOML 디코딩, RR/hash/1워커 capability,
  metrics 제거, SRV/static, 서버 timeout, 기존 SA와 제한된 SCC 연결, 잘못된 설정 거부를 확인했다.
- 실제 이미지의 UDP 통합 시험 5개 통과: hash 단방향/응답, RR 단방향/응답, BPF 권한 없는 RR 기동 실패.
  정상 4개 시험은 각각 binary payload 120개를 전달하고, RR에서는 동일 client flow가 네 워커의 backend socket으로
  균등 분배되는 것을 확인했다. 응답 payload, 부모 metrics, SIGTERM 후 exit 0 및 강제 종료 없음도 확인했다.
- 이미지 시험은 완전한 SMT 그룹인 로컬 CPU 0–7을 제한해 physical 4워커로 실행했다.
  실제 부모 프로세스의 UID `1001290000`, NoNewPrivs=1, Seccomp=2, hash의 CapEff=0,
  RR의 CapEff=BPF만 보유하는 상태를 `/proc`에서 확인했다. 이를 상용 노드의 실제 할당으로 해석하지 않는다.

```sh
# 선택 CPU가 완전한 SMT 그룹이고 전용 로컬 시험 환경인 경우에만 실행:
GOBETWEEN_DOCKER_BPF=1 GOBETWEEN_DOCKER_CPUSET=0-7 make image-test
```

### 중요한 잔여 사항: 비루트 RR 권한 전달

로컬 Docker에서 임의 UID와 `--cap-add=BPF`만 지정했을 때 CapBnd에만 BPF가 남고
CapEff/CapPrm/CapAmb는 0이었다. 따라서 RR 성공 시험에는 **테스트 전용 ambient 권한 전달 프로그램**을 사용했다.
이 프로그램은 로컬 시험 컨테이너에서 일시적으로 SETUID/SETGID를 받아 UID를 변경한 뒤 유효 권한을 BPF만 남기고
실제 `/gobetween`으로 exec한다. 이 추가 capability들과 launcher는 차트/운영 이미지에는 들어가지 않는다.

[upstream CRI-O v1.35.0 코드](https://github.com/cri-o/cri-o/blob/v1.35.0/internal/factory/container/container.go)에서도
ambient capability를 제거하는 처리를 확인했다. 상용 Red Hat 빌드의 정확한 CRI-O 버전·설정·exec 시 권한은 아직 미확인이다.
**SCC가 CAP_BPF를 허용한다는 사실만으로 임의 UID RR의 기동을 보장하면 안 된다.**
현재 차트의 RR 기본값은 비루트 capability 전달과 seccomp/SELinux를 확정해야 하는 배포 초안이다.
실제 CapEff가 확보되지 않으면 권한 있는 BPF loader 분리나 승인된 제한적 기동 방식 등에 대한 후속 설계가 필요하다.
운영용 권한 상승 wrapper, privileged 실행, 자동 SYS_ADMIN 추가 또는 노드 BPF 보안 정책 완화는 구현하지 않았다.

남은 작업은 실제 레지스트리/네임스페이스/백엔드 SRV/대상 노드 값 확정, 비루트 RR 권한 전달 방식과 SCC/ServiceAccount 승인,
상용 Pod의 기동·Headless discovery·CPU 전용 할당·RR/통신·종료 확인 및 부하 측정이다.
로컬 Docker 테스트 성공은 OpenShift 운영 인증, 목표 30만 PPS 또는 무손실/무지연 보장을 의미하지 않는다.

## OpenShift root·privileged SCC 프로파일 추가 (2026-10-02)

앞선 비루트 RR 권한 전달 문제에 대한 배포 선택지로 root·privileged 모드를 차트에 추가했다.
기본 비루트 구성은 유지하며, `charts/gobetween/values-openshift-privileged.yaml`을 명시적으로 적용한다.
차트 버전은 `0.2.0`이다. Go 애플리케이션 코드와 운영 이미지는 변경하지 않았다.

- 신규 설정: `openshift.privileged`, `openshift.createSCC`, 기존 `openshift.sccName`.
- 전용 SCC 자동 생성: UID 0 필수, privileged 허용, 초기 user namespace 허용, unconfined seccomp.
  SCC 이름에는 네임스페이스와 릴리스명을 포함하고 긴 이름은 해시로 축약한다.
- 해당 네임스페이스의 ServiceAccount 하나에 해당 SCC 하나의 `use`만 Role/RoleBinding으로 부여한다.
  SCC의 users/groups에 직접 전역 권한을 넣거나 ClusterRoleBinding을 생성하지 않는다.
- Pod/컨테이너 UID 0, `runAsNonRoot=false`, 컨테이너 `privileged=true`,
  `allowPrivilegeEscalation=true`를 일관되게 적용하고 RuntimeDefault seccomp 설정을 제거한다.
  privileged에서는 capability drop/add 블록을 생략한다. BPF 전용 권한보다 훨씬 넓은 권한이다.
- 읽기 전용 root filesystem/config와 ServiceAccount 토큰 미마운트는 유지한다.
  host network/PID/IPC/ports/path는 추가하지 않는다.
- `createSCC=false`와 승인된 `sccName`으로 기존 SCC 재사용도 가능하다.
  이 경우 SCC 자체는 변경하지 않는다. 기본 제공 SCC를 새로 생성/덮어쓰는 설정은 거부한다.
- 생성 SCC는 Helm 릴리스 소유이며 uninstall 시 삭제 대상이다. 다른 릴리스/워크로드에 공유하지 않는다.
  실제 설치자는 SCC 생성과 SCC 사용 권한 연결이 가능한 관리자 승인 권한이 필요하다.

```sh
# 로컬 검증/렌더링. 실제 클러스터 적용은 수행하지 않았다.
make chart-lint chart-test
helm template udp-proxy charts/gobetween -n target-namespace \
  -f charts/gobetween/values-openshift-privileged.yaml -f values-prod.yaml
```

WSL Helm 3.17.0에서 기본/privileged 두 프로파일의 strict lint와 차트 회귀 시험 28개가 통과했다.
전용 SCC/SA 권한 범위, UID 0/privileged 설정, 기존 SCC 재사용, 긴 이름/네임스페이스별 이름 분리,
기본 설정 보존 및 잘못된 권한 조합 거부를 검증했다.
root/src 두 Go 모듈 전체의 `make test-race vet`도 통과했고,
로컬 차트 패키지 `bin/gobetween-0.2.0.tgz`를 생성했다.
이는 로컬 템플릿 검증이며 실제 OpenShift admission, SCC 선택, CPU 전용 할당, RR 통신 및 부하 검증은 남아 있다.
이 작업에서 상용 클러스터 적용, SCC/RBAC 변경, 이미지 push는 수행하지 않았다.
