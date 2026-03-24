#!/usr/bin/env python3
"""instagram — Instagram CLI for picoclaw.

Thin wrapper around the Instagram Graph API (graph.facebook.com).
Installed to /usr/local/bin/instagram inside the Docker image
so the agent can call `instagram post`, `instagram whoami`, etc.

Auth: Set INSTAGRAM_ACCESS_TOKEN env var or pass --token.
User ID: Set INSTAGRAM_USER_ID env var or pass --user-id.
"""

import argparse
import json
import os
import sys
import time
import urllib.request
import urllib.parse
import urllib.error

API_BASE = "https://graph.instagram.com/v21.0"


# ---------------------------------------------------------------------------
# HTTP helpers (stdlib only — no extra deps needed)
# ---------------------------------------------------------------------------

def _get_token(args) -> str:
    token = getattr(args, "token", None) or os.environ.get("INSTAGRAM_ACCESS_TOKEN")
    if not token:
        print("No access token. Set INSTAGRAM_ACCESS_TOKEN or pass --token.", file=sys.stderr)
        sys.exit(1)
    return token


def _get_user_id(args) -> str:
    uid = getattr(args, "user_id", None) or os.environ.get("INSTAGRAM_USER_ID")
    if not uid:
        print("No user ID. Set INSTAGRAM_USER_ID or pass --user-id.", file=sys.stderr)
        sys.exit(1)
    return uid


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


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

def cmd_whoami(args) -> None:
    token = _get_token(args)
    uid = _get_user_id(args)
    data = _api_get(uid, {
        "fields": "id,username,name,biography,followers_count,media_count",
        "access_token": token,
    })
    if args.json_output:
        print(json.dumps(data, indent=2))
    else:
        print(f"User ID:    {data.get('id', '?')}")
        print(f"Username:   @{data.get('username', '?')}")
        if data.get("name"):
            print(f"Name:       {data['name']}")
        if data.get("biography"):
            print(f"Bio:        {data['biography']}")
        if data.get("followers_count") is not None:
            print(f"Followers:  {data['followers_count']}")
        if data.get("media_count") is not None:
            print(f"Posts:      {data['media_count']}")


def cmd_post_image(args) -> None:
    """Post a single image (Instagram requires an image — no text-only posts)."""
    token = _get_token(args)
    uid = _get_user_id(args)

    if args.dry_run:
        print(f"[DRY RUN] Would post image:")
        print(f"  Caption: {args.caption}")
        print(f"  Image:   {args.image_url}")
        return

    # Step 1: Create media container
    create_params = {
        "image_url": args.image_url,
        "caption": args.caption,
        "access_token": token,
    }
    result = _api_post(f"{uid}/media", create_params)
    container_id = result.get("id")
    if not container_id:
        print(f"Failed to create container: {result}", file=sys.stderr)
        sys.exit(1)

    # Step 2: Wait for container to be ready
    _wait_for_container(token, container_id)

    # Step 3: Publish
    publish_result = _api_post(f"{uid}/media_publish", {
        "creation_id": container_id,
        "access_token": token,
    })
    post_id = publish_result.get("id")
    print(f"Posted: {post_id}")


def cmd_post_video(args) -> None:
    """Post a reel (video). Instagram treats all video uploads as Reels."""
    token = _get_token(args)
    uid = _get_user_id(args)

    if args.dry_run:
        print(f"[DRY RUN] Would post reel:")
        print(f"  Caption: {args.caption}")
        print(f"  Video:   {args.video_url}")
        return

    # Step 1: Create media container (media_type=REELS for video)
    create_params = {
        "media_type": "REELS",
        "video_url": args.video_url,
        "caption": args.caption,
        "access_token": token,
    }
    result = _api_post(f"{uid}/media", create_params)
    container_id = result.get("id")
    if not container_id:
        print(f"Failed to create container: {result}", file=sys.stderr)
        sys.exit(1)

    # Step 2: Wait for video processing (can take a while)
    _wait_for_container(token, container_id, max_wait=120)

    # Step 3: Publish
    publish_result = _api_post(f"{uid}/media_publish", {
        "creation_id": container_id,
        "access_token": token,
    })
    post_id = publish_result.get("id")
    print(f"Posted: {post_id}")


def cmd_post_carousel(args) -> None:
    """Post a carousel (multiple images/videos)."""
    token = _get_token(args)
    uid = _get_user_id(args)

    if args.dry_run:
        print(f"[DRY RUN] Would post carousel:")
        print(f"  Caption: {args.caption}")
        for i, url in enumerate(args.urls):
            print(f"  Item {i+1}: {url}")
        return

    # Step 1: Create individual item containers
    children_ids = []
    for url in args.urls:
        # Detect if video (simple heuristic: extension)
        is_video = any(url.lower().endswith(ext) for ext in (".mp4", ".mov", ".avi"))
        if is_video:
            params = {
                "media_type": "VIDEO",
                "video_url": url,
                "is_carousel_item": "true",
                "access_token": token,
            }
        else:
            params = {
                "image_url": url,
                "is_carousel_item": "true",
                "access_token": token,
            }
        result = _api_post(f"{uid}/media", params)
        child_id = result.get("id")
        if not child_id:
            print(f"Failed to create carousel item: {result}", file=sys.stderr)
            sys.exit(1)
        _wait_for_container(token, child_id, max_wait=120 if is_video else 30)
        children_ids.append(child_id)

    # Step 2: Create carousel container
    result = _api_post(f"{uid}/media", {
        "media_type": "CAROUSEL",
        "caption": args.caption,
        "children": ",".join(children_ids),
        "access_token": token,
    })
    container_id = result.get("id")
    if not container_id:
        print(f"Failed to create carousel container: {result}", file=sys.stderr)
        sys.exit(1)

    _wait_for_container(token, container_id)

    # Step 3: Publish
    publish_result = _api_post(f"{uid}/media_publish", {
        "creation_id": container_id,
        "access_token": token,
    })
    print(f"Posted carousel: {publish_result.get('id')}")


def cmd_post_story(args) -> None:
    """Post a story (image or video)."""
    token = _get_token(args)
    uid = _get_user_id(args)

    if args.dry_run:
        print(f"[DRY RUN] Would post story:")
        print(f"  Media: {args.media_url}")
        return

    is_video = any(args.media_url.lower().endswith(ext) for ext in (".mp4", ".mov", ".avi"))
    create_params = {
        "media_type": "STORIES",
        "access_token": token,
    }
    if is_video:
        create_params["video_url"] = args.media_url
    else:
        create_params["image_url"] = args.media_url

    result = _api_post(f"{uid}/media", create_params)
    container_id = result.get("id")
    if not container_id:
        print(f"Failed to create story container: {result}", file=sys.stderr)
        sys.exit(1)

    _wait_for_container(token, container_id, max_wait=120 if is_video else 30)

    publish_result = _api_post(f"{uid}/media_publish", {
        "creation_id": container_id,
        "access_token": token,
    })
    print(f"Story posted: {publish_result.get('id')}")


def cmd_reply(args) -> None:
    """Reply to a comment on a post."""
    token = _get_token(args)
    result = _api_post(f"{args.comment_id}/replies", {
        "message": args.text,
        "access_token": token,
    })
    print(f"Replied: {result.get('id')}")


def cmd_posts(args) -> None:
    """List recent posts."""
    token = _get_token(args)
    uid = _get_user_id(args)
    data = _api_get(f"{uid}/media", {
        "fields": "id,caption,timestamp,media_type,permalink,like_count,comments_count",
        "limit": args.n,
        "access_token": token,
    })
    if args.json_output:
        print(json.dumps(data, indent=2))
    else:
        for post in data.get("data", []):
            ts = post.get("timestamp", "")
            caption = post.get("caption", "(no caption)")
            pid = post.get("id", "?")
            mtype = post.get("media_type", "?")
            likes = post.get("like_count", 0)
            comments = post.get("comments_count", 0)
            permalink = post.get("permalink", "")
            print(f"  [{ts}] {pid} ({mtype})")
            for line in caption.splitlines()[:3]:
                print(f"    {line}")
            print(f"    Likes: {likes}  Comments: {comments}")
            if permalink:
                print(f"    {permalink}")
            print()


def cmd_insights(args) -> None:
    """Get insights for a specific post."""
    token = _get_token(args)
    data = _api_get(f"{args.post_id}/insights", {
        "metric": "impressions,reach,likes,comments,shares,saved",
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


def cmd_profile(args) -> None:
    """View profile details."""
    token = _get_token(args)
    uid = _get_user_id(args)
    data = _api_get(uid, {
        "fields": "id,username,name,biography,followers_count,follows_count,media_count,profile_picture_url,website",
        "access_token": token,
    })
    if args.json_output:
        print(json.dumps(data, indent=2))
    else:
        print(f"@{data.get('username', '?')}")
        if data.get("name"):
            print(f"  Name:       {data['name']}")
        if data.get("biography"):
            print(f"  Bio:        {data['biography']}")
        print(f"  Followers:  {data.get('followers_count', '?')}")
        print(f"  Following:  {data.get('follows_count', '?')}")
        print(f"  Posts:      {data.get('media_count', '?')}")
        if data.get("website"):
            print(f"  Website:    {data['website']}")
        print(f"  ID:         {data.get('id', '?')}")


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _wait_for_container(token: str, container_id: str, max_wait: int = 30) -> None:
    """Poll container status until FINISHED or timeout."""
    s = None
    for _ in range(max_wait):
        status = _api_get(container_id, {
            "fields": "status_code",
            "access_token": token,
        })
        s = status.get("status_code")
        if s == "FINISHED":
            return
        if s == "ERROR":
            # Try to get error details
            err_info = _api_get(container_id, {
                "fields": "status_code,status",
                "access_token": token,
            })
            print(f"Container error: {err_info}", file=sys.stderr)
            sys.exit(1)
        time.sleep(1)
    print(f"Timeout waiting for container {container_id} (status: {s})", file=sys.stderr)
    sys.exit(1)


# ---------------------------------------------------------------------------
# Argument parser
# ---------------------------------------------------------------------------

def main() -> None:
    parser = argparse.ArgumentParser(prog="instagram", description="Instagram CLI for picoclaw")
    parser.add_argument("--json", dest="json_output", action="store_true", help="JSON output")
    parser.add_argument("--token", help="Access token (overrides INSTAGRAM_ACCESS_TOKEN env var)")
    parser.add_argument("--user-id", help="Instagram User ID (overrides INSTAGRAM_USER_ID env var)")
    sub = parser.add_subparsers(dest="command")

    # whoami
    sub.add_parser("whoami", help="Show current user info")

    # post-image
    p = sub.add_parser("post-image", help="Post a single image")
    p.add_argument("image_url", help="Public image URL (JPEG recommended)")
    p.add_argument("--caption", default="", help="Post caption")
    p.add_argument("--dry-run", action="store_true", help="Preview without posting")

    # post-video (reel)
    p = sub.add_parser("post-video", help="Post a reel (video)")
    p.add_argument("video_url", help="Public video URL (mp4)")
    p.add_argument("--caption", default="", help="Reel caption")
    p.add_argument("--dry-run", action="store_true", help="Preview without posting")

    # post-carousel
    p = sub.add_parser("post-carousel", help="Post a carousel (multiple images/videos)")
    p.add_argument("urls", nargs="+", help="Public media URLs (2-10 images/videos)")
    p.add_argument("--caption", default="", help="Carousel caption")
    p.add_argument("--dry-run", action="store_true", help="Preview without posting")

    # post-story
    p = sub.add_parser("post-story", help="Post a story (image or video)")
    p.add_argument("media_url", help="Public media URL")
    p.add_argument("--dry-run", action="store_true", help="Preview without posting")

    # reply
    p = sub.add_parser("reply", help="Reply to a comment")
    p.add_argument("comment_id", help="Comment ID to reply to")
    p.add_argument("text", help="Reply text")

    # posts
    p = sub.add_parser("posts", help="List recent posts")
    p.add_argument("-n", type=int, default=10, help="Number of posts")

    # insights
    p = sub.add_parser("insights", help="Get insights for a post")
    p.add_argument("post_id", help="Post ID")

    # profile
    sub.add_parser("profile", help="View your profile")

    args = parser.parse_args()
    if not args.command:
        parser.print_help()
        sys.exit(1)

    handlers = {
        "whoami": cmd_whoami,
        "post-image": cmd_post_image,
        "post-video": cmd_post_video,
        "post-carousel": cmd_post_carousel,
        "post-story": cmd_post_story,
        "reply": cmd_reply,
        "posts": cmd_posts,
        "insights": cmd_insights,
        "profile": cmd_profile,
    }

    try:
        handlers[args.command](args)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
