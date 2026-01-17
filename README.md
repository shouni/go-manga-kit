# 🎨 Go Manga Kit

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/go-manga-kit)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-manga-kit)](https://github.com/shouni/go-manga-kit/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 概要 (About) - 自動ページ分割対応・作画制作Workflowライブラリ

**Go Manga Kit** は、非構造化ドキュメントを解析し、AIによる**キャラクターDNAの一貫性を維持した作画**を行うためのエンジニア向けライブラリです。

[Gemini Image Kit](https://github.com/shouni/gemini-image-kit) を描画コアに採用。独自の**オート・チャンク・システム**により、1ページあたり最大6パネルで自動分割を行います。GCS (Google Cloud Storage) とシームレスに連携し、クラウド上のリソースを最大限に活用した作画制作を実現するハイエンドなツールキットです。

---

## ✨ コア・コンセプト (Core Concepts)

* **🧬 Character DNA System**: `domain.Character` に定義したSeed値と視覚特徴をプロンプトへ動的に注入。全ページを通じてキャラクターの外見を一貫させることが可能です。
* **📑 Auto-Chunk Pagination**: パネル数が上限を超えると自動でページをスライス。AIの描画限界を回避し、複数枚構成の漫画を安定して生成します。
* **📖 Script-to-Manga Generator**: Markdown等のソースを `parser` が解析。`net/url` を基盤とした解析により、クラウド上の相対パスを正確な絶対URL（`gs://...`）へ解決します。
* **📐 Dynamic Layout Director**: ページごとに「主役パネル（Big Panel）」を動的に決定。単調なコマ割りを防ぎ、ドラマチックな演出を自動生成します。
* **🛡️ Robust Path Resolution**: `url.ResolveReference` を採用。`../` などの相対参照も標準プロトコルに従って安全に解決し、GCSやHTTPのスキームを破壊することなくアセットを特定します。

---

## 🏗 システムスタック (System Stack)

| レイヤー | 技術 / ライブラリ | 役割 |
| --- | --- | --- |
| **Intelligence** | **Gemini 3.0 Flash** | 伝説の編集者プロンプトによるネーム構成 |
| **Artistic** | **Nano Banana** | DNA注入と空間構成プロンプトによる一括作画 |
| **Resilience** | **go-cache** | 参照画像のTTL管理（30分）による高速化 |
| **Concurrency** | `x/time/rate` | 安定したAPIクォータ遵守 |
| **Drawing Engine** | `shouni/gemini-image-kit` | Image-to-Image / Multi-Reference 描画コア |
| **I/O Factory** | `shouni/go-remote-io` | GCS/Localの透過的なアクセス |
| **Web Extract** | `shouni/go-web-exact` | Webページからのセマンティックなコンテンツ抽出。 |

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
    participant Gen as manga-kit.MangaGenerator
    participant Kit_Gen as gemini-image-kit.GeminiGenerator
    participant Kit_Core as gemini-image-kit.GeminiImageCore
    participant R_IO as remoteio.InputReader (GCS)
    participant HTTP as HTTP Client (Web)
    participant API as Gemini API (Nano Banana)

    Note over APP, Kit_Gen: 1. 初期化フェーズ
    APP->>Kit_Core: NewGeminiImageCore(reader, client, cache)
    APP->>Kit_Gen: NewGeminiGenerator(core, apiClient, model)

    Note over APP, API: 2. 生成フェーズ (Execution)
    APP->>Gen: ExecuteMangaPages
    Gen->>Kit_Gen: GenerateMangaPage(req)

    loop 各 ReferenceURL の処理 (Core Pipeline)
        Kit_Gen->>Kit_Core: prepareImagePart(url)
        
        rect rgb(240, 240, 240)
            Note over Kit_Core: 【Security】 IsSafeURL (SSRF Check)
        end

        Kit_Core->>Kit_Core: キャッシュ確認
        
        alt キャッシュなし
            Note over Kit_Core, HTTP: スキームに応じて取得先を分岐
            critical URL Scheme check
                option gs://
                    Kit_Core->>R_IO: Open / ReadAll (GCS)
                    R_IO-->>Kit_Core: []byte
                option http(s)://
                    Kit_Core->>HTTP: FetchBytes (Web)
                    HTTP-->>Kit_Core: []byte
            end

            rect rgb(230, 245, 255)
                Note over Kit_Core: 【Optimization】 imgutil.CompressToJPEG
            end
            Kit_Core->>Kit_Core: キャッシュ保存
        end
        
        Kit_Core-->>Kit_Gen: genai.Part (InlineData)
    end

    Note over Kit_Gen, API: 3. AI推論 (Inference)
    Kit_Gen->>API: GenerateContent (Part + Prompt + Seed)
    API-->>Kit_Gen: Candidate Image Data
    Kit_Gen->>Kit_Core: parseToResponse (抽出・正規化)
    Kit_Gen-->>Gen: domain.ImageResponse
    Gen-->>APP: 生成完了通知

```

---

## 🤝 依存関係 (Dependencies)

* [shouni/gemini-image-kit](https://github.com/shouni/gemini-image-kit) - Gemini 画像作成抽象化
* [shouni/go-remote-io](https://github.com/shouni/go-remote-io) - GCS、およびローカルファイルシステムへの I/O 操作を統一化

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
