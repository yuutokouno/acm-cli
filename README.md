# acm-cli

Obsidian の Markdown ノートをスキャン・ベクトル化し、意味ベースの検索と知識ギャップ検出を行う Go 製 CLI ツール。

```
acm scan ./vault      # ノートを全部ベクトル化して SQLite に保存
acm search "テスト設計" # 意味的に近いノートを検索
acm gaps              # 知識のつながりが薄い領域を検出
```

## 機能

- **`acm scan`** — Vault 内の `.md` ファイルを走査し、embedding を生成して SQLite に保存。2回目以降は変更ファイルのみ差分更新。
- **`acm search`** — クエリをベクトル化し、コサイン類似度で意味的に近いノートを返す。キーワードが違っても概念が近ければヒットする。
- **`acm gaps`** — 共有タグや wiki-link で「関連するはず」のノートペアのうち、ベクトル距離が遠いペアを知識ギャップとして提示する。

## インストール

```bash
git clone https://github.com/yuutokouno/acm-cli
cd acm-cli
go build -o acm .
```

> **依存ライブラリ:** `go-sqlite3` は CGO を使用するため、C コンパイラ（`gcc` / `clang`）が必要です。

## ONNX モデルのセットアップ

embedding 生成にはローカルの ONNX モデルが必要です。

```bash
# Hugging Face から paraphrase-multilingual-MiniLM-L12-v2 をダウンロード
mkdir -p models
# model.onnx と tokenizer.json を models/ に配置
```

モデルファイルは `.gitignore` に含まれています（サイズが大きいため）。

> モデルファイルが存在しない場合、acm-cli は **MockEmbedder**（SHA-256 ベースの決定論的ベクトル）を使用します。検索精度は落ちますが、動作確認には十分です。

## 使い方

### acm scan

```bash
# Vault をスキャンして DB を作成（デフォルト: <vault>/.acm/acm.db）
acm scan ./my-vault

# DB パスを指定
acm scan ./my-vault --db ./acm.db

# 全ファイルを強制再スキャン
acm scan ./my-vault --db ./acm.db --force
```

出力例:

```
Scanned 42 notes (5 new, 3 updated, 34 unchanged)
```

### acm search

```bash
acm search "テスト設計" --db ./acm.db

# 結果件数を変更（デフォルト: 10）
acm search "依存性注入" --db ./acm.db --limit 5
```

出力例:

```
Results for "テスト設計":

1. [0.91] testing/classical-school.md
   tags: #testing #mock
2. [0.85] architecture/onion.md
   tags: #architecture #di
```

### acm gaps

```bash
acm gaps --db ./acm.db

# 閾値と件数を指定
acm gaps --db ./acm.db --threshold 0.4 --limit 20

# Markdown ファイルに出力
acm gaps --db ./acm.db --export gaps.md
```

出力例:

```
Knowledge Gaps (threshold: 0.30):

1. go/basics.md <-> testing/khorikov.md
   similarity: 0.18
   Suggested: "Go 基本文法 と ユニットテストの原則 の関係"
```

## アーキテクチャ

詳細は [`docs/architecture.md`](docs/architecture.md) を参照。

```
acm-cli/
├── cmd/          # cobra コマンド定義（scan / search / gaps）
├── memo/         # Memo 構造体 + Store / VectorStore interface
├── markdown/     # .md ファイルの走査とパース
├── storage/      # SQLite 実装（memo.Store + memo.VectorStore）
├── embedding/    # Embedder interface / ONNX 実装 / MockEmbedder
├── similarity/   # コサイン類似度 + ギャップ検出ロジック
├── models/       # ONNX モデルファイル置き場（.gitignore 済み）
└── testdata/     # テスト用 Markdown ファイル
```

## 開発

```bash
# テストを全て実行
go test ./...

# ビルド
go build -o acm .
```

## 今後の拡張（Phase 2 予定）

- `acm gather` — テーマ関連ノートを収集して Claude へのコンテキストファイルを生成
- `acm slides` — 関連ノートから Slidev 形式スライドを自動生成
- `acm watch` — Vault のファイル監視と自動スキャン
- TUI（bubbletea によるインタラクティブ UI）

## ライセンス

MIT
