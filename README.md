# 🎨 Go Manga Kit

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/go-manga-kit)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-manga-kit)](https://github.com/shouni/go-manga-kit/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 概要 (About) - テキストから「マンガ」を紡ぐ、AIオーケストレーション・キット

**Go Manga Kit** は、AI（Gemini/Imagen）を用いたマンガ生成の複雑な工程を自動化し、構造化するためのGo言語向けツールキットなのだ。

[Gemini Image Kit](https://github.com/shouni/gemini-image-kit) を描画エンジンとして活用し、Markdown形式の台本からキャラクターの一貫性を保ったマルチパネル画像、さらには演出の効いたWebtoon（縦読みマンガ）HTMLまでを一気通貫で生成できるのだ。

---

## ✨ 主な特徴 (Features)

* **📖 Script-to-Manga Pipeline**: Markdown形式の台本を解析し、AIが理解可能な詳細な描写指示（Visual Anchor）へ自動変換します。
* **🧬 Character DNA System**: `characters.json` で定義された視覚的特徴（Visual Cues）を各プロンプトに動的に注入し、全パネルを通してキャラクターの整合性を維持します。
* **📐 Unified Prompt Engine**: 高度なスタイルサフィックスを用いた、一貫性のある画風制御ロジックを搭載しています。
* **🎭 Multi-Mode Execution**: 単一パネルの生成から、全ページを統合したWebtoonレイアウトの生成まで柔軟に対応します。
* **🌐 Hybrid Publisher**: 生成されたコンテンツをMarkdown、HTML、画像としてローカルまたはGoogle Cloud Storageへ透過的に保存します。

---

## 📂 プロジェクト構造 (Project Layout)

```text
go-manga-kit/
├── bin/             # コンパイル済みバイナリ
├── cmd/             # CLIサブコマンド定義 (image, story, root)
├── examples/        # 設定・台本サンプル (characters.json, manga_script.md)
├── internal/
│   ├── builder/     # DIコンテナ・アプリの初期化・組み立て
│   ├── config/      # 環境変数・設定管理
│   ├── pipeline/    # 実行制御の司令塔 (Pipeline管理)
│   └── runner/      # 実行単位のコアロジック (Image, Page, Publish)
├── pkg/             # 再利用可能なライブラリ群
│   ├── domain/      # ドメインモデル (Manga, Character)
│   ├── parser/      # Markdown台本のパース・正規表現ロジック
│   ├── pipeline/    # 生成戦略 (Group, Pageごとの個別パイプライン)
│   ├── prompt/      # プロンプトテンプレートと構築
│   └── publisher/   # 成果物の保存・変換 (HTML, Assets)
├── output/          # 生成結果の出力先 (Images, HTML, MD)
└── main.go          # エントリーポイント

```

---

## 🛠️ 使い方 (Usage)

### 1. セットアップ

バイナリをビルドします。

```bash
go build -o bin/mangakit main.go

```

### 2. キャラクター定義の準備 (`examples/characters.json`)

キャラクターの見た目をJSONで定義します。

```json
{
  "metan": {
    "id": "metan",
    "name": "めたん",
    "seed": 20001,
    "reference_url": "https://...",
    "visual_cues": [
      "vibrant lavender hair",
      "massive twin-tails with spiral curls",
      "strictly following the outfit from reference image"
    ]
  }
}

```

### 3. コマンドの実行

**画像生成モード:**
Markdown形式の台本（スクリプト）を読み込み、指定したページの画像を生成します。

```bash
# 特定のページ(1ページ目)を生成
# -s でMarkdown台本、-c でキャラクター定義を指定します
./bin/mangakit image -p 1 -c examples/characters.json -s examples/manga_script.md

```

---

## 🤝 依存関係 (Dependencies)

* [shouni/gemini-image-kit](https://github.com/shouni/gemini-image-kit) - 高度な画像生成エンジン
* [shouni/go-remote-io](https://github.com/shouni/go-remote-io) - ストレージ抽象化
* [shouni/go-text-format](https://github.com/shouni/go-text-format) - Webtoon変換コア

---

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
