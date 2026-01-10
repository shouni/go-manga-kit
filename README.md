# 🎨 Go Manga Kit

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/go-manga-kit)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-manga-kit)](https://github.com/shouni/go-manga-kit/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 概要 (About) - 自動ページ分割対応・作画生成Workflows

**Go Manga Kit** は、非構造化ドキュメントを解析し、AIによる**キャラクターDNAの一貫性を維持した作画**を行うためのエンジニア向けライブラリです。

[Gemini Image Kit](https://github.com/shouni/gemini-image-kit) を描画コアに採用。独自の**オート・チャンク・システム**により、1ページあたり最大6パネルで自動分割を行います。複数枚の作画として出力できるハイエンドなツールキットです。

---

## ✨ コア・コンセプト (Core Concepts)

* **🧬 Character DNA System**: `domain.Character` に定義したSeed値と視覚特徴をプロンプトへ動的に注入します。全ページを通じてキャラクターの外見を一貫させることが可能です。
* **📑 Auto-Chunk Pagination**: パネル数が上限を超えると自動でページをスライスします。AIの描画限界を回避し、複数枚構成の漫画を安定して生成します。
* **📖 Script-to-Manga Generator**: Markdown等のソースを `parser` が解析し、演出指示を含む構造化データへ変換します。これを `generator` が受け取り、一括で作画を行う一気通貫の設計です。
* **📐 Dynamic Layout Director**: ページごとに「主役パネル（Big Panel）」を動的に決定します。単調なコマ割りを防ぎ、ドラマチックな演出を自動生成します。
* **🛡️ Resilience & Rate Control**: **30s/req (2 RPM)** の厳格なレートリミット制御と、参照画像のTTL付きキャッシュにより、APIクォータを尊重しつつ安定した作画を継続します。

---

## 🏗 システムスタック

| レイヤー | 技術 / ライブラリ | 役割 |
| --- | --- | --- |
| **Intelligence** | **Gemini 3.0 Flash** | 伝説の編集者プロンプトによるネーム構成 |
| **Artistic** | **Nano Banana** | DNA注入と空間構成プロンプトによる一括作画 |
| **Resilience** | **go-cache** | 参照画像のTTL管理（30分）による高速化 |
| **Concurrency** | `x/time/rate` | 安定したAPIクォータ遵守 |
| **I/O Factory** | `shouni/go-remote-io` | GCS/Localの透過的なアクセス |
| **Drawing Engine** | `shouni/gemini-image-kit` | Image-to-Image / Multi-Reference 描画コア |

---

## 🎨 5つのワークフロー (Workflows)

制作プロセスに応じて、以下の5つの機能をWeb UIから使い分けられるのだ。

| 画面 (Command) | 役割 | 主な出力 |
| --- | --- | --- |
| **Design** | DNA抽出。設定画を生成し、**固定用のSeed値を特定**する。 | Design Image, **Final Seed (via Slack)** |
| **Generate** | 一括生成。解析から全ページのパブリッシュまで一気通貫。 | HTML, Images, MD |
| **Script** | 台本生成。AIによる構成案（JSON）のみを出力。 | JSON (Script) |
| **Image** | パネル作画。既存の台本から画像とHTMLを生成。 | Images, HTML, MD |
| **Story** | 最終錬成。プロット（Markdown）から漫画構成案を生成。 | Manga Structure (JSON) |

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
    ├── prompts/     # プロンプト構築
    ├── publisher/   # 成果物出力 (publisher.go)
    ├── runner/      # ワークフローの実行処理
    └── workflow/    # ワークフローの管理
```

---

## 🏗️ 作画生成システム 全体シーケンスフロー

```mermaid
sequenceDiagram
    participant CLI as CLI Application
    participant Gen as manga-kit.MangaGenerator (Page/Group)
    participant Kit_Gen as gemini-image-kit.GeminiGenerator
    participant Kit_Core as gemini-image-kit.GeminiImageCore
    participant Kit_Util as gemini-image-kit.imgutil (Compressor)
    participant API as Gemini API (File/Model)

    Note over CLI, Gen: 1. 初期化フェーズ (Setup)
    CLI->>Gen: NewMangaGenerator
    Gen->>Kit_Core: NewGeminiImageCore(httpClient, cache)
    Gen->>Kit_Gen: NewGeminiGenerator(core, apiClient, model)

    Note over CLI, Kit_Util: 2. 生成フェーズ (Execution)
    CLI->>Gen: ExecuteMangaPages (または ExecutePanelGroup)
    
    rect
        Note over Gen, Kit_Gen: manga-kit は core-kit の抽象化インターフェースを利用
        Gen->>Kit_Gen: GenerateMangaPage(req)
    end

    loop 各 ReferenceURL の処理 (Core Pipeline)
        Kit_Gen->>Kit_Core: GetReferenceImage(url)
        
        rect
            Note over Kit_Core: 【Security: SSRF対策】
            Kit_Core->>Kit_Core: isSafeURL (DNS/IPバリデーション)
        end
        
        Kit_Core->>Kit_Core: キャッシュ確認
        alt キャッシュなし / 新規取得
            Kit_Core->>Kit_Core: 外部から画像ダウンロード
            
            Note over Kit_Core, Kit_Util: 取得から最適化までを Core 内で完結
            Kit_Core->>Kit_Util: 画像の最適化 (JPEG圧縮)
            Kit_Util-->>Kit_Core: 最適化済みバイナリ
        end
        Kit_Core-->>Kit_Gen: 最終画像データ
        
        Kit_Gen->>API: File API へのアップロード (Multipart)
    end

    Note over Kit_Gen, API: 3. AI推論 (Inference)
    Kit_Gen->>API: GenerateContent (int32 Seed / FileData)
    API-->>Kit_Gen: 生成画像データ (PNG)
    Kit_Gen-->>Gen: domain.ImageResponse
    Gen-->>CLI: 生成完了通知

```

---

## 🤝 依存関係 (Dependencies)

* [shouni/gemini-image-kit](https://github.com/shouni/gemini-image-kit) - Gemini 画像作成抽象化
* [shouni/go-remote-io](https://github.com/shouni/go-remote-io) - GCS、およびローカルファイルシステムへの I/O 操作を統一化

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
