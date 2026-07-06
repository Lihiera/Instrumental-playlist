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

Spotifyのプレイリスト編集にはユーザー承認済みアクセストークンが必要です。Spotifyログイン後は、サーバーがプロセス内メモリに保存した最新のユーザーtokenをプレイリスト系REST APIで自動的に使います。クライアントから`Authorization: Bearer <spotify_access_token>`を明示した場合は、そのヘッダー値を優先します。Spotify Client Secretなどの秘密情報は`.env`から読み込み、APIレスポンスには返しません。

## ローカル実行

`.env.example`を参考に`.env`を作成します。

```env
HTTP_ADDR=:8080
SPOTIFY_CLIENT_ID=replace-with-spotify-client-id
SPOTIFY_CLIENT_SECRET=replace-with-spotify-client-secret
SPOTIFY_REDIRECT_URI=http://127.0.0.1:8080/oauth/spotify/callback
SPOTIFY_BASE_URL=https://api.spotify.com
SPOTIFY_ACCOUNTS_BASE_URL=https://accounts.spotify.com
```

サーバーを起動します。

```sh
go run ./cmd/instrumental-playlist
```

確認用エンドポイント:

```sh
curl http://localhost:8080/health
curl http://localhost:8080/v1/config
curl http://localhost:8080/v1/auth/status
curl -X POST http://localhost:8080/v1/auth/logout
```

`/v1/config`はSpotify Client Secretそのものを返さず、設定済みかどうかだけを返します。APIの詳細は[docs/api.md](docs/api.md)を参照してください。

Spotifyにログインする場合は、サーバー起動後にブラウザーで次のURLを開きます。

```text
http://localhost:8080/oauth/spotify/login
```

このエンドポイントはSpotify Accountsのログインページへ自動的にリダイレクトします。ログイン完了後、Spotifyは`SPOTIFY_REDIRECT_URI`に設定した`/oauth/spotify/callback`へ戻り、サーバーは取得したユーザーtokenをプロセス内メモリに保存して成功ページを表示します。成功ページにはtoken metadataだけを表示し、access tokenとrefresh token本体は表示しません。

ログイン状態は次のAPIで確認できます。

```sh
curl http://localhost:8080/v1/auth/status
```

このAPIはプロセス内メモリにユーザーtokenが保存されているかを返します。サーバーを再起動するとログイン状態は消えます。

ログイン済みであれば、ユーザーのプレイリスト操作APIは`Authorization`ヘッダーなしで実行できます。

```sh
curl http://localhost:8080/v1/playlists
curl http://localhost:8080/v1/playlists/{playlistID}/tracks
curl "http://localhost:8080/v1/search/tracks?term=piano"
```

`GET /v1/playlists/{playlistID}/tracks`はアプリ内の互換ルート名です。Spotifyへは現行の`GET /v1/playlists/{playlist_id}/items`を呼び出します。Spotify側の制限により、ログインユーザーがownerまたはcollaboratorではないplaylistは`403 Forbidden`になる場合があります。

ログアウトする場合は、プロセス内メモリのユーザーtokenを削除します。

```sh
curl -X POST http://localhost:8080/v1/auth/logout
```

## Spotify Web API連携

Phase 2ではSpotify Web APIクライアント基盤を実装済みです。内部クライアントはリクエストごとの`Authorization: Bearer <spotify_access_token>`、JSONレスポンス、ページネーション、レート制限時のリトライ、Spotifyエラー形式を扱います。

Spotify版では、プレイリスト一覧、検索、作成、曲追加・削除などの公開REST APIをPhase 3以降で追加します。必要なSpotifyスコープは主に`playlist-read-private`、`playlist-modify-public`、`playlist-modify-private`です。

Client Credentials Flowは、ログイン不要の公開プレイリスト検索でサーバー内部からSpotify Web APIを呼び出すために使います。ユーザーのプレイリスト読み書きには、ユーザーが認可したaccess tokenが必要です。ログイン済みの場合はプロセス内メモリのtokenを使い、`Authorization: Bearer <spotify_access_token>`が指定された場合はそのtokenを優先します。

Spotify Authorization Code Flowは`/oauth/spotify/login`と`/oauth/spotify/callback`で実装済みです。OAuth state、access token metadata、refresh token、期限情報は当面プロセス内メモリに保存し、Redis移行は主要機能の開発完了後に扱います。
