---
title: "Go × GraphQL で DataLoader のキャッシュが別リクエストに漏れるバグを防ぐ"
emoji: "🔒"
type: "tech"
topics: ["go", "graphql", "dataloader", "gqlgen"]
published: false
---

## はじめに

Go で GraphQL サーバーを構築していると、N+1 問題の解決策として DataLoader を導入するのは定番です。しかし、DataLoader のインスタンスをリクエスト間で共有してしまうと、**あるユーザーのリクエストで取得したデータが別のユーザーのリクエストに漏れる**という深刻なキャッシュ汚染バグが発生します。

本記事では、この問題の原因と、Factory パターンを使った安全な解決方法を紹介します。

## DataLoader とキャッシュ汚染の仕組み

### DataLoader の基本

DataLoader は、同一リクエスト内の複数の個別データ取得をバッチ化する仕組みです。gqlgen と組み合わせて使う場合、リゾルバーから個別に `Load(id)` を呼び出しても、内部でバッチクエリにまとめてくれます。

```go
// DataLoader の基本的な使い方
type UserLoader struct {
    // 内部にキャッシュを持つ
    cache map[string]*User
    batch []string
}

func (l *UserLoader) Load(id string) (*User, error) {
    // キャッシュにあればそのまま返す
    if user, ok := l.cache[id]; ok {
        return user, nil
    }
    // なければバッチに追加して一括取得
    l.batch = append(l.batch, id)
    // ...
}
```

### 問題: シングルトンで共有するとどうなるか

よくある間違いは、DI コンテナや初期化処理で DataLoader をシングルトンとして生成し、全リクエストで共有してしまうことです。

```go
// ❌ 危険な例: アプリケーション起動時に1回だけ生成
type Server struct {
    userLoader *UserLoader // 全リクエストで共有される
}

func NewServer(db *sql.DB) *Server {
    return &Server{
        userLoader: NewUserLoader(db), // 起動時に1つだけ作る
    }
}
```

この場合、以下の流れでキャッシュ汚染が起こります。

1. ユーザー A のリクエストが `userLoader.Load("user-1")` を実行
2. 取得結果がキャッシュに格納される
3. ユーザー B のリクエストが `userLoader.Load("user-1")` を実行
4. **ユーザー A のリクエスト時にキャッシュされたデータがそのまま返る**

認可フィルタリングをリゾルバー層で行っている場合、本来ユーザー B に見せてはいけないデータが返る可能性があります。

## Factory パターンによる解決

### 考え方

DataLoader は**リクエストごとに新しいインスタンスを生成**すべきです。Factory パターンを使って、リクエストのコンテキストに紐づけた DataLoader を提供します。

### 実装

まず、DataLoader を生成する Factory を定義します。

```go
// DataLoader を生成する Factory 関数
type UserLoaderFactory func(ctx context.Context) *UserLoader

func NewUserLoaderFactory(db *sql.DB) UserLoaderFactory {
    return func(ctx context.Context) *UserLoader {
        return &UserLoader{
            cache: make(map[string]*User),
            fetch: func(ids []string) ([]*User, error) {
                return fetchUsersByIDs(ctx, db, ids)
            },
        }
    }
}
```

次に、HTTP ミドルウェアでリクエストごとに DataLoader を生成し、context に格納します。

```go
type contextKey struct{ name string }

var loadersKey = &contextKey{"dataloaders"}

// Loaders はリクエストスコープの DataLoader 群
type Loaders struct {
    UserLoader *UserLoader
}

func DataLoaderMiddleware(factory UserLoaderFactory) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            loaders := &Loaders{
                UserLoader: factory(r.Context()),
            }
            ctx := context.WithValue(r.Context(), loadersKey, loaders)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// リゾルバーから取得するヘルパー
func GetLoaders(ctx context.Context) *Loaders {
    return ctx.Value(loadersKey).(*Loaders)
}
```

リゾルバーでは context から取得して使います。

```go
func (r *queryResolver) User(ctx context.Context, id string) (*User, error) {
    // リクエストごとに独立した DataLoader を使用
    return GetLoaders(ctx).UserLoader.Load(id)
}
```

### DI フレームワークとの統合

Google Wire などの DI ツールを使っている場合、Provider に Factory を登録します。

```go
// wire.go
var GraphQLSet = wire.NewSet(
    NewUserLoaderFactory,  // *UserLoader ではなく Factory を提供
    NewResolver,
    // ...
)
```

ポイントは、DI コンテナには **Factory（関数）を登録**し、実際の DataLoader インスタンスはミドルウェアで毎リクエスト生成するという点です。

## 複数サービスがある場合の注意点

マイクロサービス構成で複数の GraphQL サブグラフがある場合、**すべてのサービスで同じパターンを適用**する必要があります。1つでもシングルトンの DataLoader が残っていると、そこがキャッシュ汚染の穴になります。

チェックリスト:

- [ ] 各サービスの DataLoader が Factory 経由で生成されているか
- [ ] ミドルウェアでリクエストごとにインスタンスが作られているか
- [ ] DI コンテナに DataLoader インスタンスが直接登録されていないか
- [ ] テストでリクエスト間のキャッシュ独立性を検証しているか

## テストで検証する

キャッシュ汚染が起きないことをテストで確認しましょう。

```go
func TestDataLoaderIsolation(t *testing.T) {
    db := setupTestDB(t)
    factory := NewUserLoaderFactory(db)

    // リクエスト1: user-1 を取得
    ctx1 := context.Background()
    loader1 := factory(ctx1)
    user1, err := loader1.Load("user-1")
    require.NoError(t, err)
    require.Equal(t, "user-1", user1.ID)

    // リクエスト2: 別の DataLoader インスタンス
    ctx2 := context.Background()
    loader2 := factory(ctx2)

    // loader2 のキャッシュは空であることを確認
    // （キャッシュヒットではなく DB から取得される）
    user1Again, err := loader2.Load("user-1")
    require.NoError(t, err)
    require.Equal(t, "user-1", user1Again.ID)

    // 2つのローダーが別インスタンスであることを確認
    require.NotSame(t, loader1, loader2)
}
```

## まとめ

- DataLoader のキャッシュはリクエスト間で共有してはいけない
- Factory パターンで、リクエストごとに新しい DataLoader を生成する
- HTTP ミドルウェアで context に格納し、リゾルバーから安全に取得する
- DI コンテナにはインスタンスではなく Factory を登録する
- マイクロサービス構成では全サービスに同じパターンを適用する

このパターンはシンプルですが、見落とすとセキュリティに直結する問題を引き起こします。GraphQL サーバーに DataLoader を導入する際は、最初から Factory パターンで設計することをおすすめします。
