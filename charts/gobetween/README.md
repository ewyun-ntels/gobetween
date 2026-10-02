# gobetween UDP Helm chart

상태: 로컬 렌더링·설정 검증 완료. 상용 OpenShift의 admission, CPU 전용 할당,
비루트 BPF capability 전달, seccomp/SELinux 및 실제 Headless discovery는 아직 미검증이다.
root·privileged 실행은 `values-openshift-privileged.yaml`로 명시적으로 선택할 수 있다.
기본 비루트 설정은 유지하며, 어느 모드든 실제 OpenShift 기동/통신 확인은 별도로 필요하다.

## 기본 구성

- Pod 1개 안에서 부모 1개와 UDP 워커 4개 실행. `replicaCount`는 Pod 수이며 워커 수가 아니다.
- `worker_cpu_policy = "physical"`, CPU request/limit 각각 `8`, memory 각각 `1Gi`.
- 단방향: `max_requests = 1`, `max_responses = 0`.
- UDP Service와 내부 메트릭 TCP Service를 분리. REST API·profiler·transparent 모드는 설정하지 않음.
- 기존 Headless Service의 named UDP port를 SRV로 조회. 백엔드 Service/Pod는 이 차트가 만들지 않음.
- 임의 UID, 읽기 전용 root filesystem/config, stdout 로그, pidfile 없음.
- RR + 워커 2개 이상만 `BPF` 요청. hash 및 RR 1워커는 추가 capability 없이 실행.
- ServiceAccount 토큰은 마운트하지 않음. 승인된 `openshift.sccName`을 지정할 때만
  (또는 `openshift.createSCC: true`로 전용 SCC를 만들 때) 그 SCC 하나의 `use` 권한을
  해당 네임스페이스 SA에 연결하는 Role/RoleBinding 생성.
- 설정 변경은 checksum으로 Pod 재기동. 기본 Recreate 업데이트에는 통신 중단이 있으며
  진행 중인 UDP 패킷/세션 보존을 보장하지 않음.

조사한 상용 노드는 SMT=2이므로 CPU 8개가 완전한 SMT 그룹으로 할당되면 물리 코어 4개다.
정수 CPU의 Guaranteed Pod를 해당 `static/full-pcpus-only` 노드에 배치해야 한다.
`resources.cpu`는 논리 CPU 개수이며 이 차트가 CPU Manager 정책 자체를 설정하지는 않는다.
메모리 `1Gi`는 초기 예시이고 부하에 맞춘 값이 아니다. 사이드카를 나중에 추가한다면
모든 컨테이너의 CPU·메모리 request/limit 및 CPU Manager 참여 조건을 다시 확인한다.
physical 모드에서 불완전한 SMT 그룹 또는 부족한 물리 코어는 기동 오류이며 자동 fallback은 없다.

## 배포 전 값 준비

별도 `values-prod.yaml`에 아래와 같이 실제 값을 기입한다. 레지스트리·네임스페이스·노드
이름은 예시이므로 실제 환경 값으로 교체해야 한다. 이 문서의 명령은 이번 개발에서 배포를 실행한 기록이 아니다.

```yaml
image:
  repository: registry.example.invalid/team/gobetween
  tag: udp-dev
discovery:
  serviceName: udp-backend-headless
  namespace: backend-namespace
  portName: backend
nodeSelector:
  kubernetes.io/os: linux
  kubernetes.io/hostname: validated-worker-node
```

이 설정은 `_backend._udp.udp-backend-headless.backend-namespace.svc.cluster.local.`을 조회한다.
기존 Headless Service는 `clusterIP: None` 및 `name: backend`, `protocol: UDP` 포트를 가져야 한다.
포트 번호는 SRV 응답에서 얻는다. DNS는 기본적으로 Pod의 `/etc/resolv.conf`를 사용한다.
다른 SRV 이름을 쓸 경우 `discovery.srvLookupPattern`으로 전체 이름을 지정할 수 있다.

```sh
# 저장소 루트에서 로컬 검증만 수행 (클러스터 연결 불필요)
make chart-lint chart-test
helm template udp-proxy charts/gobetween -n target-namespace -f values-prod.yaml
```

`service.type`은 기본 ClusterIP다. 외부 UDP 진입점은 CNI/LoadBalancer/NodePort 정책에 맞게 별도로 확정한다.
metrics Service는 항상 내부 ClusterIP이며 인증 기능이 없으므로 접근 제어를 고려한다.
ServiceMonitor/NetworkPolicy는 자동 생성하지 않는다. hash 권한 최소 시험은
`--set udp.reusePortDistribution=hash`로 명시적으로 선택한다. RR 오류를 hash로 자동 전환하지 않는다.

## RR 권한: 반드시 확인할 사항

### root·privileged 모드와 SCC 사용 권한

승인된 root·privileged 실행을 사용할 경우 아래 파일을 추가한다.

```sh
# 로컬 렌더링만 수행. 실제 배포 명령이 아님.
helm template udp-proxy charts/gobetween -n target-namespace \
  -f charts/gobetween/values-openshift-privileged.yaml -f values-prod.yaml
```

이 프로파일의 설정은 다음과 같다.

```yaml
openshift:
  privileged: true
  createSCC: true
  sccName: ""  # 비우면 네임스페이스·릴리스명을 포함하여 자동 생성
hostUsers: true
```

- Pod/컨테이너의 `runAsUser: 0`, `runAsNonRoot: false`, 컨테이너의 `privileged: true`,
  `allowPrivilegeEscalation: true`를 함께 적용하고 기존 seccompProfile을 제거한다.
- RR 부모와 자식은 별도 launcher 없이 운영 이미지의 `/gobetween`으로 실행한다.
  비루트 capability 전달에 의존하지 않으며, privileged는 BPF만이 아니라 모든 capability를 부여한다.
- root filesystem/config의 읽기 전용 설정과 토큰 미마운트는 유지한다.
  hostNetwork/hostPID/hostIPC/hostPort/hostPath는 활성화하지 않는다.
- 전용 SCC는 UID 0과 privileged를 허용하고 `userNamespaceLevel: AllowHostLevel`,
  `seccompProfiles: [unconfined]`를 사용한다. users/groups 전체에 직접 권한을 주지 않는다.
- Role/RoleBinding은 이 네임스페이스의 해당 ServiceAccount 하나에 해당 SCC 하나의 `use`만 연결한다.
  ClusterRole/ClusterRoleBinding 또는 공용 SCC 변경은 없다.
- 기본 SCC 이름은 `<namespace>-<release>-gobetween-privileged`이며 긴 이름은 해시로 축약한다.
  SCC는 클러스터 범위 리소스이므로 다른 릴리스와 명시적 `sccName`을 공유하지 않는다.

SCC를 생성하고 그 사용 권한을 연결할 수 있는 **관리자 승인 계정**으로 설치해야 한다.
차트가 설치자 자신의 권한을 올리지는 않는다. 이 프로파일이 생성한 SCC는 Helm 릴리스 소유이며
uninstall 시 삭제 대상이다. 다른 워크로드에 공유하지 않는다.

이미 승인된 root·privileged SCC를 재사용하려면 `values-prod.yaml`에서 아래처럼 덮어쓴다.

```yaml
openshift:
  privileged: true
  createSCC: false
  sccName: approved-existing-privileged-scc
```

이 경우 SCC 자체는 생성/수정/삭제하지 않고 SA의 사용 권한만 연결한다.
기존 `privileged` SCC를 참조하는 것도 가능하지만 기본 제공 SCC 이름으로 새 SCC를 만들거나
덮어쓰려 하면 렌더링을 거부한다. 선택한 기존 SCC가 실제 UID 0/privileged/hostUsers 등을
허용하는지는 관리자가 확인해야 한다. 별도 PSA/Admission 정책에 의한 거부도 차트가 우회하지 않는다.

privileged는 seccomp 등 격리 제약을 크게 완화하므로 이 모드는 운영 보안 승인이 전제다.
[Kubernetes privileged 설명](https://kubernetes.io/docs/concepts/security/linux-kernel-security-constraints/#privileged-containers).
실제 OpenShift admission/SCC 선택, CPU 전용 할당, Headless discovery, RR 통신, 종료 및 부하는 아직 미검증이다.

### 기본 비루트 RR 모드의 제한

`admin/scc-rr-example.yaml`은 관리자 검토용 초안이며 Helm이 자동 적용하지 않는다.
기본 설정에서는 restricted SCC를 수정하거나 privileged/SYS_ADMIN을 자동 부여하지 않는다.
SCC 생성과 `use` RoleBinding 생성 권한은 실제 실행 계정에 따라 별도 승인이 필요하다.

RR 부모는 초기 user namespace의 BPF 권한이 필요하므로 `hostUsers: true`를 사용한다.
이는 host network/PID 사용과는 별개다. `hostUsers: false`를 요구하는 restricted-v3와 RR 기본값은 맞지 않는다.
SCC의 `allowedCapabilities: [BPF]`와 Pod의 `capabilities.add`는 권한 허용/요청일 뿐,
실제 `/gobetween` 프로세스의 유효 capability를 증명하지 않는다.

로컬 Docker 28.0.1에서는 비루트 UID + `--cap-add=BPF`만으로는 CapBnd만 설정되고
CapEff/CapPrm/CapAmb는 0이었다. RR 시험에는 **테스트 전용** ambient handoff를 사용했다.
운영 이미지에는 launcher, setuid wrapper, file capability를 추가하지 않았다.
OpenShift에서 동일한 방식으로 동작한다고 가정하지 말고 CRI-O 버전/설정과 실제 프로세스의
Uid, CapEff, CapPrm, CapAmb, NoNewPrivs, Seccomp를 확인해야 한다.
비루트 프로세스에 BPF 권한이 전달되지 않으면 배포 권한/기동 방식에 대한 추가 설계와 승인이 필요하다.

[upstream CRI-O v1.35.0의 capability 설정 코드](https://github.com/cri-o/cri-o/blob/v1.35.0/internal/factory/container/container.go)에서도
ambient capability를 제거하는 처리를 확인했다. 상용 Red Hat 빌드의 정확한 버전·동작은 아직 확인하지 않았으므로,
이 차트의 비루트 RR 구성은 **SCC 허용만으로 동작을 보장할 수 없는 배포 초안**으로 취급한다.
필요하다면 권한 있는 별도 BPF loader 또는 승인된 제한적 기동 방식 등을 후속 설계해야 한다.
이번 작업에서 운영용 권한 상승 wrapper를 추가하지 않았다.

비루트 모드의 seccomp는 RuntimeDefault를 기본으로 사용한다. 대상 CRI-O 기본 프로파일에서 필요한 BPF syscall이
차단될 경우 관리자 승인된 Localhost 프로파일 등의 대안을 검토한다. 자동 Unconfined 변경은 하지 않는다.
실제 memlock 오류가 발생하면 커널의 memcg BPF accounting과 RLIMIT_MEMLOCK을 먼저 확인한다.
SYS_RESOURCE를 무조건 추가하지 않는다. 상용 SELinux 정책도 별도로 확인한다.

참고: [OpenShift SCC 관리](https://docs.redhat.com/en/documentation/openshift_container_platform/4.22/html/authentication_and_authorization/managing-pod-security-policies),
[SCC userNamespaceLevel API](https://docs.redhat.com/en/documentation/openshift_container_platform/4.22/html/security_apis/securitycontextconstraints-security-openshift-io-v1).

## 응답 모드와 메트릭 probe

응답이 필요하면 `udp.maxRequests: 0`, `udp.maxResponses: 1` 등 실제 프로토콜에 맞게 변경한다.
무제한 응답(`0`)을 쓸 경우 idle timeout을 유한하게 설정해 세션이 끝나는 조건을 정의한다.
`max_responses = 0` 자체는 응답 비활성화가 아니라 응답 개수 제한 없음이다.

startup/liveness probe는 부모의 metrics TCP listener만 확인하며 workers/backends 상태를 검사하지 않는다.
readiness probe는 아직 제공하지 않는다. Pod Ready나 probe 성공을 전달 준비 완료로 해석하면 안 된다.
`gobetween_worker_up`, discovery backend 상태, 실제 UDP 통신을 별도로 확인해야 한다.
metrics를 끄면 관련 Service/port/probe가 함께 제거된다.

## 이미지 기능 테스트

저장소 루트에서 WSL의 로컬 Linux Docker 엔진과 Helm을 사용한다.
테스트는 로컬 host network의 임시 포트/백엔드를 사용하며 시험 컨테이너를 종료·제거한다.
원격 Docker daemon이나 다른 network namespace 환경에는 그대로 사용할 수 없다.

```sh
make image
make image-test                         # hash 단방향/응답, RR 시험은 skip
# 선택 CPU가 완전한 SMT 그룹인 전용 로컬 시험 환경에서만:
GOBETWEEN_DOCKER_BPF=1 GOBETWEEN_DOCKER_CPUSET=0-7 make image-test
```

마지막 명령은 테스트 launcher의 UID 변경을 위해 일시적으로 SETUID/SETGID를 요청한다.
launcher가 UID 변경 후 해당 유효 권한들을 버리고 CAP_BPF만 남겨 실제 gobetween으로 exec한다.
이들은 차트/운영 이미지에 필요한 capability 목록이 아니다. 프로세스가 UID `1001290000`,
CapEff=BPF만 보유, NoNewPrivs=1, Seccomp=2인지 테스트에서 직접 확인한다.
로컬 CPU 번호 `0-7`은 상용 CPU 번호를 의미하지 않는다.
