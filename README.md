# Instrumental Playlist

## プロジェクト概要

Instrumental Playlistは、Spotifyの既存プレイリストをもとに、歌声のないインストゥルメンタル曲だけで構成された新しいプレイリストを作成するためのWeb APIアプリケーションです。

このプロジェクトでは、Spotify上のプレイリスト取得、楽曲検索、プレイリスト作成・編集といった機能をREST APIとして提供し、元のプレイリストの雰囲気や楽曲の方向性をできるだけ保ちながら、作業や思考に向いたインストゥルメンタル版プレイリストを生成することを目指します。

## プロジェクトの動機

考え事や作業に集中したいとき、歌声のないインストゥルメンタル音楽は、思考を妨げずに音楽を楽しむための有効な選択肢になります。

Spotifyには、ポップス、クラシック、サウンドトラックなど多様なプレイリストがあり、ポップス楽曲のインストゥルメンタル版も存在します。一方で、特定のポップス・プレイリストに対応するインストゥルメンタル版プレイリストはほとんど用意されていません。

このプロジェクトは、好きなポップスのメロディーや雰囲気を楽しみながら、集中して考え事や作業をしたい人を支援することを目的としています。

## 開発方針

現在の実装はGo製HTTPサーバーとして起動するWeb APIアプリケーションです。プレイリスト操作や変換処理は、Spotify Web APIを内部で呼び出すREST APIとして提供します。

Spotifyのプレイリスト編集にはユーザー承認済みアクセストークンが必要です。今後追加するプレイリスト系REST APIでは、クライアントから`Authorization: Bearer <spotify_access_token>`を受け取ります。Spotify Client Secretなどの秘密情報は`.env`から読み込み、APIレスポンスには返しません。

## ローカル実行

`.env.example`を参考に`.env`を作成します。

```env
HTTP_ADDR=:8080
SPOTIFY_CLIENT_ID=replace-with-spotify-client-id
SPOTIFY_CLIENT_SECRET=replace-with-spotify-client-secret
SPOTIFY_REDIRECT_URI=http://localhost:8080/auth/spotify/callback
SPOTIFY_BASE_URL=https://api.spotify.com
```

サーバーを起動します。

```sh
go run ./cmd/instrumental-playlist
```

確認用エンドポイント:

```sh
curl http://localhost:8080/health
curl http://localhost:8080/v1/config
```

`/v1/config`はSpotify Client Secretそのものを返さず、設定済みかどうかだけを返します。APIの詳細は[docs/api.md](docs/api.md)を参照してください。

## Spotify Web API連携

Phase 2ではSpotify Web APIクライアント基盤を実装済みです。内部クライアントはリクエストごとの`Authorization: Bearer <spotify_access_token>`、JSONレスポンス、ページネーション、レート制限時のリトライ、Spotifyエラー形式を扱います。

Spotify版では、プレイリスト一覧、検索、作成、曲追加・削除などの公開REST APIをPhase 3以降で追加します。必要なSpotifyスコープは主に`playlist-read-private`、`playlist-modify-public`、`playlist-modify-private`です。
