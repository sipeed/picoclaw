#!/usr/bin/env python3
"""threads — Threads CLI for picoclaw.

Thin wrapper around the Threads API (graph.threads.net).
Installed to /usr/local/bin/threads inside the Docker image
so the agent can call `threads post`, `threads whoami`, etc.

Auth: Set THREADS_ACCESS_TOKEN env var or pass --token.
"""

import argparse
import json
import os
import sys
import time
import urllib.request
import urllib.parse
import urllib.error

API_BASE = "https://graph.threads.net/v1.0"


# ---------------------------------------------------------------------------
# HTTP helpers (stdlib only — no extra deps needed)
# ---------------------------------------------------------------------------

def _get_token(args) -> str:
    token = getattr(args, "token", None) or os.environ.get("THREADS_ACCESS_TOKEN")
    if not token:
        print("No access token. Set THREADS_ACCESS_TOKEN or pass --token.", file=sys.stderr)
        sys.exit(1)
    return token


def _api_get(path: str, params: dict) -> dict:
    qs = urllib.parse.urlencode(params)
    url = f"{API_BASE}/{path}?{qs}"
    req = urllib.request.Request(url, method="GET")
    try:
        with urllib.request.urlopen(req) as resp:
            return json.loads(resp.read())
    except urllib.error.HTTPError as e:
        body = e.read().decode()
        print(f"API error {e.code}: {body}", file=sys.stderr)
        sys.exit(1)


def _api_post(path: str, params: dict) -> dict:
    url = f"{API_BASE}/{path}"
    data = urllib.parse.urlencode(params).encode()
    req = urllib.request.Request(url, data=data, method="POST")
    try:
        with urllib.request.urlopen(req) as resp:
            return json.loads(resp.read())
    except urllib.error.HTTPError as e:
        body = e.read().decode()
        print(f"API error {e.code}: {body}", file=sys.stderr)
        sys.exit(1)


def _get_user_id(token: str) -> str:
    data = _api_get("me", {"fields": "id", "access_token": token})
    return data["id"]


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

def cmd_whoami(args) -> None:
    token = _get_token(args)
    data = _api_get("me", {
        "fields": "id,username,threads_profile_picture_url,threads_biography",
        "access_token": token,
    })
    if args.json_output:
        print(json.dumps(data, indent=2))
    else:
        print(f"User ID:  {data.get('id', '?')}")
        print(f"Username: @{data.get('username', '?')}")
        if data.get("threads_biography"):
            print(f"Bio:      {data['threads_biography']}")


def cmd_post(args) -> None:
    token = _get_token(args)
    user_id = _get_user_id(token)
    text = args.text

    if args.dry_run:
        print(f"[DRY RUN] Would post ({len(text)} chars):")
        print(text)
        return

    create_params = {
        "media_type": "TEXT",
        "text": text,
        "access_token": token,
    }

    # Attach image if provided
    if args.image:
        create_params["media_type"] = "IMAGE"
        create_params["image_url"] = args.image

    # Step 1: Create media container
    result = _api_post(f"{user_id}/threads", create_params)
    container_id = result.get("id")
    if not container_id:
        print(f"Failed to create container: {result}", file=sys.stderr)
        sys.exit(1)

    # Step 2: Wait for container to be ready (poll status)
    _wait_for_container(token, container_id)

    # Step 3: Publish
    publish_result = _api_post(f"{user_id}/threads_publish", {
        "creation_id": container_id,
        "access_token": token,
    })
    post_id = publish_result.get("id")
    print(f"Posted: {post_id}")


def cmd_post_image_url(args) -> None:
    """Post with an image URL (Threads requires publicly accessible URLs)."""
    token = _get_token(args)
    user_id = _get_user_id(token)

    if args.dry_run:
        print(f"[DRY RUN] Would post image ({len(args.text)} chars):")
        print(f"  Text: {args.text}")
        print(f"  Image: {args.image_url}")
        return

    result = _api_post(f"{user_id}/threads", {
        "media_type": "IMAGE",
        "text": args.text,
        "image_url": args.image_url,
        "access_token": token,
    })
    container_id = result.get("id")
    if not container_id:
        print(f"Failed to create container: {result}", file=sys.stderr)
        sys.exit(1)

    _wait_for_container(token, container_id)

    publish_result = _api_post(f"{user_id}/threads_publish", {
        "creation_id": container_id,
        "access_token": token,
    })
    print(f"Posted: {publish_result.get('id')}")


def cmd_post_video_url(args) -> None:
    """Post with a video URL (must be publicly accessible, mp4)."""
    token = _get_token(args)
    user_id = _get_user_id(token)

    if args.dry_run:
        print(f"[DRY RUN] Would post video ({len(args.text)} chars):")
        print(f"  Text: {args.text}")
        print(f"  Video: {args.video_url}")
        return

    result = _api_post(f"{user_id}/threads", {
        "media_type": "VIDEO",
        "text": args.text,
        "video_url": args.video_url,
        "access_token": token,
    })
    container_id = result.get("id")
    if not container_id:
        print(f"Failed to create container: {result}", file=sys.stderr)
        sys.exit(1)

    # Videos take longer to process
    _wait_for_container(token, container_id, max_wait=120)

    publish_result = _api_post(f"{user_id}/threads_publish", {
        "creation_id": container_id,
        "access_token": token,
    })
    print(f"Posted: {publish_result.get('id')}")


def cmd_reply(args) -> None:
    token = _get_token(args)
    user_id = _get_user_id(token)

    result = _api_post(f"{user_id}/threads", {
        "media_type": "TEXT",
        "text": args.text,
        "reply_to_id": args.post_id,
        "access_token": token,
    })
    container_id = result.get("id")
    if not container_id:
        print(f"Failed to create reply container: {result}", file=sys.stderr)
        sys.exit(1)

    _wait_for_container(token, container_id)

    publish_result = _api_post(f"{user_id}/threads_publish", {
        "creation_id": container_id,
        "access_token": token,
    })
    print(f"Replied: {publish_result.get('id')}")


def cmd_profile(args) -> None:
    token = _get_token(args)
    data = _api_get("me", {
        "fields": "id,username,threads_profile_picture_url,threads_biography",
        "access_token": token,
    })
    if args.json_output:
        print(json.dumps(data, indent=2))
    else:
        print(f"@{data.get('username', '?')}")
        if data.get("threads_biography"):
            print(f"  Bio: {data['threads_biography']}")
        print(f"  ID:  {data.get('id', '?')}")


def cmd_insights(args) -> None:
    """Get profile-level insights (follower count, views)."""
    token = _get_token(args)
    user_id = _get_user_id(token)
    data = _api_get(f"{user_id}/threads_insights", {
        "metric": "views,likes,replies,reposts,quotes,followers_count",
        "access_token": token,
    })
    if args.json_output:
        print(json.dumps(data, indent=2))
    else:
        for item in data.get("data", []):
            name = item.get("name", "?")
            values = item.get("values", [{}])
            val = values[0].get("value", "?") if values else "?"
            print(f"  {name}: {val}")


def cmd_posts(args) -> None:
    """List recent posts."""
    token = _get_token(args)
    user_id = _get_user_id(token)
    data = _api_get(f"{user_id}/threads", {
        "fields": "id,text,timestamp,media_type,shortcode,permalink",
        "limit": args.n,
        "access_token": token,
    })
    if args.json_output:
        print(json.dumps(data, indent=2))
    else:
        for post in data.get("data", []):
            ts = post.get("timestamp", "")
            text = post.get("text", "")
            pid = post.get("id", "?")
            permalink = post.get("permalink", "")
            print(f"  [{ts}] {pid}")
            for line in text.splitlines():
                print(f"    {line}")
            if permalink:
                print(f"    {permalink}")
            print()


def cmd_post_insights(args) -> None:
    """Get insights for a specific post."""
    token = _get_token(args)
    data = _api_get(f"{args.post_id}/insights", {
        "metric": "views,likes,replies,reposts,quotes",
        "access_token": token,
    })
    if args.json_output:
        print(json.dumps(data, indent=2))
    else:
        print(f"Insights for {args.post_id}:")
        for item in data.get("data", []):
            name = item.get("name", "?")
            values = item.get("values", [{}])
            val = values[0].get("value", "?") if values else "?"
            print(f"  {name}: {val}")


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _wait_for_container(token: str, container_id: str, max_wait: int = 30) -> None:
    """Poll container status until FINISHED or timeout."""
    for _ in range(max_wait):
        status = _api_get(container_id, {
            "fields": "status,error_message",
            "access_token": token,
        })
        s = status.get("status")
        if s == "FINISHED":
            return
        if s == "ERROR":
            print(f"Container error: {status.get('error_message', 'unknown')}", file=sys.stderr)
            sys.exit(1)
        time.sleep(1)
    print(f"Timeout waiting for container {container_id} (status: {s})", file=sys.stderr)
    sys.exit(1)


# ---------------------------------------------------------------------------
# Argument parser
# ---------------------------------------------------------------------------

def main() -> None:
    parser = argparse.ArgumentParser(prog="threads", description="Threads CLI for picoclaw")
    parser.add_argument("--json", dest="json_output", action="store_true", help="JSON output")
    parser.add_argument("--token", help="Access token (overrides THREADS_ACCESS_TOKEN env var)")
    sub = parser.add_subparsers(dest="command")

    # whoami
    sub.add_parser("whoami", help="Show current user info")

    # post
    p = sub.add_parser("post", help="Create a text post")
    p.add_argument("text", help="Post text")
    p.add_argument("--image", help="Public image URL to attach")
    p.add_argument("--dry-run", action="store_true", help="Preview without posting")

    # post-image
    p = sub.add_parser("post-image", help="Post with a public image URL")
    p.add_argument("text", help="Post text")
    p.add_argument("image_url", help="Public image URL")
    p.add_argument("--dry-run", action="store_true", help="Preview without posting")

    # post-video
    p = sub.add_parser("post-video", help="Post with a public video URL")
    p.add_argument("text", help="Post text")
    p.add_argument("video_url", help="Public video URL (mp4)")
    p.add_argument("--dry-run", action="store_true", help="Preview without posting")

    # reply
    p = sub.add_parser("reply", help="Reply to a post")
    p.add_argument("post_id", help="Post ID to reply to")
    p.add_argument("text", help="Reply text")

    # profile
    sub.add_parser("profile", help="View your profile")

    # insights
    sub.add_parser("insights", help="View profile insights (followers, views)")

    # posts
    p = sub.add_parser("posts", help="List recent posts")
    p.add_argument("-n", type=int, default=10, help="Number of posts")

    # post-insights
    p = sub.add_parser("post-insights", help="Get insights for a post")
    p.add_argument("post_id", help="Post ID")

    args = parser.parse_args()
    if not args.command:
        parser.print_help()
        sys.exit(1)

    handlers = {
        "whoami": cmd_whoami,
        "post": cmd_post,
        "post-image": cmd_post_image_url,
        "post-video": cmd_post_video_url,
        "reply": cmd_reply,
        "profile": cmd_profile,
        "insights": cmd_insights,
        "posts": cmd_posts,
        "post-insights": cmd_post_insights,
    }

    try:
        handlers[args.command](args)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
