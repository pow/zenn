# Zenn 記事リポジトリ

## プロジェクト概要
Zenn CLI を使った技術記事管理リポジトリ。

## 記事作成ルール

### ファイル命名規則
- ファイル名: `articles/{slug}.md`
- slug は英数字・ハイフンで構成（12〜50文字）
- 例: `articles/claude-code-tips-2026-04-06.md`

### Frontmatter 形式
```yaml
---
title: "記事タイトル（60文字以内推奨）"
emoji: "絵文字1つ"
type: "tech"
topics: ["topic1", "topic2", "topic3"]
published: false
---
```

### 記事の品質基準
- 具体的なコード例やコマンド例を含める
- 実際に動作する内容であること
- 読者が再現・実践できる内容にする
- 日本語で執筆する
- 見出し（##, ###）を使って構造化する
- 1500〜3000文字程度を目安にする

### topics に使えるタグ例
claudecode, claude, ai, llm, github, typescript, javascript, python,
rust, go, docker, linux, shell, git, vscode, productivity, tips

## コマンド
- `npx zenn preview` - ローカルプレビュー
- `npx zenn new:article --slug {slug}` - 新規記事作成
