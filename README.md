# 🎨 Go Manga Kit

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/go-manga-kit)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-manga-kit)](https://github.com/shouni/go-manga-kit/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 概要 (About) - 技術を「物語」へ。高精細8コマ漫画生成パイプライン

**Go Manga Kit** は、非構造化された技術ドキュメントやWeb記事を解析し、AIによる「精密なネーム構成」と「キャラクターDNAの一貫性を維持した一括作画」を介して、認知的負荷の低い**解説漫画ページ**へと変換するハイエンド・ツールキットなのだ！

[Gemini Image Kit](https://github.com/shouni/gemini-image-kit) を描画エンジンとして活用し、単なる画像の羅列ではなく、1枚の3:4キャンバスに複数のワイドパネルを整列させる「マルチパネル・グリッド・システム」を搭載。日本式の読み順（右から左、上から下）に最適化された、本格的な漫画体験を提供するのだ。

---

## ✨ 主な特徴 (Features)

* **📖 Script-to-Manga Pipeline**: 「伝説の漫画編集者プロンプト」がドキュメントを解析。セリフ、構図指示、SHA1ハッシュ化された `speaker_id` を含む構造化データを自動生成。
* **🧬 Character DNA System**: キャラクターの視覚的特徴（Visual Cues）と参照URLをプロンプトへ動的に注入。全パネルで外見の一貫性を保持するのだ。
* **📐 Dynamic Layout Director**: ページ生成時にランダムで「主役パネル（Big Panel）」を決定。生成のたびに異なる演出を楽しめるのだ。
* **🎭 Multi-Stage Workflow**: 台本生成（JSON）→ 個別画像生成 → 統合ページ錬成の段階的プロセスにより、AIとの共同制作（Human-in-the-loop）を実現。
* **🛡️ Resilience & Rate Control**: **30s/req (2 RPM)** の厳格なレートリミット制御と、参照画像のTTL付きインメモリキャッシュにより、APIクォータを尊重しつつ安定した作画を継続するのだ。

---

## 🏗 システムスタック

| レイヤー | 技術 / ライブラリ | 役割 |
| --- | --- | --- |
| **Intelligence** | **Gemini 3.0 Flash** | 伝説の編集者プロンプトによるネーム構成 |
| **Artistic** | **Nano Banana** | DNA注入と空間構成プロンプトによる一括作画 |
| **Resilience** | **go-cache** | 参照画像のTTL管理（30分）による高速化 |
| **Concurrency** | `x/time/rate` | 安定したAPIクォータ遵守 |
| **I/O Factory** | `shouni/go-remote-io` | GCS/Localの透過的なアクセス |

---

## 🛠 ワークフローとサブコマンド

本ツールは、制作プロセスに応じて4つのサブコマンドを使い分けられるのだ。

| コマンド | 役割 | 出力 |
| --- | --- | --- |
| **`generate`** | **一括生成**。解析からパブリッシュまでを一気通貫で行うのだ。 | HTML, Images, MD |
| **`script`** | **台本生成**。AIによる構成案のみを出力。人間が調整したい時に使うのだ。 | **JSON** |
| **`image`** | **個別作画**。JSONを読み込み、パネルごとの画像とHTMLを生成するのだ。 | Images, HTML, MD |
| **`story`** | **最終錬成**。Markdown台本から「8コマ完成済み漫画ページ」を生成するのだ。 | **Final Image (PNG)** |

---

### 🎨 主要なフラグ (Major Flags)

| フラグ | ショートカット | 説明 | デフォルト値 | 必須 |
| --- | --- | --- | --- | --- |
| `--script-url` | **`-u`** | ソースとなるWebページのURL。コンテンツを自動抽出するのだ。 | **なし** | ✅ (※) |
| `--script-file` | **`-f`** | ローカルのテキストファイル、または `script` コマンドで出力したJSONパス。 | **なし** | ✅ (※) |
| `--output-file` | **`-o`** | 生成されるMarkdown/HTML、または台本JSONの保存先パスなのだ。 | `output/manga_plot.md` | ❌ |
| `--output-image-dir` | **`-i`** | 生成された画像を保存するディレクトリ（ローカルまたは `gs://...`）。 | `output/images` | ❌ |
| `--mode` | **`-m`** | キャラクター構成を指定 (`'duet'`, `'dialogue'` など)。 | `dialogue` | ❌ |
| `--model` | なし | テキスト生成（台本構成）に使用する Gemini モデル名なのだ。 | `gemini-3-flash-preview` | ❌ |
| `--image-model` | なし | 画像生成に使用する Gemini モデル名なのだ。 | `gemini-3-pro-image-preview` | ❌ |
| `--char-config` | **`-c`** | **キャラクターの視覚情報（DNA）を定義したJSONパスなのだ。** | `examples/characters.json` | ❌ |
| `--panel-limit` | **`-p`** | 生成する漫画パネルの最大数。開発時のコスト節約に便利なのだ。 | `10` | ❌ |
| `--http-timeout` | なし | Webリクエスト（スクレイピング等）のタイムアウト時間なのだ。 | `30s` | ❌ |

**(※) 注意:** `--script-url` または `--script-file` のいずれか一方は必ず指定する必要があるのだ！

---


## 💻 実行例 (Usage)

### 1. 最高の1枚を一気に作る (Standard)

```bash
./mangakit generate --script-url "https://example.com/tech-blog" -m duet

```

### 2. 人間とAIの共作フロー (Advanced)

```bash
# 1. 台本JSONの生成
./mangakit script -u "https://example.com/tech-blog" -o "output/my_script.json"

# (ここで JSON を編集可能)

# 2. JSONからプロット情報（Markdown）と個別画像を生成
./mangakit image -f "output/my_script.json" -o "output/manga_plot.md"

# 3. 編集済みMarkdownから「完成済み1ページ漫画」を錬成
./mangakit story -f "output/manga_plot.md" -o "output/final_manga_page.png"

```

---

## 📂 プロジェクト構造 (Project Layout)

```text
go-manga-kit/
├── cmd/             # CLIサブコマンド (generate, script, image, story)
├── internal/
│   ├── builder/     # DIコンテナ・アプリの初期化
│   ├── config/      # 環境変数・設定管理
│   ├── pipeline/    # 実行制御の司令塔 (Pipeline管理)
│   ├── prompt/      # 台本作成用テンプレート (Markdown)
│   └── runner/      # 実行コア (Script, Image, Page, Publish)
├── pkg/
│   ├── domain/      # ドメインモデル (Manga, Character DNA)
│   ├── parser/      # Markdown台本の構造解析・正規表現による要素抽出
│   ├── pipeline/    # 生成戦略 (Group: 個別, Page: 統合)
│   ├── prompt/      # 画像生成用 PromptBuilder
│   └── publisher/   # 成果物出力 (Webtoon HTML, Assets)
└── output/          # 生成結果 (Images, HTML, Markdown)

```

---

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。

**デフォルトキャラクター**: VOICEVOX:ずんだもん、VOICEVOX:四国めたん
