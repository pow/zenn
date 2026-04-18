---
title: "TypeScript でエラーコード体系を設計して API エラーレスポンスを統一する"
emoji: "🏷"
type: "tech"
topics: ["typescript", "nodejs", "tips"]
published: false
---

## はじめに

API を複数人で開発していると、エラーレスポンスの形式がエンドポイントごとにバラバラになりがちです。あるエンドポイントは `{ error: "not found" }` を返し、別のエンドポイントは `{ message: "User not found", statusCode: 404 }` を返す。フロントエンドはエラーの種類を判定するために文字列マッチングを強いられ、メッセージ文言の変更でフロントが壊れるリスクもあります。

本記事では、RFC 9457（Problem Details for HTTP APIs）(*1) の考え方を参考に、TypeScript でエラーコード体系を設計し、API のエラーレスポンスを統一するパターンを紹介します。

## エラーコードの範囲設計

エラーを文字列メッセージではなく**一意のコード**で識別します。マイクロサービスやモジュールが増えてもコードが衝突しないよう、サービスごとに範囲を予約します。

```ts
// 共通エラー: E0001–E0006
export const ERR_UNAUTHORIZED = "E0001" as const
export const ERR_FORBIDDEN = "E0002" as const
export const ERR_NOT_FOUND = "E0003" as const
export const ERR_CONFLICT = "E0004" as const
export const ERR_VALIDATION = "E0005" as const
export const ERR_INTERNAL = "E0006" as const

// ユーザーサービス: E0100–E0199
export const ERR_USER_EMAIL_TAKEN = "E0100" as const
export const ERR_USER_DEACTIVATED = "E0101" as const

// 注文サービス: E0200–E0299
export const ERR_ORDER_ALREADY_SHIPPED = "E0200" as const
export const ERR_ORDER_ITEM_OUT_OF_STOCK = "E0201" as const

export type ErrorCode =
  | typeof ERR_UNAUTHORIZED
  | typeof ERR_FORBIDDEN
  | typeof ERR_NOT_FOUND
  | typeof ERR_CONFLICT
  | typeof ERR_VALIDATION
  | typeof ERR_INTERNAL
  | typeof ERR_USER_EMAIL_TAKEN
  | typeof ERR_USER_DEACTIVATED
  | typeof ERR_ORDER_ALREADY_SHIPPED
  | typeof ERR_ORDER_ITEM_OUT_OF_STOCK
```

`as const` でリテラル型にし、`ErrorCode` をユニオン型でまとめます。存在しないコードを使うとコンパイル時にエラーになるため、タイポや古いコードの参照を防げます。

範囲を 100 刻みにしておくと、1サービスあたり最大 99 個のエラーコードを定義でき、実用上は十分です。新しいサービスを追加するときは次の範囲（E0300–E0399 など）を予約するだけで済みます。

## AppError クラスの実装

エラーコードを運ぶ `AppError` クラスを定義します(*2)。標準の `Error` を継承するので `throw` / `catch` でそのまま使えます。

```ts
import { type ErrorCode } from "./error-codes.ts"

export class AppError extends Error {
  readonly code: ErrorCode
  readonly detail?: string

  constructor(code: ErrorCode, message: string, detail?: string) {
    super(message)
    this.name = "AppError"
    this.code = code
    this.detail = detail
  }

  toJSON() {
    return {
      code: this.code,
      message: this.message,
      ...(this.detail !== undefined && { detail: this.detail }),
    }
  }
}
```

`toJSON()` でスタックトレースなどの内部情報を除外し、クライアントに返すべきフィールドだけに絞ります。RFC 9457(*1) の `type` → `code`、`title` → `message`、`detail` → `detail` と対応づけると、標準に近い構造になります。

## HTTP ステータスコードへのマッピング

エラーコードから HTTP ステータスを導出するマッピングを `Record<ErrorCode, number>` で定義します。

```ts
import { type ErrorCode } from "./error-codes.ts"

const STATUS_MAP: Record<ErrorCode, number> = {
  E0001: 401, // Unauthorized
  E0002: 403, // Forbidden
  E0003: 404, // Not Found
  E0004: 409, // Conflict
  E0005: 400, // Bad Request
  E0006: 500, // Internal Server Error
  E0100: 409, // Email taken → Conflict
  E0101: 403, // Deactivated → Forbidden
  E0200: 409, // Already shipped → Conflict
  E0201: 422, // Out of stock → Unprocessable Entity
}

export const toHttpStatus = (code: ErrorCode): number => STATUS_MAP[code]
```

`Record<ErrorCode, number>` にすることで、`ErrorCode` に新しいコードを追加したときに `STATUS_MAP` への追加が漏れるとコンパイルエラーになります。エラーコードの網羅性が型レベルで保証される仕組みです(*2)。

API ハンドラーではこのように統一的にエラーを返せます。

```ts
const handleError = (error: unknown): Response => {
  if (error instanceof AppError) {
    return Response.json(error.toJSON(), {
      status: toHttpStatus(error.code),
    })
  }
  return Response.json(
    { code: "E0006", message: "Internal server error" },
    { status: 500 },
  )
}
```

すべてのエンドポイントが `handleError` を経由することで、エラーレスポンスが `{ code, message, detail? }` に統一されます。フロントエンドは `code` フィールドだけで分岐でき、メッセージ文言の変更に影響されません。

## まとめ

- エラーコードに**範囲**を持たせると、サービスが増えてもコードの衝突を防げる
- `AppError` の `toJSON()` で RFC 9457(*1) に近い統一フォーマットを実現し、スタックトレースの漏洩も防止できる
- `Record<ErrorCode, number>` でマッピングを定義すると、新しいエラーコード追加時のマッピング漏れを型レベルで検知できる(*2)

## 参考リンク

- *1: [RFC 9457 — Problem Details for HTTP APIs](https://www.rfc-editor.org/rfc/rfc9457)
- *2: [TypeScript Handbook — Classes](https://www.typescriptlang.org/docs/handbook/2/classes.html)
