# 🎨 Go Manga Kit

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/go-manga-kit)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-manga-kit)](https://github.com/shouni/go-manga-kit/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 概要 (About) - 自動ページ分割対応・作画制作Workflowライブラリ

**Go Manga Kit** は、非構造化ドキュメントを解析し、AIによる**キャラクターDNAの一貫性を維持した作画**を行うためのエンジニア向けライブラリです。

[Gemini Image Kit](https://github.com/shouni/gemini-image-kit) を描画コアに採用。独自の**オート・チャンク・システム**により、1ページあたり最大6パネルでの自動スライス生成を行います。Gemini File API を最大限に活用し、リソースの事前アップロードとキャッシュ戦略を組み合わせることで、高速かつ安定した作画制作を実現します。

---

## ✨ コア・コンセプト (Core Concepts)

* **🧬 Character DNA System**: `domain.Character` に定義したSeed値と視覚特徴をプロンプトへ動的に注入。全ページを通じてキャラクターの外見を一貫させることが可能です。
* **📑 Auto-Chunk Pagination**: パネル数が上限を超えると自動でページをスライス。AIの描画限界を回避し、複数枚構成の漫画を安定して生成します。
* **⚡ Smart Asset Preloading**: 生成前に全アセットを Gemini File API へ並列アップロード。`singleflight` 制御により、同一URLの二重アップロードを完全に排除し、APIクォータを節約します。
* **🎯 Visual Anchor Mapping**: デフォルトキャラクター（デザインシート）を常にリソースの0番（`input_file_0`）に固定。AIが迷うことなく基準スタイルを参照できる仕組みを提供します。
* **📐 Dynamic Layout Director**: ページごとに「主役パネル（Big Panel）」を動的に決定。単調なコマ割りを防ぎ、ドラマチックな演出を自動生成します。

---

## 🏗 システムスタック (System Stack)

| レイヤー | 技術 / ライブラリ | 役割 |
| --- | --- | --- |
| **Intelligence** | **Gemini 3.0 Flash** | ネーム（台本）構成およびマルチモーダル推論 |
| **Artistic** | **Nano Banana** | DNA注入と空間構成プロンプトによる一括作画 |
| **Resilience** | `singleflight` & `sync.Map` | アセットアップロードの重複抑制と高速再利用 |
| **Concurrency** | `x/time/rate` & `errgroup` | 安定したAPIクォータ遵守とリソースの並列準備 |
| **Drawing Engine** | `shouni/gemini-image-kit` | Gemini File API 連携および描画コア |
| **I/O Factory** | `shouni/go-remote-io` | GCS/Localの透過的なアクセス |

---

## 🎨 5つのワークフロー (Workflows)

以下は `pkg/workflow` インターフェースによって定義される、漫画制作の主要な工程です。

| ワークフロー | 担当インターフェース | 内容 |
| --- | --- | --- |
| **1. Designing**  | `DesignRunner` | キャラクターのDNA（特徴）を固定し、一貫性のあるデザインシートを生成。 |
| **2. Scripting**  | `ScriptRunner` | Web/テキストから、キャラクター・セリフ・構図を含むJSON台本を生成。 |
| **3. Panel Gen** | `PanelImageRunner` | 台本の各パネル（コマ）に対応する画像を、並列かつレート制限を遵守しながら個別に生成。 |
| **4. Publishing** | `PublishRunner` | 画像とテキストを統合し、HTML/Markdown等で出力。 |
| **5. Page Gen**   | `PageImageRunner` | 生成済みのパネル画像を、JSON台本に基づきページ単位にレイアウトし、ページ画像を生成。 |

---

## 📦 パッケージ構成 (Package Layout)

| パッケージ | 役割 |
| --- | --- |
| **`pkg/asset`** | GCSやローカルパスなど、異なるストレージ間でのパス解決（resolver）を担う。 |
| **`pkg/domain`** | `Character`, `Panel`, `Manga` 等の基底モデル。DNA情報やコアとなるデータ構造を定義。 |
| **`pkg/generator`** | **中核機能**。`PageGenerator` や `GroupGenerator` による作画・レイアウト制御を担当。 |
| **`pkg/parser`** | ソーステキストをネーム（台本）へ解析・変換。 |
| **`pkg/prompts`** | 描画AIへの空間構成指示や、テンプレート管理を行うプロンプトの司令塔。 |
| **`pkg/publisher`** | 生成したアセットを書き出す最終出力を担当。 |
| **`pkg/runner`** | `workflow` インターフェースを満たす具体的な実行実体（各工程のメインロジック）。 |
| **`pkg/workflow`** | 全体のワークフロー定義、インターフェース、および `Builder` による統合。 |

---

## 📂 プロジェクト構造 (Project Structure)

```text
go-manga-kit/
└── pkg/             # 公開ライブラリパッケージ
    ├── asset/       # アセット管理 (パス解決、リソース管理)
    ├── config/      # 環境変数管理
    ├── domain/      # ドメインモデル (character.go, manga.go)
    ├── generator/   # 生成戦略 (builder.go, page/group_generator.go)
    ├── parser/      # 構文解析
    ├── prompts/     # プロンプト構築
    ├── publisher/   # 成果物出力 (publisher.go)
    ├── runner/      # ワークフローの実行処理
    └── workflow/    # ワークフローの管理
```

---

## 🏗️ 作画生成シーケンスフロー (Image Generation Sequence Flow)

```mermaid
sequenceDiagram
    participant APP as Application
    participant Comp as generator.MangaComposer
    participant Page as generator.PageGenerator
    participant Asset as gemini-image-kit.AssetManager
    participant API as Gemini API (File API / Nano Banana)

    Note over APP, Comp: 1. アセット事前準備 (Parallel)
    APP->>Comp: PrepareCharacterResources / PreparePanelResources

    loop 各ユニークURL (Character/Panel)
        Comp->>Comp: getOrUploadResource (URL Key)
        Note right of Comp: singleflight で二重送出を防止
        Comp->>Asset: UploadFile(ctx, url)
        Asset->>API: File API Upload
        API-->>Asset: File API URI (gs://...)
        Asset-->>Comp: URI
        Comp->>Comp: PanelResourceMap[URL] に保存
    end

    Note over APP, API: 2. ページ生成 (Inference)
    APP->>Page: Execute
    Page->>Page: collectResources (MapからURIを取得)
    Note right of Page: 0番目にデフォルトキャラ(input_file_0)を固定

    Page->>API: GenerateContent (FileAPIURIs + Prompt + Seed)
    API-->>Page: Generated Image Data
    Page-->>APP: []imagedom.ImageResponse

```

---

## 🤝 依存関係 (Dependencies)

* [shouni/gemini-image-kit](https://github.com/shouni/gemini-image-kit) - Gemini 画像作成抽象化
* [shouni/go-remote-io](https://github.com/shouni/go-remote-io) - GCS、およびローカルファイルシステムへの I/O 操作を統一化

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
