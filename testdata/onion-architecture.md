---
tags: [architecture, DDD, Go]
created: 2024-03-15
---

# Onion Architecture

依存の方向をドメイン層に向け、インフラ層をプラグイン化するアーキテクチャ。

## レイヤー構成

1. Domain（中心）— エンティティ・値オブジェクト・ドメインサービス
2. Application — ユースケース・インターフェース定義
3. Infrastructure — DB・外部 API・ファイルシステムの実装
4. Presentation — HTTP ハンドラー・CLI

## Go での適用

Go では interface が薄く使える。Repository interface をドメイン層に置き、実装を infrastructure に。

関連: [[DI パターン]] / [[Clean Architecture]]
