---
title: "Next.js App Router で Apollo Client の初期データとキャッシュを安全に併用する"
emoji: "🔄"
type: "tech"
topics: ["nextjs", "typescript", "graphql", "tips"]
published: false
---

## はじめに

Next.js App Router で Apollo Client を使っていると、**Server Component で取得したデータを Client Component でどう扱うか**が設計上の悩みどころになります。

Server Component で GraphQL API から初回データを取得し、Client Component ではタブ切り替えなどでクライアントサイドのキャッシュを活用したい。しかし素朴に実装すると、初期データの二重取得や stale データの表示といった問題が起きます。

本記事では、実プロジェクトで試した **initialData フォールバック + cache-first** のパターンと、**ID ガードによる stale データ防止**の2つの実践パターンを紹介します。

## 課題: SSR データと Apollo キャッシュの二重管理

典型的な構成として、Server Component でデータを取得し props で Client Component に渡すケースを考えます。

```tsx
// Server Component (page.tsx)
export default async function ProjectPage({
  params,
  searchParams,
}: {
  params: Promise<{ projectId: string }>
  searchParams: Promise<{ caseId?: string }>
}) {
  const { projectId } = await params
  const { caseId = 'default' } = await searchParams
  const summary = await fetchSummary(projectId, caseId)

  return (
    <SummaryView
      initialData={summary}
      initialCaseId={caseId}
      projectId={projectId}
    />
  )
}
```

Client Component 側で `useQuery` を呼ぶと、Server Component ですでに取得済みのデータをもう一度フェッチしてしまいます。かといって `useQuery` を使わなければ、タブ切り替えでデータを取得できません。

## パターン1: initialData フォールバック + cache-first

解決策は、**Apollo の `fetchPolicy: 'cache-first'` と initialData のフォールバックを組み合わせる**ことです(*1)。

まず、どのデータソースを使うかを決める純粋関数を用意します。

```ts
export type DataSource<T> =
  | { source: 'query'; data: T }
  | { source: 'initial'; data: T }
  | { source: 'none' }

export function resolveData<T>(params: {
  queryData: T | undefined
  initialData: T | undefined
}): DataSource<T> {
  const { queryData, initialData } = params
  if (queryData !== undefined) return { source: 'query', data: queryData }
  if (initialData !== undefined) return { source: 'initial', data: initialData }
  return { source: 'none' }
}
```

ロジックは単純です。Apollo のクエリ結果があればそれを優先し、なければ Server Component から渡された initialData にフォールバックします。Client Component では次のように使います。

```tsx
'use client'

import { useQuery } from '@apollo/client'
import { resolveData } from '@/lib/resolve-data'

export function SummaryView({ initialData, initialCaseId, projectId }) {
  const [activeCaseId, setActiveCaseId] = useState(initialCaseId)

  const { data: queryData } = useQuery(GET_SUMMARY, {
    variables: { projectId, caseId: activeCaseId },
    fetchPolicy: 'cache-first',
  })

  const resolved = resolveData({
    queryData: queryData?.summary,
    initialData,
  })

  if (resolved.source === 'none') return <LoadingSkeleton />
  return <SummaryContent data={resolved.data} />
}
```

`cache-first` を指定すると、同じ variables でのクエリはキャッシュから即座に返されます(*1)。初回表示では Apollo キャッシュが空なので `queryData` は `undefined` になり、initialData がフォールバックとして表示されます。タブを切り替えると `activeCaseId` が変わり、新しい variables で `useQuery` がフェッチを実行します。一度取得したタブに戻ると、キャッシュヒットして即座に表示されます。

## パターン2: ID ガードで stale データを防ぐ

パターン1には落とし穴があります。**ブラウザの「戻る」ボタンやタブ切り替え時に、initialData が現在の選択と一致しないケース**です。

例えば、ユーザーが `?caseId=A` のページを開き、タブで `caseId=B` に切り替えた後、ブラウザバックで戻ったとします。Server Component は `caseId=A` で再レンダリングされますが、React の状態復元で `activeCaseId` が `B` のままになることがあります。このとき `initialData`（A のデータ）を `caseId=B` の表示に使ってしまうと、**別のケースのデータが表示される**バグになります。

解決策は、**initialData の ID と現在のアクティブな ID を比較するガード**を追加することです。

```ts
export function resolveDataWithGuard<T>(params: {
  queryData: T | undefined
  initialData: T | undefined
  initialDataKey: string
  currentKey: string
}): DataSource<T> {
  const { queryData, initialData, initialDataKey, currentKey } = params
  if (queryData !== undefined) return { source: 'query', data: queryData }
  if (initialData !== undefined && initialDataKey === currentKey) {
    return { source: 'initial', data: initialData }
  }
  return { source: 'none' }
}
```

`initialDataKey`（Server Component が取得時に使った ID）と `currentKey`（クライアント側の現在の選択 ID）が一致するときだけ initialData を使います。不一致なら `{ source: 'none' }` を返し、ローディング表示にフォールバックします(*2)。

```tsx
const resolved = resolveDataWithGuard({
  queryData: queryData?.summary,
  initialData,
  initialDataKey: initialCaseId,
  currentKey: activeCaseId,
})
```

このガードにより、stale な initialData が表示されることを防げます。

## データ解決ロジックを純粋関数にする利点

`resolveData` と `resolveDataWithGuard` を純粋関数として切り出すことで、React コンポーネントから独立してテストできます。テストでは Apollo や Next.js のセットアップが一切不要です。

```ts
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { resolveDataWithGuard } from '../src/resolve-data.ts'

test('initialData key mismatch returns none', () => {
  const result = resolveDataWithGuard({
    queryData: undefined,
    initialData: { title: 'Case A summary' },
    initialDataKey: 'case-a',
    currentKey: 'case-b',
  })
  assert.equal(result.source, 'none')
})
```

Apollo Client の `useQuery` や Next.js の Server Component と組み合わせる部分は結合テストで検証し、データ解決のロジックはユニットテストで高速にカバーする。この分離が、安心してリファクタリングできる設計につながります。

## まとめ

- **initialData + cache-first** で SSR → CSR のシームレスなデータ引き継ぎを実現する(*1)
- **ID ガード**で initialData の stale 表示を防ぎ、タブ切り替えやブラウザバックでも正しいデータを表示する
- データ解決ロジックを純粋関数に切り出すと、テストしやすく再利用性も高い

`fetchPolicy` の選択は他にも `cache-and-network`（キャッシュを返しつつバックグラウンドで最新取得）などがあり、データの鮮度要件に応じて使い分けてください(*1)。

## 参考リンク

- *1: [Queries - Apollo GraphQL Docs](https://www.apollographql.com/docs/react/data/queries)
- *2: [Getting Started: Fetching Data - Next.js Docs](https://nextjs.org/docs/app/getting-started/fetching-data)
