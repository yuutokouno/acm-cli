---
tags: [testing, unit-test, Go]
created: 2024-02-01
---

# テスト設計

良いテストは実装の詳細ではなく、振る舞いを検証する。

## 古典学派 vs ロンドン学派

- **古典学派**: 実オブジェクトを使い、結合度を下げない
- **ロンドン学派**: モックを多用し、協調オブジェクトを隔離

## Go でのテスト

```go
func TestAdd(t *testing.T) {
    got := Add(1, 2)
    if got != 3 {
        t.Errorf("Add(1,2) = %d; want 3", got)
    }
}
```

関連: [[Khorikov 本]] / [[モック設計]]
