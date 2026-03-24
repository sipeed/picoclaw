#!/usr/bin/env python3
"""Fetch all tracks from a Spotify playlist and write them to JSON.

Usage:
  spotify-playlist.py [--playlist URL_OR_ID] [--output PATH]

Requires SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET env vars.
Get them at https://developer.spotify.com/dashboard (create a free app).
Add http://127.0.0.1:8888/callback as a Redirect URI in your app settings.

On first run, you'll be prompted to visit a URL and paste back the redirect.
The token is cached at ~/.spotify-token.json for subsequent runs.

Example:
  export SPOTIFY_CLIENT_ID="abc123"
  export SPOTIFY_CLIENT_SECRET="xyz789"
  python3 spotify-playlist.py \
    --playlist 530uTl6cSboMQWNRJSJrjz \
    --output /root/.picoclaw/workspace/data/playlist.json
"""

import argparse
import base64
import json
import os
import re
import secrets
import sys
import webbrowser
from urllib.request import Request, urlopen
from urllib.parse import urlencode, urlparse, parse_qs

DEFAULT_PLAYLIST = "530uTl6cSboMQWNRJSJrjz"
DEFAULT_OUTPUT = "playlist.json"
TOKEN_URL = "https://accounts.spotify.com/api/token"
AUTH_URL = "https://accounts.spotify.com/authorize"
REDIRECT_URI = "http://127.0.0.1:8888/callback"
SCOPE = "playlist-read-private playlist-read-collaborative"
API_BASE = "https://api.spotify.com/v1"
TOKEN_CACHE = os.path.join(os.path.expanduser("~"), ".spotify-token.json")


def _load_cached_token() -> dict | None:
    """Load cached token if it exists."""
    if os.path.exists(TOKEN_CACHE):
        with open(TOKEN_CACHE) as f:
            return json.load(f)
    return None


def _save_token(token_data: dict) -> None:
    """Cache the token to disk."""
    with open(TOKEN_CACHE, "w") as f:
        json.dump(token_data, f)


def _refresh_token(client_id: str, client_secret: str, refresh_token: str) -> dict:
    """Refresh an expired access token."""
    creds = base64.b64encode(f"{client_id}:{client_secret}".encode()).decode()
    data = urlencode({
        "grant_type": "refresh_token",
        "refresh_token": refresh_token,
    }).encode()
    req = Request(TOKEN_URL, data=data, headers={
        "Authorization": f"Basic {creds}",
        "Content-Type": "application/x-www-form-urlencoded",
    })
    with urlopen(req) as resp:
        result = json.loads(resp.read())
    # Spotify may or may not return a new refresh token
    if "refresh_token" not in result:
        result["refresh_token"] = refresh_token
    _save_token(result)
    return result


def _authorize(client_id: str, client_secret: str) -> dict:
    """Authorization Code flow via manual URL paste (no local server needed)."""
    state = secrets.token_urlsafe(16)

    params = urlencode({
        "client_id": client_id,
        "response_type": "code",
        "redirect_uri": REDIRECT_URI,
        "scope": SCOPE,
        "state": state,
    })
    url = f"{AUTH_URL}?{params}"
    print("1. Visit this URL and authorize the app:\n")
    print(f"   {url}\n")
    webbrowser.open(url)
    print("2. After authorizing, you'll be redirected to a page that won't load.")
    print("   Copy the FULL URL from your browser's address bar and paste it here.\n")

    redirect_url = input("Paste redirect URL: ").strip()

    qs = parse_qs(urlparse(redirect_url).query)
    if qs.get("state", [None])[0] != state:
        print("Error: State mismatch — possible CSRF. Try again.", file=sys.stderr)
        sys.exit(1)
    if "error" in qs:
        print(f"Authorization denied: {qs['error'][0]}", file=sys.stderr)
        sys.exit(1)

    auth_code = qs.get("code", [None])[0]
    if not auth_code:
        print("Error: No authorization code found in URL.", file=sys.stderr)
        sys.exit(1)

    # Exchange code for token
    creds = base64.b64encode(f"{client_id}:{client_secret}".encode()).decode()
    data = urlencode({
        "grant_type": "authorization_code",
        "code": auth_code,
        "redirect_uri": REDIRECT_URI,
    }).encode()
    req = Request(TOKEN_URL, data=data, headers={
        "Authorization": f"Basic {creds}",
        "Content-Type": "application/x-www-form-urlencoded",
    })
    with urlopen(req) as resp:
        token_data = json.loads(resp.read())

    _save_token(token_data)
    return token_data


def get_token(client_id: str, client_secret: str) -> str:
    """Get a valid access token, refreshing or re-authorizing as needed."""
    cached = _load_cached_token()
    if cached:
        # Try using the cached token
        try:
            req = Request(f"{API_BASE}/me", headers={
                "Authorization": f"Bearer {cached['access_token']}"
            })
            urlopen(req)
            return cached["access_token"]
        except Exception:
            pass
        # Try refreshing
        if cached.get("refresh_token"):
            try:
                refreshed = _refresh_token(client_id, client_secret, cached["refresh_token"])
                return refreshed["access_token"]
            except Exception:
                pass
    # Full re-authorization
    token_data = _authorize(client_id, client_secret)
    return token_data["access_token"]


def _api_get(token: str, url: str) -> dict:
    """GET a Spotify API endpoint."""
    req = Request(url, headers={"Authorization": f"Bearer {token}"})
    with urlopen(req) as resp:
        return json.loads(resp.read())


def _extract_playlist_id(url_or_id: str) -> str:
    """Extract playlist ID from a URL or return as-is."""
    m = re.search(r"playlist/([a-zA-Z0-9]+)", url_or_id)
    return m.group(1) if m else url_or_id


def fetch_playlist(token: str, playlist_id: str) -> dict:
    """Fetch playlist metadata and all tracks (handles pagination)."""
    info = _api_get(token, f"{API_BASE}/playlists/{playlist_id}?fields=name,description,external_urls")

    tracks = []
    url = f"{API_BASE}/playlists/{playlist_id}/items?limit=50"

    while url:
        page = _api_get(token, url)
        for item in page.get("items", []):
            t = item.get("item") or item.get("track")
            if not t or not t.get("id") or t.get("type") == "episode":
                continue  # skip local files / unavailable tracks

            album = t.get("album") or {}
            images = album.get("images") or []
            art_url = images[0]["url"] if images else None

            tracks.append({
                "id": t["id"],
                "name": t["name"],
                "artists": [a["name"] for a in t.get("artists", [])],
                "album": album.get("name", ""),
                "release_date": album.get("release_date", ""),
                "duration_ms": t.get("duration_ms", 0),
                "spotify_url": (t.get("external_urls") or {}).get("spotify", ""),
                "preview_url": t.get("preview_url"),
                "album_art_url": art_url,
            })
        url = page.get("next")

    return {
        "playlist_name": info.get("name", ""),
        "playlist_description": info.get("description", ""),
        "playlist_url": (info.get("external_urls") or {}).get("spotify", ""),
        "total_tracks": len(tracks),
        "tracks": tracks,
    }


def main():
    parser = argparse.ArgumentParser(description="Fetch Spotify playlist tracks to JSON")
    parser.add_argument("--playlist", default=DEFAULT_PLAYLIST,
                        help="Spotify playlist URL or ID (default: The Primer playlist)")
    parser.add_argument("--output", "-o", default=DEFAULT_OUTPUT,
                        help="Output JSON file path (default: playlist.json)")
    args = parser.parse_args()

    client_id = os.environ.get("SPOTIFY_CLIENT_ID", "")
    client_secret = os.environ.get("SPOTIFY_CLIENT_SECRET", "")
    if not client_id or not client_secret:
        print("Error: Set SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET env vars.", file=sys.stderr)
        print("Get them at https://developer.spotify.com/dashboard", file=sys.stderr)
        sys.exit(1)

    playlist_id = _extract_playlist_id(args.playlist)
    print(f"Fetching playlist {playlist_id}...")

    token = get_token(client_id, client_secret)
    data = fetch_playlist(token, playlist_id)

    os.makedirs(os.path.dirname(os.path.abspath(args.output)), exist_ok=True)
    with open(args.output, "w", encoding="utf-8") as f:
        json.dump(data, f, indent=2, ensure_ascii=False)

    print(f"Wrote {data['total_tracks']} tracks to {args.output}")
    print(f"Playlist: {data['playlist_name']}")


if __name__ == "__main__":
    main()
