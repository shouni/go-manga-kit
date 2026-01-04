package main

import (
	"github.com/shouni/go-manga-kit/cmd"
)

// main はアプリケーションの唯一のエントリーポイントなのだ！
// コマンドライン引数の解析と実行はすべて cmd パッケージに委ねるのだよ。
func main() {
	cmd.Execute()
}
