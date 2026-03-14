# acm-cli アーキテクチャ解説

## 何を解決するツールか

Obsidian にノートを書き溜めていくと、**「あの知識どこに書いたっけ」「自分の知識の穴がわからない」** という問題が起きる。

キーワード検索（grep）は「すでに知っているワードでしか探せない」。
acm-cli は**意味ベースの検索**と**知識ギャップ検出**でこれを解決する。

```
acm scan ./vault     → ノートを全部ベクトル化して DB に保存
acm search "テスト設計" → 「TDD」「モック」「品質保証」なども引っかかる
acm gaps             → 「関連しているはずなのに知識が繋がっていない領域」を提示
```

---

## 技術スタック

| 技術 | 役割 | 選定理由 |
|------|------|---------|
| Go 1.22+ | 言語 | 静的型付け・高速・単一バイナリ配布 |
| cobra | CLI フレームワーク | サブコマンド・フラグ・ヘルプ自動生成 |
| SQLite | ローカル DB | サーバー不要のファイル DB。Vault と一緒に iCloud 同期可能 |
| ONNX Runtime | 推論エンジン | PyTorch モデルをランタイム非依存形式で Go から実行 |
| paraphrase-multilingual-MiniLM-L12-v2 | embedding モデル | 日英混在対応・384次元・軽量 |

---

## パッケージ構成

```
acm-cli/
├── main.go           — エントリーポイント（cobra を呼ぶだけ）
├── cmd/              — CLI コマンド定義
│   ├── root.go       — ルートコマンド、--version
│   ├── scan.go       — acm scan
│   ├── search.go     — acm search
│   └── gaps.go       — acm gaps
│
├── memo/             — ドメイン層（最も依存されない）
│   ├── memo.go       — Memo 構造体
│   └── store.go      — Store / VectorStore interface 定義
│
├── markdown/         — Obsidian ファイル操作
│   ├── scanner.go    — .md ファイルを再帰走査
│   └── parser.go     — front matter / H1 / [[wikilink]] 抽出
│
├── storage/          — DB 実装（Store + VectorStore を満たす）
│   ├── migrations.go — テーブル作成 SQL
│   └── sqlite.go     — SQLiteStore 実装
│
├── embedding/        — ベクトル生成
│   ├── embedder.go   — Embedder interface
│   ├── mock.go       — テスト用（決定論的偽ベクトル）
│   ├── onnx.go       — 本番用（ONNX モデルで推論）
│   └── wordpiece.go  — 純 Go WordPiece tokenizer
│
└── similarity/       — 類似度計算（純 Go、外部依存なし）
    └── cosine.go     — Cosine() + TopN()
```

**Go のフラットパッケージ**: Python の Onion Architecture（domain/application/infrastructure）のような階層は Go の文化に合わない。機能ごとのフラット構成が標準。

---

## データモデル

```go
type Memo struct {
    ID          string    // SHA256(file_path)  ← ファイル移動を検知
    FilePath    string    // Vault 内の相対パス
    Title       string    // H1 見出し or ファイル名（フォールバック）
    Content     string    // front matter を除いた本文
    ContentHash string    // SHA256(Content) ← 差分スキャンで使う
    Tags        []string  // front matter の tags
    Links       []string  // [[wiki-link]] のリンク先
    CreatedAt   time.Time
    UpdatedAt   time.Time
    ScannedAt   time.Time
}
```

### SQLite テーブル

```sql
-- メタデータ
CREATE TABLE notes (
    id           TEXT PRIMARY KEY,    -- SHA256(file_path)
    file_path    TEXT NOT NULL UNIQUE,
    title        TEXT NOT NULL,
    content      TEXT NOT NULL,
    content_hash TEXT NOT NULL,       -- 変更検知用
    tags         TEXT,                -- JSON 配列 '["Go","CLI"]'
    links        TEXT,                -- JSON 配列 '["cobra入門"]'
    created_at   DATETIME,
    updated_at   DATETIME,
    scanned_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ベクトル（384次元を JSON でシリアライズ）
CREATE TABLE note_embeddings (
    note_id   TEXT PRIMARY KEY,
    embedding TEXT NOT NULL           -- '[0.12, -0.34, ...]'
);
```

sqlite-vec（ネイティブベクトルインデックス）は CGO 設定が複雑なため、
**ベクトルは JSON で保存し、コサイン類似度は Go 側で計算**する設計を採用。

---

## インタフェース設計

Go の interface は「差し替え可能にしたい箇所だけ」に使う。

```go
// memo/store.go

// Store: SQLite → Notion への差し替えを想定
type Store interface {
    Save(m Memo) error
    GetByID(id string) (*Memo, error)
    GetByPath(path string) (*Memo, error)
    GetAll() ([]Memo, error)
    GetContentHash(id string) (string, error)
    Delete(id string) error
}

// VectorStore: 将来 sqlite-vec などへの差し替えを想定
type VectorStore interface {
    SaveEmbedding(noteID string, vec []float64) error
    GetAllEmbeddings() (map[string][]float64, error)
}
```

```go
// embedding/embedder.go

// Embedder: ONNX → Python subprocess → API への差し替えを想定
type Embedder interface {
    Embed(text string) ([]float64, error)
    EmbedBatch(texts []string) ([][]float64, error)
}
```

### コンパイル時インタフェース検証

```go
var _ Embedder = (*MockEmbedder)(nil)
// ↑ これが nil でも代入できる = *MockEmbedder は Embedder を満たす
// 満たさない場合はビルドエラー（実行時ではなくビルド時に発覚）
```

---

## 処理フロー

### acm scan

```
vaultPath
  └─ markdown.Scan()
       └─ 各 .md ファイル
            ├─ os.ReadFile()
            ├─ markdown.Parse()
            │    └─ ParseResult{ Title, Content, Tags, Links }
            │
            ├─ ID          = hex(SHA256(filePath))
            ├─ ContentHash = hex(SHA256(content))
            │
            ├─ store.GetContentHash(id)
            │    ├─ 一致 → スキップ（unchanged）
            │    └─ 不一致 or 新規
            │         ├─ store.Save(Memo{...})
            │         ├─ embedder.Embed(title + "\n" + content)
            │         └─ store.SaveEmbedding(id, vec)
            │
            └─ 結果サマリー表示
                 "✅ Scanned 100 notes (38 new, 12 updated, 50 unchanged)"
```

### acm search

```
query 文字列
  └─ embedder.Embed(query)         → queryVec
       └─ store.GetAllEmbeddings() → map[noteID][]float64
            └─ store.GetAll()      → []Memo
                 └─ similarity.TopN(embeddings, notes, queryVec, limit)
                      └─ 結果表示
```

### acm gaps

```
store.GetAllEmbeddings() → map[noteID][]float64
store.GetAll()           → []Memo（タグ・リンク情報）

全ペアのコサイン類似度を計算
  └─ タグやリンクで「関連しているはず」のペアを特定
        └─ 類似度が低いペアを抽出 → ギャップとして提示
```

---

## Embedding の仕組み

### テキスト → ベクトルの変換

```
"Go言語のテスト設計"
       │
  WordPiece tokenizer（tokenizer.json から語彙ロード）
       │
  ["[CLS]", "go", "##言語", "のテスト", "設計", "[SEP]"]
       │
  token IDs: [101, 23456, 11234, ...]
       │
  ONNX モデル（paraphrase-multilingual-MiniLM-L12-v2）
       │
  last_hidden_state: shape [1, seq_len, 384]
       │
  attention-masked mean pooling
       │
  L2 正規化
       │
  384次元の単位ベクトル: [0.12, -0.34, 0.87, ...]
```

### モデルファイルのセットアップ（使用時に必要）

```bash
# ONNX Runtime shared library（macOS）
# https://github.com/microsoft/onnxruntime/releases からダウンロード

# モデルファイル
mkdir -p models
# models/model.onnx
# models/tokenizer.json
# → HuggingFace: sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2
```

モデルファイルがない場合は `MockEmbedder`（SHA-256 ベースの偽ベクトル）を使用。
テストはすべて `MockEmbedder` で動作する。

---

## Go のエラーハンドリング

```go
// ❌ アンチパターン（エラーを無視）
result, _ := someFunc()

// ✅ 必ず伝播（%w でラップするとエラーチェーンが辿れる）
result, err := someFunc()
if err != nil {
    return fmt.Errorf("scan %s: %w", path, err)
}

// 呼び出し元でエラー種別を判定
if errors.Is(err, sql.ErrNoRows) {
    // 「見つからない」は正常系として扱う
}
```

---

## テスト戦略

| テスト対象 | 方針 |
|-----------|------|
| markdown パース | testdata/ の実ファイルを使って結合テスト |
| SQLite CRUD | `:memory:` DB を使って高速テスト |
| コサイン類似度 | 数学的性質（同一=1、直交=0、逆向き=-1）を直接検証 |
| embedding | `MockEmbedder` を使用（モデルファイル不要） |
| scan コマンド | `MockEmbedder` + `:memory:` DB で E2E テスト |

**テスト関数名は日本語**（プロジェクト規約）:

```go
func Test_同一ベクトルのコサイン類似度は1(t *testing.T) { ... }
func Test_Memoを保存して取得できる(t *testing.T) { ... }
```

---

## 実装ステップ

| Step | 内容 | ステータス |
|------|------|-----------|
| 0 | プロジェクト初期化 | ✅ 完了 |
| 1 | CLI 骨格 + Markdown パーサー | ✅ 完了 |
| 2 | SQLite ストレージ | ✅ 完了 |
| 3 | Embedding 生成 + コサイン類似度 | ✅ 完了 |
| 4 | `acm scan` 実装 | 🔜 次 |
| 5 | `acm search` 実装 | 未着手 |
| 6 | `acm gaps` 実装 | 未着手 |
