# Project Metadata

## Objective

Spotifyの既存プレイリストを解析し、インストゥルメンタル曲のみを抽出した新しいSpotifyプレイリストを作成するWeb APIアプリケーションを構築する。

## Success Criteria

- Go製HTTPサーバーとして起動できる。
- `.env`からSpotify Client ID、Client Secret、Redirect URI、HTTP待受アドレスを読み込める。
- SpotifyユーザーのアクセストークンをRESTリクエストの`Authorization`ヘッダーで受け取れる。
- REST API経由でSpotifyプレイリスト一覧を取得できる。
- REST API経由でSpotifyプレイリストの作成、曲追加、曲削除ができる。
- REST API経由でSpotifyカタログの楽曲検索ができる。
- 既存プレイリストをdry-runで評価し、採用/除外理由をJSONで返せる。
- 既存プレイリストからインストゥルメンタル候補のみを新規Spotifyプレイリストへ追加できる。

## Non-Goals for v1

- フロントエンドUIまたはGUI。
- CLIからのプレイリスト操作。
- 既存プレイリストの破壊的な上書き変換。
- 歌詞API、音源解析、機械学習分類器による判定。
- 複数ユーザー向けの永続セッション管理。
- Spotify OAuthログイン画面の完全実装。v1ではアクセストークンをリクエストヘッダーで受け取る。

## Technical Direction

- Language: Go
- Interface: REST API over HTTP
- Repository default branch: `main`
- Spotify app settings source: `.env`
- Spotify user access token source: `Authorization: Bearer <spotify_access_token>`
- Conversion default: create a new playlist
- Instrumental detection: rule-based heuristic scoring

## Primary Risks

- Spotify OAuthスコープ不足によるプレイリスト取得・編集失敗。
- Spotifyアクセストークンをクライアントから安全に受け渡すためのAPI境界。
- Spotify Client Secretやアクセストークンをログや設定確認APIへ漏らさないための秘匿設計。
- Spotify Web APIの開発モード、クォータ、レート制限。
- ヒューリスティック判定による誤採用/誤除外。
- 大きなプレイリストでのページネーション、100件単位の曲追加制限、部分失敗。
