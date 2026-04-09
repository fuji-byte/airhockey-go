# airhockeyバックエンド

[参照](https://github.com/fuji-byte/RtsGame/tree/feature/ingame)

反省点
現時点で、リポジトリ層はオンメモリの設計なので、サービス層とリポジトリ層の境目が薄い。（リポジトリ層にサービス層のものまで入ってしまっている）
roomとplayerのmutexが共有になっている

issue
プレイ中にプレイヤーが切断したらプレイヤーが変わる（フロントの動くパドルが変わる）

最優先実装機能（優先度：高）
ゲーム中の通知（フロント実装も）

実装要素（優先度：中）
room キュー、ランダム参加など
observer の仕様 ゲーム開始に入れるか、ルーム内に通知するか
観戦者系（観戦者がいたらゲームを終了しないなど）
ルーム時間終了後の処理（チャットができるようにして再戦もかのう）

将来実装要素（優先度：低）
更新した部分だけブロードキャストする
ルームネーム、プレイヤーネームをつけれるようにする
ルームに入るためにキーを必要とする
sync あたりの最適化も
ランキング
フレンド機能
guest でもトークンを生成してユーザー認証を行う
savelog function の実装
cookieにより自動認証できるようにする

<!-- #Redis の利用
redis を入れていないなら、redis を導入する
sudo apt update
sudo apt install redis-server
sudo service redis-server start -->

<!-- wsl の redis の実行
sudo service redis-server start -->

golang 実行
go mod tidy
air init
air

<!-- http://localhost:8080/login?user_id=testuser123 -->

<!-- Redis クライアント確認(wsl)
redis-cli
keys \*
get session:クライアント id -->
