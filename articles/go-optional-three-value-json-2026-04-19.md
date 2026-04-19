---
title: "Go のジェネリクスで JSON の「未指定・null・値あり」を型安全に扱う"
emoji: "🔧"
type: "tech"
topics: ["go", "json", "tips"]
published: false
---

## はじめに

Go で REST API の PATCH エンドポイントを実装していると、ある厄介な問題にぶつかります。クライアントから送られてきた JSON の中で、あるフィールドが **省略された（変更しない）** のか、**明示的に `null` が指定された（値をクリアする）** のかを区別できないのです。

```json
// リクエスト1: nickname を変更しない（フィールド省略）
{ "name": "Alice" }

// リクエスト2: nickname をクリアする（明示的に null）
{ "name": "Alice", "nickname": null }
```

Go の `encoding/json` はどちらのケースでもフィールドをゼロ値にするため、この2つを見分けられません。本記事では、Go のジェネリクスを使った `Optional[T]` 型でこの **three-value problem** を型安全に解決するパターンを紹介します。

## three-value problem: ポインタだけでは足りない

最初に思いつく対策は、フィールドをポインタにすることです。

```go
type UpdateUserRequest struct {
    Name     *string `json:"name"`
    Nickname *string `json:"nickname"`
}
```

しかしポインタでは2つの状態しか表現できません。

| JSON の状態 | Go の値 | 意味 |
|------------|---------|------|
| `"nickname": "Bob"` | `*string → "Bob"` | 値をセット |
| `"nickname": null` **または** フィールド省略 | `nil` | ??? |

`null` が送られた場合も、フィールドが省略された場合も、どちらも `nil` になってしまいます(*1)。「値をクリアしたい」と「変更しない」を区別できなければ、部分更新のロジックを正しく書けません。

## Optional[T] の設計

この問題を解決するために、3つの状態を明示的に持つジェネリック型 `Optional[T]` を定義します(*2)。

```go
package optional

import (
	"bytes"
	"encoding/json"
)

type Optional[T any] struct {
	Set   bool // JSON にフィールドが存在したか
	Null  bool // 値が null だったか
	Value T    // 実際の値
}
```

3つの状態をそれぞれ生成するコンストラクタを用意します。

```go
func NewValue[T any](v T) Optional[T] {
	return Optional[T]{Set: true, Null: false, Value: v}
}

func NewNull[T any]() Optional[T] {
	return Optional[T]{Set: true, Null: true}
}

func NewUnset[T any]() Optional[T] {
	return Optional[T]{}
}
```

状態を判定するメソッドも定義します。

```go
func (o Optional[T]) IsUnset() bool { return !o.Set }
func (o Optional[T]) IsNull() bool  { return o.Set && o.Null }
func (o Optional[T]) IsSet() bool   { return o.Set && !o.Null }
```

### JSON のアンマーシャル

`UnmarshalJSON` を実装すると、`encoding/json` がフィールドを検出したときだけこのメソッドが呼ばれます。呼ばれなければ `Set` はゼロ値の `false` のままなので、フィールド省略を自然に検出できます(*1)。

```go
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(data, []byte("null")) {
		o.Null = true
		return nil
	}
	o.Null = false
	return json.Unmarshal(data, &o.Value)
}
```

### JSON のマーシャル

レスポンスや永続化で JSON に書き出す場合も考慮します。`IsZero()` を定義しておくと、Go 1.24 以降の `omitzero` タグや、`omitempty` タグとの併用で未設定フィールドを省略できます。

```go
func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.Set || o.Null {
		return []byte("null"), nil
	}
	return json.Marshal(o.Value)
}

func (o Optional[T]) IsZero() bool {
	return !o.Set
}
```

## 既存値への適用ヘルパー

`Optional[T]` をデコードしたら、既存のデータに適用する処理が必要です。`ApplyToPtr` ヘルパーを用意すると、各フィールドの更新ロジックが簡潔になります。

```go
func ApplyToPtr[T any](o Optional[T], current *T) *T {
	if o.IsUnset() {
		return current
	}
	if o.IsNull() {
		return nil
	}
	v := o.Value
	return &v
}
```

PATCH ハンドラでの使用例です。

```go
type User struct {
	Name     string
	Nickname *string
}

type UpdateUserRequest struct {
	Name     Optional[string] `json:"name,omitempty"`
	Nickname Optional[string] `json:"nickname,omitempty"`
}

func applyUpdate(user *User, req UpdateUserRequest) {
	if req.Name.IsSet() {
		user.Name = req.Name.Value
	}
	user.Nickname = ApplyToPtr(req.Nickname, user.Nickname)
}
```

`Nickname` は `ApplyToPtr` 一行で3パターンを正しく処理します。

- **未指定** → 既存値を維持
- **null** → `nil` にクリア
- **値あり** → 新しいポインタをセット

## まとめ

- Go の `encoding/json` ではポインタを使っても「フィールド省略」と「明示的な null」を区別できない(*1)
- `Optional[T]` ジェネリック型で `Set` / `Null` / `Value` の3状態を明示すれば、部分更新を型安全に扱える(*2)
- `IsZero()` と `omitempty` を組み合わせると、未設定フィールドを JSON 出力から省略できる

PATCH API に限らず、イベントソーシングの部分更新イベントや、GraphQL の nullable input にも応用できるパターンです。

## 参考リンク

- *1: [JSON and Go - The Go Blog](https://go.dev/blog/json)
- *2: [Tutorial: Getting started with generics - The Go Programming Language](https://go.dev/doc/tutorial/generics)
