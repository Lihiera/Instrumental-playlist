# Project Metadata

## Objective

Apple Musicの既存プレイリストを解析し、インストゥルメンタル曲のみを抽出した新しいライブラリプレイリストを作成するWeb APIアプリケーションを構築する。

## Success Criteria

- Go製HTTPサーバーとして起動できる。
- `.env`からDeveloper Token、HTTP待受アドレス、Storefrontを読み込める。
- Apple MusicユーザーのMusic User TokenをRESTリクエストのヘッダーで受け取れる。
- REST API経由でプレイリスト一覧を取得できる。
- REST API経由でプレイリストの作成、削除、曲追加、曲削除ができる。
- REST API経由でApple Musicカタログの楽曲検索ができる。
- 既存プレイリストをdry-runで評価し、採用/除外理由をJSONで返せる。
- 既存プレイリストからインストゥルメンタル候補のみを新規プレイリストへ追加できる。

## Non-Goals for v1

- フロントエンドUIまたはGUI。
- CLIからのプレイリスト操作。
- 既存プレイリストの破壊的な上書き変換。
- 歌詞API、音源解析、機械学習分類器による判定。
- 複数ユーザー向けの永続セッション管理。

## Technical Direction

- Language: Go
- Interface: REST API over HTTP
- Repository default branch: `main`
- Developer Token source: `.env`
- Music User Token source: request header, for example `X-Music-User-Token`
- Conversion default: create a new playlist
- Instrumental detection: rule-based heuristic scoring

## Primary Risks

- Apple Music APIの権限・地域差・ライブラリ曲とカタログ曲のID差異。
- Music User Tokenをクライアントから安全に受け渡すためのAPI境界。
- Developer Tokenをログや設定確認APIへ漏らさないための秘匿設計。
- ヒューリスティック判定による誤採用/誤除外。
- 大きなプレイリストでのAPI制限、ページネーション、部分失敗。
