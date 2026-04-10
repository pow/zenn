---
title: "TypeScript で async 関数を指数バックオフ+ジッター付きでリトライする汎用ヘルパー"
emoji: "🔁"
type: "tech"
topics: ["typescript", "nodejs", "tips"]
published: false
---

## はじめに

Next.js のサーバーサイドで外部 API（認証サービスや GraphQL ゲートウェイ）を叩いていると、起動直後や瞬間的な 5xx で稀にリクエストが失敗します。ユーザーには気づかれないうちに 1〜2 回リトライすれば復旧することがほとんどなのに、そのたびにエラーページを見せるのはもったいないです。

実プロジェクトで Logto と Apollo Client の両方にリトライを入れたところ、同じ設計判断を何度もすることになったので、今回は依存ゼロで書ける**指数バックオフ + ジッター + retryIf** 付きの汎用 retry 関数をまとめます。コードは Node.js 22 の組み込みテストランナー(*2)で検証しているので、そのままコピペで動きます。

## ナイーブな固定遅延リトライの落とし穴

まず「固定遅延で N 回試すだけ」の素朴な実装がなぜ弱いかを押さえます。

```ts
// ❌ 固定遅延の素朴な実装
const sleep = (ms: number) => new Promise(r => setTimeout(r, ms))

async function retryFixed<T>(fn: () => Promise<T>, max = 3, delay = 500) {
  let lastError: unknown
  for (let i = 0; i < max; i++) {
    try {
      return await fn()
    } catch (e) {
      lastError = e
      await sleep(delay)
    }
  }
  throw lastError
}
```

問題は2つあります。

1. **Thundering Herd**: 大規模障害が復旧した瞬間、クライアント全員が同じタイミングで再送信してサーバーを再び潰します(*1)。
2. **回復可能なエラーと不可能なエラーが区別できない**: 400 Bad Request でもリトライしてしまうと、無駄な待機時間が入り UX が悪化します。

この2つを解決するのが **ジッター付き指数バックオフ** と **retryIf** 述語です。

## retry 関数の実装

設計方針は以下の通りです。

- `maxAttempts` は**初回の呼び出しを含めた**総試行回数にする（「2回までリトライ」と「3回まで試す」の混乱を避ける）
- 遅延は `initialDelayMs * 2^(attempt-1)` を `maxDelayMs` で頭打ちにする
- `jitter: true` のときは `[0, 計算した遅延)` の一様乱数を使う（AWS の "Full Jitter" 戦略)(*1)
- `retryIf` はエラーを受けて `boolean` を返す述語。`false` なら即座に throw
- `sleep` と `random` は差し替え可能にして、テストで時間と乱数を固定できるようにする

```ts
export interface RetryOptions {
  /** 最大試行回数（初回含む）。デフォルト 3 */
  maxAttempts?: number
  /** 初回の遅延（ミリ秒）。デフォルト 300 */
  initialDelayMs?: number
  /** 遅延の上限（ミリ秒）。デフォルト 3000 */
  maxDelayMs?: number
  /** ジッターを有効化するか。デフォルト true */
  jitter?: boolean
  /** リトライすべきかを判定する関数。デフォルト: すべてリトライ */
  retryIf?: (error: unknown) => boolean
  /** sleep 実装。テスト時に差し替えるためのフック */
  sleep?: (ms: number) => Promise<void>
  /** 乱数生成器。テスト時に差し替えるためのフック */
  random?: () => number
}

const defaultSleep = (ms: number): Promise<void> =>
  new Promise(resolve => setTimeout(resolve, ms))

export async function retry<T>(
  fn: () => Promise<T>,
  options: RetryOptions = {},
): Promise<T> {
  const {
    maxAttempts = 3,
    initialDelayMs = 300,
    maxDelayMs = 3000,
    jitter = true,
    retryIf = () => true,
    sleep = defaultSleep,
    random = Math.random,
  } = options

  let lastError: unknown
  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      return await fn()
    } catch (error) {
      lastError = error
      const isLastAttempt = attempt >= maxAttempts
      if (isLastAttempt || !retryIf(error)) {
        throw error
      }
      const exp = Math.min(initialDelayMs * 2 ** (attempt - 1), maxDelayMs)
      const delay = jitter ? Math.floor(random() * exp) : exp
      await sleep(delay)
    }
  }
  throw lastError
}
```

### ポイント1: 遅延計算の順序

`initialDelayMs * 2 ** (attempt - 1)` を**先に** `maxDelayMs` でクランプしてからジッターをかけています。順序を逆にして「ジッターの結果を maxDelayMs でクランプ」すると、jitter=true のときに遅延の分布が歪みます（頭が削られて実質の平均遅延が下がる）。計算順序は地味に大事です。

### ポイント2: retryIf の返り値を素直に boolean にする

リトライを条件分岐させたくなると、つい `retryIf: (err) => err.status >= 500 ? true : false` のように書きがちですが、**「エラーを受けてそれが一時的か」** というドメインロジックを外に切り出す方が再利用しやすくなります。

```ts
// 5xx と「ネットワークエラー」だけリトライする例
const isTransient = (err: unknown): boolean => {
  if (err instanceof TypeError) return true // fetch failed
  if (err instanceof Response) return err.status >= 500
  return false
}

await retry(() => fetchUserProfile(userId), { retryIf: isTransient })
```

## テストで挙動を固定する

`sleep` と `random` を注入できるようにしたので、テストでは時間を止めて決定的に検証できます。

```ts
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { retry } from '../src/retry.ts'

test('exponential backoff grows and is capped by maxDelayMs (no jitter)', async () => {
  const sleepCalls: number[] = []
  await assert.rejects(
    retry(
      async () => {
        throw new Error('always')
      },
      {
        maxAttempts: 5,
        initialDelayMs: 100,
        maxDelayMs: 500,
        jitter: false,
        sleep: async ms => {
          sleepCalls.push(ms)
        },
      },
    ),
  )
  // 100 → 200 → 400 → 500(capped) の順で遅延が積まれる
  assert.deepEqual(sleepCalls, [100, 200, 400, 500])
})

test('retryIf=false stops retrying immediately', async () => {
  let calls = 0
  await assert.rejects(
    retry(
      async () => {
        calls += 1
        throw new Error('validation error')
      },
      {
        maxAttempts: 5,
        retryIf: err =>
          !(err instanceof Error && err.message.includes('validation')),
        sleep: async () => {},
      },
    ),
  )
  // バリデーションエラーはリトライしない = 1回で終わる
  assert.equal(calls, 1)
})
```

`node --import tsx --test tests/retry.test.ts` で実行すると TAP 出力が得られます。CI でそのまま使えるので、Jest や Vitest を入れたくない小さなパッケージに向いています(*2)。

## 既存エコシステムに任せる選択肢

もし Apollo Client を使っているなら、自前で書く前に `RetryLink` を見てみるのがおすすめです。`delay` と `attempts` を関数で渡せて、指数バックオフ・ジッター・カスタム述語が用途を満たすならそちらが早いです(*3)。

```ts
import { RetryLink } from '@apollo/client/link/retry'

const retryLink = new RetryLink({
  delay: { initial: 300, max: 3000, jitter: true },
  attempts: {
    max: 3,
    retryIf: (error, _operation) => !!error,
  },
})
```

ただし、GraphQL 以外の素の `fetch` や、Server Actions から認証 SDK を叩くケース（`RetryLink` の層を通らない場所）では、やはり上のような汎用ヘルパーが1つあると便利です。プロジェクトの中で**同じパターンで書けること**が、レビューのときの認知負荷を下げてくれます。

## まとめ

- 固定遅延のリトライは Thundering Herd を招く。指数バックオフ + ジッターで分散させる(*1)
- `maxAttempts` は「初回を含む総試行回数」にして命名の混乱を避ける
- `retryIf` を渡せるようにして、400系などの非一時的エラーで無駄に待たない
- `sleep` と `random` を差し替えられるようにしておくと、Node.js 組み込みテストランナー(*2)だけで決定的にテストできる
- Apollo Client を使っているなら `RetryLink`(*3) に任せるのが先。素の `fetch` や SDK 呼び出しにだけ自前ヘルパーを使う

小さな関数ですが、どのプロジェクトでも1回は書き直すことになるので、テストまでセットにしておくと後から安心して触れます。

## 参考リンク

- *1: [Exponential Backoff And Jitter | AWS Architecture Blog](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/)
- *2: [Test runner | Node.js Documentation](https://nodejs.org/api/test.html)
- *3: [RetryLink | Apollo GraphQL Docs](https://www.apollographql.com/docs/react/api/link/apollo-link-retry)
