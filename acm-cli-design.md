# acm-cli — Obsidian ナレッジベース強化ツール 設計書

## 概要

Obsidian の Markdown ノートをスキャン・ベクトル化・検索・分析する Go 製 CLI ツール。
自分の学習ナレッジを構造化し、知識ギャップの検出や LT 資料作成を支援する「パーソナルインフラ」。

**Phase 1（今回作るもの）:**
- `acm scan` — Vault 内の .md ファイルをスキャンし、embedding を生成して SQLite に保存
- `acm search` — クエリをベクトル化し、類似ノートを検索
- `acm gaps` — ノート間の距離が遠いペアを検出し、知識ギャップを提示

**将来の拡張（今は作らない）:**
- `acm gather` — テーマに関連するノートを収集し、Claude に渡すコンテキストファイルを生成
- `acm add` — 新規ノートを作成（フロントマター自動付与）
- `acm slides` — 関連ノートから Slidev 形式スライドを生成
- `acm watch` — Vault をファイル監視し、変更時に自動スキャン
- `acm stats` — ノート数、カバレッジ、学習傾向の可視化
- Slack Bot 連携（音声メモの自動取り込み）
- TUI（bubbletea によるインタラクティブ UI）

---

## 技術スタック

- **Go 1.22+**
- **cobra** — CLI コマンド定義・引数解析
- **SQLite** + **sqlite-vec** — ノートメタデータ + ベクトル検索
- **ONNX Runtime** (Go バインディング) — ローカル embedding 生成
- **paraphrase-multilingual-MiniLM-L12-v2** — 多言語対応 384 次元モデル
- **goldmark** — Markdown パーサー
- **lipgloss** — ターミナル出力スタイリング（Phase 1 では最小限）

---

## ディレクトリ構成

```
acm-cli/
├── main.go                 # エントリーポイント（cobra root command）
├── go.mod
├── go.sum
│
├── cmd/                    # CLI コマンド定義
│   ├── root.go             # ルートコマンド、グローバルフラグ
│   ├── scan.go             # acm scan
│   ├── search.go           # acm search
│   └── gaps.go             # acm gaps
│
├── memo/                   # ドメインロジック
│   ├── memo.go             # Memo 構造体、ドメインロジック
│   └── store.go            # Store interface 定義
│
├── embedding/              # ベクトル生成
│   ├── embedder.go         # Embedder interface
│   └── onnx.go             # ONNX Runtime 実装
│
├── markdown/               # Obsidian ファイル操作
│   ├── parser.go           # Markdown パース、フロントマター抽出
│   └── scanner.go          # Vault 内 .md ファイルの走査
│
├── storage/                # データ永続化
│   ├── sqlite.go           # SQLite + sqlite-vec 実装（Store interface を満たす）
│   └── migrations.go       # テーブル作成 SQL
│
├── similarity/             # 類似度計算・ギャップ検出
│   ├── cosine.go           # コサイン類似度
│   └── gaps.go             # ギャップ分析ロジック
│
├── models/                 # ONNX モデルファイル置き場
│   └── .gitkeep            # モデルは .gitignore、README にDL手順を記載
│
├── testdata/               # テスト用の .md ファイル群
│   ├── go-basics.md
│   ├── test-design.md
│   └── onion-architecture.md
│
├── .gitignore
└── README.md
```

---

## データモデル

### Memo 構造体

```go
package memo

import "time"

type Memo struct {
    ID          string    // SHA256(file_path)
    FilePath    string    // Vault 内の相対パス
    Title       string    // ファイル名 or H1
    Content     string    // Markdown 本文
    ContentHash string    // SHA256(Content) — 変更検知用
    Tags        []string  // フロントマターの tags
    Links       []string  // [[wiki-link]] のリンク先
    CreatedAt   time.Time
    UpdatedAt   time.Time
    ScannedAt   time.Time
}
```

### SQLite テーブル

```sql
CREATE TABLE IF NOT EXISTS notes (
    id TEXT PRIMARY KEY,
    file_path TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    tags TEXT,              -- JSON 配列 '["Go", "CLI"]'
    links TEXT,             -- JSON 配列 '["cobra入門", "bubbletea"]'
    created_at DATETIME,
    updated_at DATETIME,
    scanned_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- sqlite-vec による仮想テーブル
CREATE VIRTUAL TABLE IF NOT EXISTS note_embeddings USING vec0(
    note_id TEXT PRIMARY KEY,
    embedding float[384]
);
```

---

## Interface 設計

### Store（memo/store.go）

```go
package memo

type Store interface {
    Save(m Memo) error
    GetByID(id string) (*Memo, error)
    GetByPath(path string) (*Memo, error)
    GetAll() ([]Memo, error)
    GetContentHash(id string) (string, error)
    Delete(id string) error
}
```

### Embedder（embedding/embedder.go）

```go
package embedding

type Embedder interface {
    Embed(text string) ([]float64, error)
    EmbedBatch(texts []string) ([][]float64, error)
}
```

### VectorStore（memo/store.go に追加）

```go
type VectorStore interface {
    SaveEmbedding(noteID string, vec []float64) error
    Search(queryVec []float64, limit int) ([]SearchResult, error)
    GetAllEmbeddings() (map[string][]float64, error)
}

type SearchResult struct {
    NoteID     string
    FilePath   string
    Title      string
    Similarity float64
}
```

テストや差し替えが必要になった時点で interface を抽出する（Go の慣習）。
ただし Store と Embedder は最初から定義する（SQLite → Notion、ONNX → API への差し替えを想定）。

---

## コマンド仕様

### acm scan

```
使い方: acm scan <vault-path>

Vault 内の .md ファイルをスキャンし、embedding を生成して保存する。
初回は全ファイル、2回目以降は content_hash が変わったファイルのみ更新。

フラグ:
  --force    全ファイルを強制的に再スキャン
  --db       DB ファイルのパス（デフォルト: <vault-path>/.acm/acm.db）

出力例:
  🔍 Scanning /Users/yuto/obsidian-vault ...
  ████████████████████░░░░░░  80/100 notes
  ✅ Scanned 100 notes (38 new, 12 updated, 50 unchanged)
```

### acm search

```
使い方: acm search <query>

クエリ文字列を embedding に変換し、コサイン類似度で類似ノートを検索。

フラグ:
  --limit N    結果件数（デフォルト: 10）
  --db         DB ファイルのパス

出力例:
  🔎 Results for "テスト設計":

  1. [0.92] Testing/Khorikov本.md
     tags: #testing #unit-test
  2. [0.87] Testing/Classical-School.md
     tags: #testing #mock
  3. [0.81] Architecture/Onion.md
     tags: #architecture #di
```

### acm gaps

```
使い方: acm gaps

全ノートのベクトルを比較し、関連するはずなのに距離が遠いペアを検出。
「知っているべきなのに知識が欠けている領域」を提示する。

フラグ:
  --threshold F    類似度の閾値（デフォルト: 0.3）
  --limit N        表示件数（デフォルト: 10）
  --export FILE    結果を Markdown ファイルに出力

出力例:
  🧩 Knowledge Gaps:

  1. Testing/Khorikov本.md ↔ Go/CLI設計.md
     similarity: 0.24
     💡 Suggested: "Go でのテスト設計パターン"

  2. Architecture/DI.md ↔ Go/基本文法.md
     similarity: 0.19
     💡 Suggested: "Go における依存性注入"
```

---

## 処理フロー

### acm scan の内部処理

```
1. vault-path から .md ファイル一覧を取得（markdown/scanner.go）
2. 各ファイルについて:
   a. ファイルを読み込み、フロントマターとコンテンツを分離（markdown/parser.go）
   b. content_hash を計算（SHA256）
   c. DB の既存ハッシュと比較
   d. 変更がある場合:
      - Memo 構造体を生成
      - Store.Save() で DB に保存
      - Embedder.Embed() でベクトル生成
      - VectorStore.SaveEmbedding() でベクトル保存
   e. 変更がない場合: スキップ
3. 結果サマリーを表示（new / updated / unchanged の件数）
```

### acm search の内部処理

```
1. クエリ文字列を Embedder.Embed() でベクトル化
2. VectorStore.Search() でコサイン類似度上位 N 件を取得
3. 結果を整形して表示
```

### acm gaps の内部処理

```
1. VectorStore.GetAllEmbeddings() で全ベクトルを取得
2. 全ペアのコサイン類似度を計算（similarity/cosine.go）
3. タグやリンクで「関連するはず」のペアを特定
4. 関連するはずなのに類似度が低いペアを抽出
5. 結果を整形して表示
```

---

## ONNX モデルのセットアップ

embedding 生成はローカルで行う（API 費用ゼロ）。

```bash
# Hugging Face から ONNX モデルをダウンロード
# paraphrase-multilingual-MiniLM-L12-v2（384次元、多言語対応）
mkdir -p models
cd models
# モデルファイル（model.onnx, tokenizer.json 等）をダウンロード
```

README にダウンロード手順を記載する。
models/ ディレクトリは .gitignore に追加（ファイルサイズが大きいため）。

---

## Claude Code への実装指示

### Step 0: プロジェクト初期化
1. `mkdir acm-cli && cd acm-cli`
2. `go mod init github.com/<username>/acm-cli`
3. `git init`
4. `gh repo create acm-cli --private --source=. --remote=origin`
5. `.gitignore` を作成
6. GitHub Projects でカンバンボード作成、Step ごとに Issue 作成

#### .gitignore
```
# Binary
acm-cli

# ONNX models (large files)
models/*.onnx
models/*.bin

# SQLite
*.db

# IDE
.vscode/
.idea/

# OS
.DS_Store
```

#### GitHub Issues
```bash
gh issue create --title "Step 1: CLI 骨格 + Markdown パーサー" --label "core"
gh issue create --title "Step 2: SQLite ストレージ" --label "storage"
gh issue create --title "Step 3: Embedding 生成" --label "embedding"
gh issue create --title "Step 4: acm scan 実装" --label "command"
gh issue create --title "Step 5: acm search 実装" --label "command"
gh issue create --title "Step 6: acm gaps 実装" --label "command"
```

### Step 1: CLI 骨格 + Markdown パーサー
1. `go get github.com/spf13/cobra@latest`
2. `main.go` + `cmd/root.go` でルートコマンド作成
3. `acm --version` が動くことを確認
4. `markdown/scanner.go` — ディレクトリを再帰走査して .md ファイルパスを返す
5. `markdown/parser.go` — .md ファイルを読み、フロントマター（YAML）とコンテンツを分離
6. `memo/memo.go` — Memo 構造体を定義
7. テスト: `testdata/` の .md ファイルを読み込んでパースできることを確認
8. コミット

### Step 2: SQLite ストレージ
1. `go get github.com/mattn/go-sqlite3`
2. `storage/migrations.go` — テーブル作成 SQL
3. `storage/sqlite.go` — Store interface + VectorStore interface を実装
4. `memo/store.go` — interface 定義
5. テスト: Memo の CRUD が動くことを確認
6. コミット

### Step 3: Embedding 生成
1. ONNX Runtime の Go バインディングをセットアップ
2. `embedding/embedder.go` — Embedder interface
3. `embedding/onnx.go` — ONNX モデルによる embedding 生成実装
4. `similarity/cosine.go` — コサイン類似度計算
5. テスト: テキストを渡して 384 次元ベクトルが返ることを確認
6. コミット

**注意:** ONNX Runtime の Go バインディングが困難な場合、
代替案として以下を検討:
- Python の sentence-transformers を subprocess で呼ぶ
- ローカルの HTTP サーバーとして embedding API を立てる
- Go の native な embedding ライブラリ（github.com/nicholasgasior/goembed 等）を使う

### Step 4: acm scan 実装
1. `cmd/scan.go` — scan コマンドの cobra 定義
2. scanner → parser → hasher → store → embedder → vector store の一連のパイプライン
3. 差分検知（content_hash 比較）
4. プログレス表示（最小限、fmt.Printf でOK）
5. テスト: testdata/ を scan して DB にデータが入ることを確認
6. コミット

### Step 5: acm search 実装
1. `cmd/search.go` — search コマンドの cobra 定義
2. クエリ → embedding → VectorStore.Search() → 結果表示
3. `--limit` フラグ
4. テスト: scan 後に search して関連ノートが返ることを確認
5. コミット

### Step 6: acm gaps 実装
1. `cmd/gaps.go` — gaps コマンドの cobra 定義
2. 全ベクトル取得 → ペア比較 → ギャップ検出
3. `--threshold`, `--limit`, `--export` フラグ
4. テスト: ギャップが検出されることを確認
5. コミット

---

## 設計上の重要な判断

1. **Go のフラットなパッケージ構成**
   - Python 版 ACM では Onion Architecture（domain/application/infrastructure/presentation/di）
   - Go 版では Go の思想に合わせて機能ごとのフラットな構成
   - ただし interface による依存の逆転は必要な箇所（Store, Embedder）で適用
   - 面接で「両方やったことで、設計パターンは言語の文化に合わせて適用すべきだと学んだ」と言える

2. **SQLite を選んだ理由**
   - CLI ツール = ローカルで完結。サーバー不要のファイルDB が最適
   - sqlite-vec でベクトル検索も可能
   - Obsidian Vault 内に .acm/acm.db として配置、Vault と一緒に同期可能

3. **embedding をローカルで生成する理由**
   - API 費用ゼロ（Claude Max Plan は API とは別課金）
   - オフラインでも動作する
   - ノート数百〜数千件なら十分な速度

4. **ONNX 実装が困難な場合のフォールバック**
   - Step 3 で ONNX の Go バインディングに詰まった場合
   - まず Python subprocess で動くものを作る
   - 後から ONNX native に差し替え（Embedder interface があるので差し替え容易）
