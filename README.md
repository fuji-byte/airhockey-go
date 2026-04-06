https://github.com/fuji-byte/RtsGame/tree/feature/ingame
参照

最優先実装機能（優先度：高）
ゲーム中の通知（フロント実装も）
ルーム待機画面からの退出。フロントから退出時に送信されたものをもとに削除処理を行う

error 報告
なし

実装要素（優先度：中）
三細胞以上でサイクルできてしまう
room キュー、ランダム参加など
observer の仕様 ゲーム開始に入れるか、ルーム内に通知するか
cell 線の双方向　おもにフロント
更新した部分だけブロードキャストする
観戦者系（観戦者がいたらゲームを終了しないなど）
ルーム時間終了後の処理（チャットができるようにして再選もかのう）
cell 座標の修正（フロントとバックエンド両方）

将来実装要素（優先度：低）
cell power を基に、攻撃をする。（０以下にはならない）
cell heal を基に、回復（rank を上昇）させる
ルームネーム、プレイヤーネームをつけれるようにする
ルームに入るためにキーを必要とする
sync あたりの最適化も
ランキング
フレンド機能
guest でもトークンを生成してユーザー認証を行う
savelog function の実装

<!-- #Redis の利用
redis を入れていないなら、redis を導入する
sudo apt update
sudo apt install redis-server
sudo service redis-server start -->

<!-- wsl の redis の実行
sudo service redis-server start -->

golang 実行
air

<!-- http://localhost:8080/login?user_id=testuser123 -->

<!-- Redis クライアント確認(wsl)
redis-cli
keys \*
get session:クライアント id -->
