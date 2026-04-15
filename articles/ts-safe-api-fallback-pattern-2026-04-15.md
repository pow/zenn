---
title: "API フォールバックの罠 — stale データで無限エラーループを防ぐ設計"
emoji: "🛑"
type: "tech"
topics: ["typescript", "nextjs", "tips"]
published: false
---

## はじめに

Next.js の Server Component から外部 API を呼ぶとき、API が一時的に失敗したらどうしますか？「キャッシュやセッションに残っている前回のデータをフォールバックとして返せばいい」と考えるのは自然です。

しかし、この素朴なフォールバックには罠があります。stale（古くなった）データの中に**次の API 呼び出しに使われるコンテキスト情報**が含まれていると、そのコンテキストがすでに無効化されている場合、API 呼び出しが繰り返し失敗する**無限エラーループ**に陥ります。

本記事では、この問題の原因と、TypeScript の Discriminated Union(*1) を使った安全なフォールバック設計を紹介します。

## なぜ stale フォールバックが無限ループを起こすのか

典型的な例として、マルチテナントの SaaS アプリを考えます。ユーザーがログインすると組織コンテキスト（組織名、内部 ID など）を取得し、以降の API 呼び出しでその組織 ID を使います。

素朴なフォールバック実装はこうなります。

```ts
type OrgContext = {
  orgId: string
  orgName: string    // サイドバー表示用
  orgInterId: number // API リクエストに使う内部 ID
}

async function naiveResolveContext(
  fetchOrg: () => Promise<OrgContext>,
  staleContext: OrgContext | null,
): Promise<OrgContext | null> {
  try {
    return await fetchOrg()
  } catch {
    return staleContext // ❌ stale な orgInterId も返してしまう
  }
}
```

ユーザーが組織から除外された場合、API は 403 を返します。フォールバックとして stale データを丸ごと返すと、古い `orgInterId` を使って再び API を叩き、また 403 が返る。このサイクルが繰り返されて、ユーザーはエラー画面から抜け出せなくなります。

問題の本質は、**「表示にしか使わないフィールド」と「API リクエストに使うフィールド」がひとつの型にまとめられていること**です。フォールバックで安全に使えるのは前者だけなのに、後者まで一緒に返してしまうことでエラーが伝播します。

## 解決策: 表示用データと API 用データを型で分離する

stale データを「表示用（display）」と「API 用（api）」に分離し、TypeScript の Discriminated Union(*1) で状態を明示します。

```ts
type DisplayContext = { orgName: string }
type ApiContext = { orgInterId: number }

type SafeContext =
  | { status: 'active'; display: DisplayContext; api: ApiContext }
  | { status: 'display-only'; display: DisplayContext; api: null }
  | { status: 'unavailable'; display: null; api: null }

async function resolveSafeContext(
  fetchOrg: () => Promise<OrgContext>,
  staleContext: OrgContext | null,
): Promise<SafeContext> {
  try {
    const org = await fetchOrg()
    return {
      status: 'active',
      display: { orgName: org.orgName },
      api: { orgInterId: org.orgInterId },
    }
  } catch {
    if (staleContext) {
      return {
        status: 'display-only',
        display: { orgName: staleContext.orgName },
        api: null, // ✅ API 用データは返さない
      }
    }
    return { status: 'unavailable', display: null, api: null }
  }
}
```

`status` フィールドで絞り込むと、`api` が `null` でないことが型レベルで保証されます(*1)。API を呼ぶコードは `status === 'active'` のブランチでしか `api.orgInterId` にアクセスできないため、stale な ID でリクエストを送る事故を型システムが防ぎます。

```ts
const ctx = await resolveSafeContext(fetchOrg, staleContext)

if (ctx.status === 'active') {
  // ctx.api は ApiContext 型 — 安全に API を呼べる
  await callApi(ctx.api.orgInterId)
} else if (ctx.status === 'display-only') {
  // サイドバーには組織名を表示できるが、API は呼ばない
  showSidebar(ctx.display.orgName)
  showRecoveryPage()
} else {
  // 表示データもない — サインイン画面へ
  redirectToSignIn()
}
```

## 一時的障害と恒久的無効化を区別する

ただし、すべての API エラーで `api: null` にすると、ネットワークの瞬断でもリカバリーページが表示されてしまいます。一時的障害（5xx やタイムアウト）と恒久的な無効化（403 Forbidden）を区別することで、UX を改善できます(*2)。

```ts
type ErrorClassification = 'transient' | 'permanent'

async function resolveContextWithStrategy(
  fetchOrg: () => Promise<OrgContext>,
  staleContext: OrgContext | null,
  classifyError: (error: unknown) => ErrorClassification,
): Promise<SafeContext> {
  try {
    const org = await fetchOrg()
    return {
      status: 'active',
      display: { orgName: org.orgName },
      api: { orgInterId: org.orgInterId },
    }
  } catch (error) {
    const kind = classifyError(error)

    if (kind === 'transient' && staleContext) {
      // 一時的障害 → stale データを全部使う（すぐ復旧する想定）
      return {
        status: 'active',
        display: { orgName: staleContext.orgName },
        api: { orgInterId: staleContext.orgInterId },
      }
    }
    if (staleContext) {
      // 恒久的無効化 → 表示用だけ返す
      return {
        status: 'display-only',
        display: { orgName: staleContext.orgName },
        api: null,
      }
    }
    return { status: 'unavailable', display: null, api: null }
  }
}
```

`classifyError` を外から注入することで、判定ロジックを呼び出し元が決められます。HTTP ステータスコードで分岐する例を示します。

```ts
const classifyError = (error: unknown): ErrorClassification => {
  if (error instanceof Response && error.status === 403) return 'permanent'
  return 'transient'
}
```

一時的障害なら stale データを**全部**使って表示を維持し、恒久的な無効化なら表示用だけに制限してエラーループを断ち切る。この使い分けが、UX とデータ整合性の両立につながります。

## まとめ

- stale データを**丸ごと**フォールバックに使うと、無効化されたコンテキストで API を呼び続けるエラーループになりうる
- Discriminated Union で「表示用（display）」と「API 用（api）」を分離し、型レベルで安全境界を作る(*1)
- `classifyError` で一時的障害と恒久的無効化を区別すれば、UX とデータ整合性を両立できる(*2)

フォールバックは「何を返すか」だけでなく、**「何を返さないか」**が重要です。

## 参考リンク

- *1: [TypeScript Handbook — Narrowing (Discriminated Unions)](https://www.typescriptlang.org/docs/handbook/2/narrowing.html#discriminated-unions)
- *2: [Handling Errors — Next.js Docs](https://nextjs.org/docs/app/getting-started/error-handling)
