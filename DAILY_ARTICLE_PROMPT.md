# 日次記事生成プロンプト（改善版）

以下の手順で Zenn の技術記事を1本作成してください。

---

## ステップ1: pow/spark リポジトリの分析（Chain of Thought）

pow/spark リポジトリを分析し、**思考過程を明示しながら**以下を整理してください。

### 1-1. 技術スタックの把握
以下のファイルを確認し、使用技術を一覧にまとめる：
- `package.json` / `Cargo.toml` / `go.mod` / `requirements.txt` 等
- README やドキュメント

### 1-2. 最近の変更の分析
直近1〜2週間のコミットを確認し、以下のフォーマットで整理する：

```
【変更の要約】
- コミット: {hash} {message}
- 変更ファイル: {paths}
- 技術的ポイント: {どんな技術課題を解決しているか}
```

### 1-3. 記事ネタ候補のブレインストーム
分析結果から、**最低3つ**の記事ネタ候補を挙げる。各候補について以下を評価する：

```
候補A: {テーマ名}
- 読者ニーズ: ★★★（高/中/低 — この技術を調べる人がどれくらいいるか）
- 独自性: ★★☆（実体験ベースの知見がどれくらい含まれるか）
- 実用性: ★★★（読者がすぐ使える度合い）
- コード例の豊富さ: ★★☆（具体的なコードをどれだけ示せるか）
→ 合計: 10/12
```

## ステップ2: 既存記事の確認と最終テーマ決定

pow/zenn の `articles/` ディレクトリ内の既存記事を確認し：
1. タイトルと topics の一覧を取得
2. ステップ1の候補と重複がないか確認
3. **最もスコアが高く、既存記事と重複しないテーマ**を1つ選ぶ

選定理由を1〜2文で明記すること。

## ステップ3: 記事構成の設計（アウトライン）

記事を書き始める前に、以下のアウトラインを作成する：

```
タイトル: {60文字以内}
想定読者: {誰に向けた記事か}
読後のゴール: {読者がこの記事を読んで何ができるようになるか}

## はじめに
- 導入で触れる課題/背景: {1文}

## 本題
- セクション1: {見出し} — {何を説明するか}
- セクション2: {見出し} — {何を説明するか}
- セクション3: {見出し} — {何を説明するか}
- 含めるコード例: {最低2つ、どんなコードか}

## サンプルコード計画
- 言語/ランタイム: {例: TypeScript (Node.js), Rust, Python 等}
- 配置先: samples/{slug}/
- ファイル構成:
  - src/ or main file: {実装コードの概要}
  - test file: {各コード例に対応するテストの概要}
- 依存: {必要なパッケージ — 最小限にする}
- 実行方法: {npm test, cargo test, pytest 等}

## まとめ
- 要点: {3つ}

## 参考リンク
- *1: {タイトル — URL（公式ドキュメント、GitHub、RFC 等）}
- *2: {タイトル — URL}
- （最低2つ。本文中の該当箇所に `*N` を付けて、どの出典に基づく記述かを明示する）
```

## ステップ4: 記事作成

CLAUDE.md のルールに従い、`articles/{slug}.md` に記事を作成する。

### 命名規則
- slug: `{テーマを表す英語}-{YYYY-MM-DD}`
- 例: `rust-error-handling-tips-2026-04-06`

### 品質チェックリスト（書き終えたら自己採点）
- [ ] タイトルが具体的で、何が学べるか分かる
- [ ] 「はじめに」で読者の課題に共感している
- [ ] コード例が最低2つ含まれ、コピペで動く
- [ ] 各コード例に「なぜこう書くのか」の説明がある
- [ ] pow/spark のプライベートコードをそのまま掲載していない
- [ ] 技術を一般化し、読者が自分のプロジェクトに応用できる
- [ ] 1500〜3000文字の範囲内
- [ ] 見出しで構造化されている
- [ ] 「参考リンク」セクションに `*1`, `*2`, ... の番号付きで出典が最低2つある
- [ ] 本文中の該当箇所に `(*N)` を付け、どの出典に基づく記述かが明確である
- [ ] すべての出典URLを WebFetch または WebSearch で実在確認済み（ステップ4.5参照）
- [ ] サンプルコードとテストが `samples/{slug}/` に作成され、全テストがパスしている（ステップ4.3参照）

## ステップ4.3: サンプルコードの実装とテスト実行（必須）

記事内のすべてのコード例が**実際に動作する**ことを、サンプルコードとテストで保証する。
このステップを飛ばしてはならない。

### 4.3.1 ディレクトリ構成

記事の slug に対応するディレクトリを `samples/` 配下に作成する。

```
samples/{slug}/
├── README.md          # セットアップ手順・実行方法
├── src/               # 記事中のコード例を動作する形にまとめたもの
│   ├── example1.{ext}
│   └── example2.{ext}
├── tests/             # 各コード例に対応するテスト
│   ├── example1.test.{ext}
│   └── example2.test.{ext}
└── package.json / Cargo.toml / requirements.txt 等
```

言語ごとの標準的な構成に従うこと（例: Rust なら `src/lib.rs` + `tests/`、
TypeScript なら `src/` + `*.test.ts`、Python なら `src/` + `test_*.py`）。

### 4.3.2 サンプルコードの作成ルール

- 記事中のコード例をそのまま含める（記事とサンプルでコードが乖離しないこと）
- 依存パッケージは最小限にする
- `README.md` にセットアップ手順と実行コマンドを記載する

### 4.3.3 テストの作成ルール

- 記事中の**各コード例に最低1つのテスト**を書く
- テストは「記事の説明どおりに動作すること」を検証する
- テスト名は何を検証しているかが分かるようにする

```
例: 記事で「NotFound を返すとステータス 404 になる」と書いたなら
→ テスト: "AppError::NotFound returns 404 status code"
```

### 4.3.4 テストの実行と結果確認

1. 依存をインストールする（`npm install`, `cargo build`, `pip install` 等）
2. テストを実行する（`npm test`, `cargo test`, `pytest` 等）
3. 以下のフォーマットで結果を記録する：

```
【テスト実行結果】
$ {実行コマンド}
  ✅ example1: AppError::NotFound returns 404 — PASSED
  ✅ example1: AppError::Internal returns 500 — PASSED
  ✅ example2: ... — PASSED
結果: 3/3 PASSED
```

4. テストが失敗した場合：
   - エラー内容を確認し、**サンプルコードを修正**する
   - サンプルコードの修正に伴い、**記事内の該当コード例も同期して修正**する
   - 再度テストを実行する
5. **全テストが PASSED になるまでステップ4.5に進まない**

## ステップ4.5: 出典の実在確認（必須）

記事を書き終えたら、**参考リンクに記載したすべてのURLが実在するか**を検証する。
このステップを飛ばしてはならない。

### 手順

1. 記事内の `*1`, `*2`, ... に対応するURLをすべてリストアップする
2. 各URLに対して **WebFetch** でアクセスし、HTTPステータスとページ内容を確認する
3. 以下のフォーマットで検証結果を記録する：

```
【出典検証結果】
- *1: https://docs.rs/axum/... → ✅ 200 OK（ページタイトル: "axum::error_handling"）
- *2: https://docs.rs/anyhow/... → ✅ 200 OK（ページタイトル: "anyhow"）
- *3: https://example.com/not-found → ❌ 404 Not Found → 代替URLを検索して差し替え
```

4. ❌ のURLがあった場合：
   - **WebSearch** で正しいURLを検索する
   - 見つかった正しいURLで記事内の該当箇所を修正する
   - 修正後に再度 WebFetch で確認する
5. **すべてのURLが ✅ になるまでステップ5に進まない**

---

## Few-Shot: 良い記事の例

### 例1: ライブラリ活用系

```markdown
---
title: "Axum でエラーハンドリングを整理する3つのパターン"
emoji: "🛡"
type: "tech"
topics: ["rust", "axum", "tips"]
published: false
---

## はじめに

Rust の Web フレームワーク Axum でAPIを開発していると、エラーハンドリングが散らかりがちです。
最初は各ハンドラで個別に処理していましたが、パターンが増えるにつれて統一的な方法が必要になりました。

本記事では、Axum プロジェクトで実際に試した3つのエラーハンドリングパターンを紹介します。

## パターン1: カスタムエラー型 + IntoResponse

Axum では `IntoResponse` トレイトを実装することで、任意の型をレスポンスに変換できます(*1)。
アプリケーション固有のエラー型を定義し、これを実装するのが最もシンプルなアプローチです。

｀｀｀rust
use axum::response::{IntoResponse, Response};
use axum::http::StatusCode;

enum AppError {
    NotFound(String),
    Internal(anyhow::Error),
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let (status, message) = match self {
            AppError::NotFound(msg) => (StatusCode::NOT_FOUND, msg),
            AppError::Internal(err) => (
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("Internal error: {err}"),
            ),
        };
        (status, message).into_response()
    }
}
｀｀｀

この方法のメリットは、各ハンドラの戻り値を `Result<impl IntoResponse, AppError>` に
統一できることです。`anyhow::Error` と組み合わせると、`?` 演算子で簡潔にエラー伝播できます(*2)。

## パターン2: ...（省略）

## パターン3: thiserror で構造化エラー

エラーの種類ごとに明示的な型を持ちたい場合は `thiserror` が便利です(*3)。

## まとめ

- **パターン1**（カスタムエラー型）: 小〜中規模プロジェクトに最適
- **パターン2**: ...
- **パターン3**: ...

プロジェクトの規模やチームの好みに合わせて選択してください。

## 参考リンク

- *1: [Axum 公式ドキュメント - Error Handling](https://docs.rs/axum/latest/axum/error_handling/index.html)
- *2: [anyhow クレート](https://docs.rs/anyhow/latest/anyhow/)
- *3: [thiserror クレート](https://docs.rs/thiserror/latest/thiserror/)
```

### 例1 に対応するサンプルコード・テスト

```
samples/axum-error-handling-tips-2026-04-06/
├── Cargo.toml
├── README.md
├── src/
│   └── lib.rs        # AppError 型 + IntoResponse 実装
└── tests/
    └── error_test.rs  # 各パターンの動作検証
```

**samples/axum-error-handling-tips-2026-04-06/src/lib.rs:**
```rust
use axum::response::{IntoResponse, Response};
use axum::http::StatusCode;

pub enum AppError {
    NotFound(String),
    Internal(anyhow::Error),
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let (status, message) = match self {
            AppError::NotFound(msg) => (StatusCode::NOT_FOUND, msg),
            AppError::Internal(err) => (
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("Internal error: {err}"),
            ),
        };
        (status, message).into_response()
    }
}
```

**samples/axum-error-handling-tips-2026-04-06/tests/error_test.rs:**
```rust
use axum::http::StatusCode;
use axum::response::IntoResponse;
use axum_error_handling::AppError;

#[tokio::test]
async fn not_found_returns_404() {
    let error = AppError::NotFound("user not found".to_string());
    let response = error.into_response();
    assert_eq!(response.status(), StatusCode::NOT_FOUND);
}

#[tokio::test]
async fn internal_error_returns_500() {
    let error = AppError::Internal(anyhow::anyhow!("db connection failed"));
    let response = error.into_response();
    assert_eq!(response.status(), StatusCode::INTERNAL_SERVER_ERROR);
}
```

**テスト実行結果:**
```
$ cargo test
  ✅ not_found_returns_404 — PASSED
  ✅ internal_error_returns_500 — PASSED
結果: 2/2 PASSED
```

---

### 例2: 開発プラクティス系

```markdown
---
title: "GitHub Actions で Rust のビルドキャッシュを効かせて CI を3倍速にする"
emoji: "⚡"
type: "tech"
topics: ["rust", "github", "docker", "tips"]
published: false
---

## はじめに

Rust プロジェクトの CI、遅くないですか？
私のプロジェクトではフルビルドに10分以上かかっていましたが、
キャッシュ戦略を見直すことで3分台まで短縮できました。

## なぜ Rust の CI は遅いのか

（具体的な原因の解説 + 計測データ）

## 解決策: 3層キャッシュ戦略

### 1. cargo registry のキャッシュ

`actions/cache` を使って依存クレートのレジストリをキャッシュします(*1)。
キーには `Cargo.lock` のハッシュを使い、依存が変わったときだけ再取得します(*2)。

｀｀｀yaml
- uses: actions/cache@v4
  with:
    path: |
      ~/.cargo/registry
      ~/.cargo/git
    key: cargo-registry-${{ hashFiles('**/Cargo.lock') }}
    restore-keys: cargo-registry-
｀｀｀

### 2. target ディレクトリのキャッシュ

（コード例 + 説明）

### 3. sccache の導入

Mozilla が開発した `sccache` を使うと、コンパイル結果をキャッシュして再ビルドを高速化できます(*3)。

（コード例 + 説明）

## 結果

| 施策 | ビルド時間 |
|------|-----------|
| 施策なし | 10分30秒 |
| registry キャッシュ | 7分15秒 |
| + target キャッシュ | 4分20秒 |
| + sccache | 3分10秒 |

## まとめ

- cargo registry → target → sccache の順にキャッシュを追加
- `Cargo.lock` のハッシュをキーにすると、依存更新時だけキャッシュが更新される(*2)
- sccache はローカル開発でも有効(*3)

## 参考リンク

- *1: [actions/cache 公式リポジトリ](https://github.com/actions/cache)
- *2: [GitHub Actions のキャッシュドキュメント](https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows)
- *3: [sccache - Shared Compilation Cache](https://github.com/mozilla/sccache)
```

---

## Few-Shot: 避けるべき記事の例（アンチパターン）

```markdown
❌ 悪い例:
title: "Rust について"
→ 抽象的すぎる。何が学べるか分からない

❌ 悪い例:
## はじめに
Rustは素晴らしい言語です。今回はRustについて書きます。
→ 読者の課題に触れていない。動機が不明

❌ 悪い例:
（コード例なし、概念の説明だけが続く）
→ 読者が実践できない

❌ 悪い例:
「〇〇というライブラリはパフォーマンスが良いと言われています」
→ 出典なし。「(*1)」のように脚注番号を付け、参考リンクで根拠を示す

❌ 悪い例:
参考リンクに推測URL（存在しないページ）を記載
→ ステップ4.5で WebFetch して実在確認してから掲載する

❌ 悪い例:
参考リンクはあるが、本文中のどの記述が出典に対応するか不明
→ 本文の該当箇所に「(*N)」を付け、読者が根拠を辿れるようにする

❌ 悪い例:
記事にコード例があるが、サンプルコードやテストがない
→ samples/{slug}/ にコードとテストを作成し、全テストがパスすることを確認する

❌ 悪い例:
サンプルコードと記事内のコード例が異なる
→ テスト修正で動くコードに直したら、記事側も必ず同期して更新する
```

---

## ステップ5: コミット

- `published: false` を確認
- 以下のファイルをまとめて git commit & push する：
  - `articles/{slug}.md` — 記事本体
  - `samples/{slug}/` — サンプルコード・テスト一式

---

## 補足: プロンプト設計の意図

| 手法 | 適用箇所 | 効果 |
|------|---------|------|
| **Chain of Thought** | ステップ1の段階的分析、ステップ2のスコアリング | テーマ選定の根拠が明確になり、質が安定する |
| **Few-Shot** | ステップ4の良い例・悪い例 | 出力フォーマットとトーンが安定する |
| **Self-Evaluation** | ステップ4の品質チェックリスト | 書き終えた記事を自己検証し、品質の底上げ |
| **Structured Output** | 各ステップのフォーマット指定 | 思考の抜け漏れを防ぐ |
| **Scoring Rubric** | ステップ1-3の評価基準 | 主観的判断を定量化し、再現性を高める |
| **Grounding（出典明記）** | 本文中の `(*N)` 脚注 + 参考リンクの `*N` 対応 | 主張と根拠の対応が明確になり、読者が検証可能 |
| **Code Verification** | ステップ4.3のサンプルコード実装+テスト実行 | 記事内コード例が実際に動作することを保証。テスト全通過まで次に進めない |
| **Verification Gate** | ステップ4.5の WebFetch による全URL検証 | ハルシネーションURL を確実に排除。全 ✅ まで次に進めない |
