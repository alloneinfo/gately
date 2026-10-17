# コントリビュート

Gately はコミュニティ主導のオープンソースプロジェクトで、Issue と Pull Request を歓迎します。リポジトリ：<https://github.com/wangwanjie/gately>

## ローカル開発

必要なもの：Go 1.26+、Node.js 24+、pnpm。

```bash
make web      # フロントエンドをビルド（web/dist に出力され、go:embed で埋め込まれます）
make build    # bin/gately をコンパイル
make test     # go test -race ./...
make lint     # gofmt、go vet、フロントエンドの型チェック
```

フロントエンドの開発時は `cd web && pnpm dev` を実行します。`/api` はローカルの `127.0.0.1:16680` にプロキシされます。

## コントリビュートの決まりごと

- Lucky との互換性は、公開されている観察可能な挙動、インターフェース、設定形式に基づきます。Lucky のバイナリやフロントエンド成果物から複製したコードは提出しないでください。
- 新しいモジュールには、設定構造とバリデーション、実行時の起動・停止、REST エンドポイント、ユニットテストが必要です。Lucky から移行できるモジュールは `luckyimport.Importer` を実装し、概要ページに状態を表示したいモジュールは `module.Summarizer` を実装します。
- コードのコメントは中国語で書き、制御文には必ず波括弧を使います。

## 追加のテスト

一部のテストはデフォルトでスキップされ、追加の条件が必要です。

| テスト | 有効化の方法 |
| --- | --- |
| ACME エンドツーエンド（Pebble） | pebble と pebble-challtestsrv をインストールし、`GATELY_PEBBLE_BIN`、`GATELY_PEBBLE_SRC` を設定する |
| 実際の公開 STUN サーバー | `GATELY_NET_TEST=1` |
| 実際の Lucky 設定でのインポート検証 | `GATELY_LUCKY_SAMPLE` など。各モジュールの `TestImportRealSample` を参照 |

## ライセンス

[Apache License 2.0](https://github.com/wangwanjie/gately/blob/main/LICENSE)
