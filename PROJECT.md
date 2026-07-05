# Project Metadata

## Objective

Apple Musicの既存プレイリストを解析し、インストゥルメンタル曲のみを抽出した新しいライブラリプレイリストを作成するCLIバックエンドを構築する。

## Success Criteria

- Apple Musicユーザー認証を完了できる。
- ユーザーのライブラリプレイリスト一覧を取得できる。
- プレイリストの作成、削除、曲追加、曲削除ができる。
- Apple Musicカタログから楽曲検索ができる。
- 既存プレイリストをdry-runで評価し、採用/除外理由を表示できる。
- 既存プレイリストからインストゥルメンタル候補のみを新規プレイリストへ追加できる。

## Non-Goals for v1

- GUIまたはWebフロントエンド。
- 既存プレイリストの破壊的な上書き変換。
- 歌詞API、音源解析、機械学習分類器による判定。
- 複数ユーザー向けのサーバー運用。

## Technical Direction

- Language: Go
- Interface: CLI
- Repository default branch: `main`
- Authentication: localhost + MusicKit JS for Music User Token acquisition
- Conversion default: create a new playlist
- Instrumental detection: rule-based heuristic scoring

## Primary Risks

- Apple Music APIの権限・地域差・ライブラリ曲とカタログ曲のID差異。
- Music User Token取得フローのCLI環境での扱い。
- ヒューリスティック判定による誤採用/誤除外。
- 大きなプレイリストでのAPI制限、ページネーション、部分失敗。
