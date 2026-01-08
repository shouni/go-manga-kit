# 🎨 Go Manga Kit

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/go-manga-kit)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-manga-kit)](https://github.com/shouni/go-manga-kit/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 概要 (About) - 自動ページ分割対応・解説漫画生成パイプライン

**Go Manga Kit** は、非構造化ドキュメントを解析し、AIによる**キャラクターDNAの一貫性を維持した作画**を行うためのエンジニア向けライブラリです。

[Gemini Image Kit](https://github.com/shouni/gemini-image-kit) を描画コアに採用。独自の**オート・チャンク・システム**により、1ページあたり最大6パネルで自動分割を行います。複数枚の「連載漫画」として出力できるハイエンドなツールキットです。

---

## ✨ コア・コンセプト (Core Concepts)

* **🧬 Character DNA System**: `domain.Character` に定義したSeed値と視覚特徴をプロンプトへ動的に注入します。全ページを通じてキャラクターの外見を一貫させることが可能です。
* **📑 Auto-Chunk Pagination**: パネル数が上限を超えると自動でページをスライスします。AIの描画限界を回避し、複数枚構成の漫画を安定して生成します。
* **📖 Script-to-Manga Pipeline**: Markdown等のソースを `parser` が解析し、演出指示を含む構造化データへ変換します。これを `generator` が受け取り、一括で作画を行う一気通貫の設計です。
* **📐 Dynamic Layout Director**: ページごとに「主役パネル（Big Panel）」を動的に決定します。単調なコマ割りを防ぎ、ドラマチックな演出を自動生成します。

---

## 📦 パッケージ構成 (Package Layout)

| パッケージ | 役割 |
| --- | --- |
| **`pkg/domain`** | `Character`, `Panel`, `Manga` 等の基底モデル。DNA情報やコアとなるデータ構造を定義します。 |
| **`pkg/parser`** | Markdown や正規表現を用いて、ソーステキストをネーム（台本）へ解析・変換します。 |
| **`pkg/generator`** | **中核機能**。`PageGenerator` や `GroupGenerator` による作画・レイアウト制御を担います。 |
| **`pkg/prompt`** | 描画AIへの空間構成指示や、テンプレート管理を行うプロンプトの司令塔です。 |
| **`pkg/publisher`** | 生成したアセットを統合画像（PNG）やHTMLとして書き出す最終出力を担当します。 |

---

## 📂 プロジェクト構造 (Project Structure)

```text
go-manga-kit/
└── pkg/             # 公開ライブラリパッケージ
    ├── domain/      # ドメインモデル (character.go, manga.go)
    ├── generator/   # 生成戦略 (builder.go, page/group_generator.go)
    ├── parser/      # 構文解析 (markdown.go, regex.go)
    ├── prompt/      # プロンプト構築 (template.go)
    └── publisher/   # 成果物出力 (publisher.go)
```

---

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
