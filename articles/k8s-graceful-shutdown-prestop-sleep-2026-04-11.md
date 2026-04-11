---
title: "K8s ローリングアップデートで 502 を出さない Go + preStop sleep パターン"
emoji: "🛑"
type: "tech"
topics: ["kubernetes", "go", "helm", "tips"]
published: false
---

## はじめに

Kubernetes で Go の HTTP サービスを運用していると、**ローリングアップデート中にクライアント側で散発的な 502 / Connection refused が出る**という現象に遭遇することがあります。アプリ側では `http.Server.Shutdown` をちゃんと呼んでいて、`terminationGracePeriodSeconds` も十分取っているのに、なぜか落ちる。

先日、自分のプロジェクトでも preStop の sleep を 10 秒から 30 秒に伸ばしたところ rollout 時の 5xx がほぼ消えたので、その理由と仕組み、そして Go 側の「正しい落とし方」をテスト付きでまとめます。

## なぜ停止中の Pod にトラフィックが届くのか

Pod を削除すると、Kubernetes は並行していくつかの作業を始めます(*1)。

1. kubelet が preStop フックを起動し、その後コンテナに SIGTERM を送る
2. Endpoints / EndpointSlice コントローラが、対象 Pod を Service のエンドポイントから外す
3. 各ノードの kube-proxy が EndpointSlice の変更を検知し、ローカルの iptables（または IPVS）ルールを書き換える

問題は **2 と 3 が非同期**だという点です。Pod が Terminating になった瞬間にエンドポイントから論理的には外れますが、各ノードの iptables に反映されるまでには数秒〜数十秒のラグがあります。このラグの間、他のノードから見るとその Pod はまだ宛先の候補で、新規接続が届きます。

Mercari の Production Readiness Checklist にも「Pod のローリング更新中、旧 Pod が削除されている最中でも新しい接続が張られることがある。これは Kubernetes の仕様上の動作だ」と明記されています(*4)。つまり、アプリが SIGTERM を受け取った直後にリスナーを閉じると、まだルーティング先として生きている状態で接続を拒否してしまい、クライアントには 502 / connection refused として観測されます。

## 解決策: preStop sleep で「外される時間」を稼ぐ

K8s のコンテナライフサイクルフック `preStop` は **SIGTERM より前に** 同期実行されます。ハンドラが終わるまで SIGTERM は送られません(*2)。つまり、`preStop` にただの sleep を置けば、その間アプリは普通に 200 を返し続け、その裏で iptables の伝播が進みます。

最もシンプルな Helm/Deployment 例です（`samples/.../helm-snippet.yaml` に収録）。

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      # preStop sleep + Shutdown が両方収まる時間を確保する
      terminationGracePeriodSeconds: 60
      containers:
        - name: app
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8080
            periodSeconds: 2
            failureThreshold: 1
          lifecycle:
            preStop:
              exec:
                # 30 秒は経験則。kube-proxy の伝播が
                # 収まる時間を計測して決めるのがベスト
                command: ["/bin/sh", "-c", "sleep 30"]
```

ポイントは 3 つです。

- `preStopSleep` は **kube-proxy の反映時間より長く** 取る。クラスタの規模によって適正値が変わるので、rollout 時の 5xx を観測しながら調整します。小さすぎると 502、大きすぎると rollout が遅くなる、というトレードオフです。
- `terminationGracePeriodSeconds` は `preStop sleep + アプリのドレイン時間` を収められる値にする。これを超えると kubelet が SIGKILL を送り、in-flight が強制切断されます(*1)。
- `maxUnavailable: 0` にすると、旧 Pod が死に切る前に新 Pod が Ready になるので、合計容量がトータルで割れません。

## Go 側: signal.NotifyContext + http.Server.Shutdown

preStop で時間を稼いでも、アプリが SIGTERM を受けた瞬間に `os.Exit` してしまったら台無しです。標準ライブラリの `http.Server.Shutdown` は、**新規接続を受け付けず、in-flight リクエストを待ってから** サーバを停止するメソッドです。`ctx` が切れるまで in-flight の完了を待ち、待ちきれなかった場合のみエラーを返します(*3)。

サンプル（`samples/.../server.go`）から抜粋します。

```go
// handleReadyz: Shutdown が呼ばれるまで 200、以降は 503。
// K8s の readinessProbe に返すことで Service のエンドポイントからも早く外れる。
func (s *Server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
    if s.ready.Load() {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
        return
    }
    w.WriteHeader(http.StatusServiceUnavailable)
    _, _ = w.Write([]byte("shutting down"))
}

// Shutdown: readiness を先に落とし、その後 http.Server.Shutdown に委譲する。
func (s *Server) Shutdown(ctx context.Context) error {
    s.ready.Store(false)
    return s.httpSrv.Shutdown(ctx)
}
```

`atomic.Bool` は好みですが、複数ゴルーチンから読まれる「終了開始フラグ」なので `sync/atomic` か `sync.RWMutex` を使ってください(*3)。

エントリポイント側は `signal.NotifyContext` で SIGTERM/SIGINT を拾い、Shutdown に渡す ctx をタイムアウト付きで作るのが定石です。

```go
func main() {
    srv := graceful.NewServer(":8080", 0)

    // SIGTERM / SIGINT で ctx がキャンセルされる
    ctx, stop := signal.NotifyContext(context.Background(),
        syscall.SIGTERM, syscall.SIGINT)
    defer stop()

    go func() {
        if err := srv.ListenAndServe(); err != nil {
            log.Fatalf("serve: %v", err)
        }
    }()

    <-ctx.Done()

    // terminationGracePeriodSeconds より少し短い値を使う。
    // SIGKILL が飛んでくる前に必ず Shutdown が返るようにする。
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
    defer cancel()

    if err := srv.Shutdown(shutdownCtx); err != nil {
        log.Printf("graceful shutdown: %v", err)
    }
}
```

ここで **Shutdown のタイムアウトは `terminationGracePeriodSeconds - preStopSleep` より短く** します。例えば `terminationGracePeriodSeconds: 60`、`preStopSleep: 30` なら、SIGTERM 到着後に使える時間は約 30 秒。Shutdown には 25 秒ほど渡しておくと余裕があります。

## テストで挙動を固定する

このパターンの要点は「Shutdown 中も in-flight は 200 を返し切ること」「readiness は即座に 503 を返すこと」の 2 点です。サンプルの `server_test.go` では以下を検証しています（すべて標準ライブラリのみ・外部依存なしでパスします）。

```go
// Shutdown 中に飛んできた /readyz は 503 を返す
func TestReadyzFlipsTo503OnShutdown(t *testing.T) {
    s := NewServer(":0", 0)
    ts := httptest.NewServer(s.Handler())
    defer ts.Close()

    if err := s.Shutdown(context.Background()); err != nil {
        t.Fatalf("shutdown: %v", err)
    }

    resp, _ := http.Get(ts.URL + "/readyz")
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusServiceUnavailable {
        t.Fatalf("expected 503 after shutdown, got %d", resp.StatusCode)
    }
}
```

もうひとつ重要なのが、Shutdown 開始後の in-flight が切られないことの検証です。

```go
// 遅いリクエストを投げ、ハンドラが sleep 中に Shutdown を呼んで
// リクエストが 200 で完了することを確認する
func TestShutdownWaitsForInFlightRequest(t *testing.T) {
    s := NewServer("127.0.0.1:0", 300*time.Millisecond)
    ln, _ := net.Listen("tcp", "127.0.0.1:0")
    go s.httpSrv.Serve(ln)

    url := "http://" + ln.Addr().String() + "/"
    // ... (別ゴルーチンで http.Get)
    time.Sleep(50 * time.Millisecond) // ハンドラが sleep に入るのを待つ

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    _ = s.Shutdown(ctx) // ← ここが core: in-flight を切ってはいけない

    // 結果: statusCode == 200, body == "done"
}
```

`go test ./...` の結果:

```
$ go test ./... -v
=== RUN   TestReadyzFlipsTo503OnShutdown
--- PASS: TestReadyzFlipsTo503OnShutdown (0.00s)
=== RUN   TestShutdownWaitsForInFlightRequest
--- PASS: TestShutdownWaitsForInFlightRequest (0.32s)
=== RUN   TestShutdownRejectsNewConnections
--- PASS: TestShutdownRejectsNewConnections (0.00s)
=== RUN   TestIsReadyReflectsShutdownState
--- PASS: TestIsReadyReflectsShutdownState (0.00s)
PASS
```

CI にこの種のテストを入れておくと、誰かが `srv.Close()` に書き換えたり Shutdown を非同期化したりといったリグレッションを検知できます。

## まとめ

- ローリング更新中の 502 の主因は **Pod が Terminating になってから各ノードの iptables が更新されるまでのタイムラグ**(*1,*4)
- `preStop` に sleep を仕込むだけで、その間アプリは普通に 200 を返し続ける（SIGTERM より前に同期実行される)(*2)
- Go 側は `signal.NotifyContext` で SIGTERM を受け、`http.Server.Shutdown(ctx)` で in-flight を待ってから止める(*3)
- `terminationGracePeriodSeconds` = `preStopSleep` + `Shutdown タイムアウト` + 余裕、で設計する
- readiness プローブも Shutdown 開始で 503 に倒しておくと二重に安全

「preStop sleep」は一見泥臭いハックに見えますが、K8s の非同期な endpoint 伝播に対する現時点で標準的な解法です。rollout で 5xx が出ている場合、最初に疑うポイントとして覚えておくと便利です。

## 参考リンク

- *1: [Kubernetes Docs — Pod Lifecycle (Termination of Pods)](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/)
- *2: [Kubernetes Docs — Container Lifecycle Hooks](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/)
- *3: [Go 標準ライブラリ — net/http Server.Shutdown](https://pkg.go.dev/net/http#Server.Shutdown)
- *4: [Mercari Production Readiness Checklist — PreStop Hook](https://github.com/mercari/production-readiness-checklist/blob/master/docs/concepts/container-lifecycle-hooks-pre-stop.md)
