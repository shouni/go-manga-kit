# 🎨 Go Manga Kit

## 🚀 概要 (About) - テキストから「マンガ」を紡ぐ、AIオーケストレーション・キット

**Go Manga Kit** は、AI（Gemini/Imagen）を用いたマンガ生成の複雑な工程を自動化し、構造化するためのGo言語向けライブラリなのだ。

[Gemini Image Kit](https://github.com/shouni/gemini-image-kit) を強力な描画エンジンとして活用し、Markdown形式の台本から、キャラクターの一貫性を保ったマルチパネル画像、さらには演出の効いたWebtoon（縦読みマンガ）HTMLまでを一気通貫で生成できるのだ。

---

## ✨ 主な特徴 (Features)

* **📖 Script-to-Manga Pipeline**: Markdown形式の台本を解析し、AIが理解可能な詳細な描写指示へ自動変換。
* **🧬 Character DNA System**: キャラクターの視覚的特徴（Visual Cues）とSeed値を管理し、全パネルを通して「同じ顔」を維持。
* **📐 Unified Prompt Engine**: 日本式の読み順（右から左）、コマ割り、マージン、ライティングをAIに叩き込む、高度なプロンプト構築ロジックを搭載。
* **🎭 Visual Director**: Webtoon特有の「セリフの交互配置」やレイアウト演出を自動計算。
* **🌐 Hybrid Publisher**: 生成されたコンテンツをMarkdown、HTML、画像としてローカルまたはGoogle Cloud Storageへ透過的に保存。

---

## 📂 プロジェクト構造 (Layout)

```text
pkg/
├── parser/      # Markdown台本のパースと構造化
├── generator/   # 統合プロンプトの構築とDNA注入
├── director/    # Webtoonレイアウトと演出の制御
├── publisher/   # 成果物の書き出しとフォーマット変換
└── pipeline/    # 全工程を繋ぐ実行制御（メインエントリ）

```

---

## 🛠️ 使い方 (Usage) - 3ステップでマンガ生成なのだ！

```go
import "github.com/shouni/go-manga-kit/pkg/pipeline"

// 1. パイプラインの初期化
p := pipeline.NewMangaPipeline(imageAdapter, outputWriter)

// 2. Markdown台本の準備
script := `# タイトル
## Panel:
- speaker: zundamon
- text: 餅を食べるのだ！
- action: 幸せそうに餅を頬張る様子`

// 3. 実行！画像からWebtoon HTMLまで自動生成なのだ
err := p.Execute(ctx, script, opts)

```

---

## 🤝 依存関係 (Dependencies)

* [shouni/gemini-image-kit](https://github.com/shouni/gemini-image-kit) - 高度な画像生成エンジン
* [shouni/go-remote-io](https://github.com/shouni/go-remote-io) - ストレージ抽象化
* [shouni/go-text-format](https://github.com/shouni/go-text-format) - Webtoon変換コア

---

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
