#!/usr/bin/env python3
"""bsky — Bluesky CLI for picoclaw.

Thin wrapper around the atproto SDK. Installed to /usr/local/bin/bsky
inside the Docker image so the agent can call `bsky post`, `bsky search`, etc.

Config is stored at $XDG_CONFIG_HOME/bsky/config.json (defaults to ~/.config/bsky/).
"""

import argparse
import json
import os
import re
import sys
import textwrap
from pathlib import Path

from atproto import Client, client_utils, models

# URL regex for detecting links in post text
_URL_RE = re.compile(r'https?://[^\s)>"]+')

# ---------------------------------------------------------------------------
# Config helpers
# ---------------------------------------------------------------------------

def _config_path() -> Path:
    base = os.environ.get("XDG_CONFIG_HOME", os.path.expanduser("~/.config"))
    return Path(base) / "bsky" / "config.json"


def _load_config() -> dict:
    p = _config_path()
    if p.exists():
        return json.loads(p.read_text())
    return {}


def _save_config(cfg: dict) -> None:
    p = _config_path()
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(cfg, indent=2))


def _client(require_auth: bool = True) -> Client:
    """Return an authenticated Client, restoring the saved session."""
    cfg = _load_config()
    client = Client()
    session_string = cfg.get("session_string")
    if session_string:
        try:
            client.login(session_string=session_string)
            # Persist refreshed session
            _save_config({**cfg, "session_string": client.export_session_string()})
            return client
        except Exception:
            pass  # session expired — fall through
    if require_auth:
        # Try env-var credentials as fallback
        handle = os.environ.get("BSKY_HANDLE") or cfg.get("handle")
        password = os.environ.get("BSKY_APP_PASSWORD") or cfg.get("app_password")
        if handle and password:
            client.login(handle, password)
            _save_config({
                **cfg,
                "handle": handle,
                "session_string": client.export_session_string(),
            })
            return client
        print("Not logged in. Run: bsky login --handle HANDLE --password APP_PASSWORD", file=sys.stderr)
        sys.exit(1)
    return client


def _resolve_post_uri(client: Client, url_or_uri: str) -> str:
    """Convert a https://bsky.app/profile/…/post/… URL to an at:// URI."""
    if url_or_uri.startswith("at://"):
        return url_or_uri
    # https://bsky.app/profile/<handle>/post/<rkey>
    parts = url_or_uri.rstrip("/").split("/")
    try:
        idx = parts.index("post")
        handle = parts[idx - 1]
        rkey = parts[idx + 1]
    except (ValueError, IndexError):
        print(f"Cannot parse post URL: {url_or_uri}", file=sys.stderr)
        sys.exit(1)
    did = client.resolve_handle(handle).did
    return f"at://{did}/app.bsky.feed.post/{rkey}"


def _resolve_did(client: Client, handle: str) -> str:
    handle = handle.lstrip("@")
    if not "." in handle:
        handle += ".bsky.social"
    return client.resolve_handle(handle).did


def _print_post(post, *, indent: str = "") -> None:
    record = post.record if hasattr(post, "record") else post.value
    author = post.author if hasattr(post, "author") else None
    handle = author.handle if author else "?"
    text = record.text if hasattr(record, "text") else str(record)
    uri = post.uri if hasattr(post, "uri") else "?"
    like_count = post.like_count if hasattr(post, "like_count") else ""
    repost_count = post.repost_count if hasattr(post, "repost_count") else ""
    created = record.created_at if hasattr(record, "created_at") else ""
    print(f"{indent}@{handle}  {created}")
    for line in text.splitlines():
        print(f"{indent}  {line}")
    stats = []
    if like_count:
        stats.append(f"♥ {like_count}")
    if repost_count:
        stats.append(f"🔁 {repost_count}")
    if stats:
        print(f"{indent}  {' | '.join(stats)}")
    print(f"{indent}  {uri}")
    print()


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

def cmd_login(args) -> None:
    client = Client()
    client.login(args.handle, args.password)
    cfg = _load_config()
    cfg["handle"] = args.handle
    cfg["session_string"] = client.export_session_string()
    _save_config(cfg)
    profile = client.me
    print(f"Logged in as @{profile.handle} ({profile.did})")


def cmd_whoami(args) -> None:
    cfg = _load_config()
    try:
        client = _client(require_auth=True)
        profile = client.me
        if args.json_output:
            print(json.dumps({"handle": profile.handle, "did": profile.did}))
        else:
            print(f"Handle: @{profile.handle}")
            print(f"DID:    {profile.did}")
    except SystemExit:
        print("Not logged in")


def cmd_post(args) -> None:
    client = _client()
    text = args.text
    if args.dry_run:
        print(f"[DRY RUN] Would post ({len(text)} chars):")
        print(text)
        return

    # Detect if the attachment is a video by extension
    VIDEO_EXTS = {".mp4", ".webm", ".mov", ".mpeg", ".mpg"}
    media_path = Path(args.image) if args.image else None
    is_video = media_path and media_path.suffix.lower() in VIDEO_EXTS

    if is_video:
        if not media_path.exists():
            print(f"Video not found: {args.image}", file=sys.stderr)
            sys.exit(1)
        with open(media_path, "rb") as f:
            video_data = f.read()
        alt_text = args.alt or None
        resp = client.send_video(
            text=text,
            video=video_data,
            video_alt=alt_text,
        )
        print(f"Posted (with video): {resp.uri}")
        return

    images = []
    if args.image:
        if not media_path.exists():
            print(f"Image not found: {args.image}", file=sys.stderr)
            sys.exit(1)
        alt_text = args.alt or ""
        with open(media_path, "rb") as f:
            img_data = f.read()
        upload = client.upload_blob(img_data)
        images.append(models.AppBskyEmbedImages.Image(
            alt=alt_text,
            image=upload.blob,
        ))

    embed = None
    if images:
        embed = models.AppBskyEmbedImages.Main(images=images)

    # Build rich text with clickable links
    tb = _build_rich_text(text)
    resp = client.send_post(text=tb, embed=embed)
    print(f"Posted: {resp.uri}")


def _build_rich_text(text: str) -> client_utils.TextBuilder:
    """Parse URLs in text and return a TextBuilder with link facets."""
    tb = client_utils.TextBuilder()
    last_end = 0
    for m in _URL_RE.finditer(text):
        if m.start() > last_end:
            tb.text(text[last_end:m.start()])
        url = m.group(0)
        tb.link(url, url)
        last_end = m.end()
    if last_end < len(text):
        tb.text(text[last_end:])
    return tb


def cmd_create_thread(args) -> None:
    client = _client()
    texts = args.texts
    if not texts:
        print("No text provided for thread", file=sys.stderr)
        sys.exit(1)

    if args.dry_run:
        print(f"[DRY RUN] Would create thread with {len(texts)} posts:")
        for i, t in enumerate(texts, 1):
            print(f"  [{i}] {t}")
        return

    # First post may have an image
    images = []
    if args.image:
        img_path = Path(args.image)
        if not img_path.exists():
            print(f"Image not found: {args.image}", file=sys.stderr)
            sys.exit(1)
        with open(img_path, "rb") as f:
            img_data = f.read()
        upload = client.upload_blob(img_data)
        images.append(models.AppBskyEmbedImages.Image(
            alt=args.alt or "",
            image=upload.blob,
        ))

    embed = models.AppBskyEmbedImages.Main(images=images) if images else None
    parent = client.send_post(text=texts[0], embed=embed)
    print(f"[1/{len(texts)}] {parent.uri}")
    root_ref = models.create_strong_ref(parent)
    parent_ref = root_ref

    for i, text in enumerate(texts[1:], 2):
        reply_to = models.AppBskyFeedPost.ReplyRef(root=root_ref, parent=parent_ref)
        resp = client.send_post(text=text, reply_to=reply_to)
        print(f"[{i}/{len(texts)}] {resp.uri}")
        parent_ref = models.create_strong_ref(resp)


def cmd_reply(args) -> None:
    client = _client()
    uri = _resolve_post_uri(client, args.url)
    # Fetch the post to get its CID and root
    thread = client.get_post_thread(uri=uri)
    post = thread.thread.post
    post_ref = models.create_strong_ref(post)
    # Determine root — if the post is itself a reply, use its root
    root_ref = post_ref
    if hasattr(post, "record") and hasattr(post.record, "reply") and post.record.reply:
        root_ref = post.record.reply.root
    reply_to = models.AppBskyFeedPost.ReplyRef(root=root_ref, parent=post_ref)
    resp = client.send_post(text=args.text, reply_to=reply_to)
    print(f"Replied: {resp.uri}")


def cmd_quote(args) -> None:
    client = _client()
    uri = _resolve_post_uri(client, args.url)
    thread = client.get_post_thread(uri=uri)
    post = thread.thread.post
    ref = models.create_strong_ref(post)
    embed = models.AppBskyEmbedRecord.Main(record=ref)
    resp = client.send_post(text=args.text, embed=embed)
    print(f"Quoted: {resp.uri}")


def cmd_like(args) -> None:
    client = _client()
    uri = _resolve_post_uri(client, args.url)
    thread = client.get_post_thread(uri=uri)
    post = thread.thread.post
    client.like(uri=post.uri, cid=post.cid)
    print(f"Liked: {uri}")


def cmd_repost(args) -> None:
    client = _client()
    uri = _resolve_post_uri(client, args.url)
    thread = client.get_post_thread(uri=uri)
    post = thread.thread.post
    client.repost(uri=post.uri, cid=post.cid)
    print(f"Reposted: {uri}")


def cmd_follow(args) -> None:
    client = _client()
    did = _resolve_did(client, args.handle)
    client.follow(did)
    print(f"Followed: {args.handle}")


def cmd_unfollow(args) -> None:
    client = _client()
    did = _resolve_did(client, args.handle)
    # Need to find the follow record to delete it
    follows = client.app.bsky.graph.get_follows({"actor": client.me.did, "limit": 100})
    for f in follows.follows:
        if f.did == did:
            # Delete the follow record
            repo = client.me.did
            rkey = f.viewer.following.split("/")[-1] if f.viewer and f.viewer.following else None
            if rkey:
                client.app.bsky.graph.follow.delete(repo, rkey)
                print(f"Unfollowed: {args.handle}")
                return
    print(f"Not following: {args.handle}")


def cmd_block(args) -> None:
    client = _client()
    did = _resolve_did(client, args.handle)
    client.app.bsky.graph.block.create(client.me.did, models.AppBskyGraphBlock.Record(
        subject=did,
        created_at=client.get_current_time_iso(),
    ))
    print(f"Blocked: {args.handle}")


def cmd_mute(args) -> None:
    client = _client()
    did = _resolve_did(client, args.handle)
    client.app.bsky.graph.mute_actor({"actor": did})
    print(f"Muted: {args.handle}")


def cmd_search(args) -> None:
    client = _client()
    limit = args.n or 10
    resp = client.app.bsky.feed.search_posts({"q": args.query, "limit": limit})
    if args.json_output:
        posts = []
        for item in resp.posts:
            posts.append({
                "uri": item.uri,
                "author": item.author.handle,
                "text": item.record.text,
                "created_at": item.record.created_at if hasattr(item.record, "created_at") else "",
                "like_count": item.like_count,
                "repost_count": item.repost_count,
            })
        print(json.dumps(posts, indent=2, default=str))
    else:
        if not resp.posts:
            print("No results found.")
            return
        for post in resp.posts:
            _print_post(post)


def cmd_timeline(args) -> None:
    client = _client()
    limit = args.n or 20
    resp = client.get_timeline({"limit": limit})
    if args.json_output:
        posts = []
        for item in resp.feed:
            posts.append({
                "uri": item.post.uri,
                "author": item.post.author.handle,
                "text": item.post.record.text if hasattr(item.post.record, "text") else "",
                "created_at": item.post.record.created_at if hasattr(item.post.record, "created_at") else "",
            })
        print(json.dumps(posts, indent=2, default=str))
    else:
        for item in resp.feed:
            _print_post(item.post)


def cmd_notifications(args) -> None:
    client = _client()
    resp = client.app.bsky.notification.list_notifications({"limit": args.n or 20})
    if args.json_output:
        notifs = []
        for n in resp.notifications:
            notifs.append({
                "reason": n.reason,
                "author": n.author.handle,
                "uri": n.uri,
                "is_read": n.is_read,
            })
        print(json.dumps(notifs, indent=2, default=str))
    else:
        for n in resp.notifications:
            emoji = {"like": "♥", "repost": "🔁", "follow": "👤", "reply": "💬", "mention": "📢"}.get(n.reason, "•")
            print(f"{emoji} @{n.author.handle} {n.reason}")


def cmd_delete(args) -> None:
    client = _client()
    uri = _resolve_post_uri(client, args.url)
    client.delete_post(uri)
    print(f"Deleted: {uri}")


def cmd_thread(args) -> None:
    client = _client()
    uri = _resolve_post_uri(client, args.url)
    resp = client.get_post_thread(uri=uri)

    def _walk(node, depth=0):
        if hasattr(node, "post"):
            _print_post(node.post, indent="  " * depth)
        if hasattr(node, "replies"):
            for r in (node.replies or []):
                _walk(r, depth + 1)

    _walk(resp.thread)


def cmd_profile(args) -> None:
    client = _client()
    handle = args.handle.lstrip("@")
    if "." not in handle:
        handle += ".bsky.social"
    resp = client.app.bsky.actor.get_profile({"actor": handle})
    if args.json_output:
        print(json.dumps({
            "handle": resp.handle,
            "did": resp.did,
            "display_name": resp.display_name,
            "description": resp.description,
            "followers_count": resp.followers_count,
            "follows_count": resp.follows_count,
            "posts_count": resp.posts_count,
        }, indent=2, default=str))
    else:
        print(f"@{resp.handle}")
        if resp.display_name:
            print(f"  Name: {resp.display_name}")
        if resp.description:
            print(f"  Bio:  {resp.description}")
        print(f"  Followers: {resp.followers_count}  Following: {resp.follows_count}  Posts: {resp.posts_count}")


# ---------------------------------------------------------------------------
# Argument parser
# ---------------------------------------------------------------------------

def main() -> None:
    parser = argparse.ArgumentParser(prog="bsky", description="Bluesky CLI")
    parser.add_argument("--json", dest="json_output", action="store_true", help="JSON output for read commands")
    sub = parser.add_subparsers(dest="command")

    # login
    p = sub.add_parser("login", help="Log in to Bluesky")
    p.add_argument("--handle", required=True)
    p.add_argument("--password", required=True)

    # whoami
    sub.add_parser("whoami", help="Show current user")

    # post
    p = sub.add_parser("post", help="Create a post")
    p.add_argument("text", help="Post text")
    p.add_argument("--image", help="Path to image or video file (mp4/webm/mov auto-detected as video)")
    p.add_argument("--alt", help="Alt text for image/video")
    p.add_argument("--dry-run", action="store_true", help="Preview without posting")

    # create-thread / ct
    for name in ("create-thread", "ct"):
        p = sub.add_parser(name, help="Create a thread")
        p.add_argument("texts", nargs="+", help="Text for each post in the thread")
        p.add_argument("--image", help="Image for first post")
        p.add_argument("--alt", help="Alt text for image")
        p.add_argument("--dry-run", action="store_true", help="Preview without posting")

    # reply
    p = sub.add_parser("reply", help="Reply to a post")
    p.add_argument("url", help="Post URL or AT-URI")
    p.add_argument("text", help="Reply text")

    # quote
    p = sub.add_parser("quote", help="Quote a post")
    p.add_argument("url", help="Post URL or AT-URI")
    p.add_argument("text", help="Quote text")

    # like
    p = sub.add_parser("like", help="Like a post")
    p.add_argument("url", help="Post URL or AT-URI")

    # repost / boost / rt
    for name in ("repost", "boost", "rt"):
        p = sub.add_parser(name, help="Repost a post")
        p.add_argument("url", help="Post URL or AT-URI")

    # follow
    p = sub.add_parser("follow", help="Follow a user")
    p.add_argument("handle", help="Handle (@user or user.bsky.social)")

    # unfollow
    p = sub.add_parser("unfollow", help="Unfollow a user")
    p.add_argument("handle", help="Handle")

    # block
    p = sub.add_parser("block", help="Block a user")
    p.add_argument("handle", help="Handle")

    # mute
    p = sub.add_parser("mute", help="Mute a user")
    p.add_argument("handle", help="Handle")

    # search
    p = sub.add_parser("search", help="Search posts")
    p.add_argument("query", help="Search query")
    p.add_argument("-n", type=int, default=10, help="Number of results")

    # timeline / tl
    for name in ("timeline", "tl"):
        p = sub.add_parser(name, help="View timeline")
        p.add_argument("-n", type=int, default=20, help="Number of posts")

    # notifications / n
    for name in ("notifications", "n"):
        p = sub.add_parser(name, help="View notifications")
        p.add_argument("-n", type=int, default=20, help="Number of notifications")

    # delete
    p = sub.add_parser("delete", help="Delete a post")
    p.add_argument("url", help="Post URL or AT-URI")

    # thread
    p = sub.add_parser("thread", help="View a thread")
    p.add_argument("url", help="Post URL or AT-URI")

    # profile
    p = sub.add_parser("profile", help="View a user profile")
    p.add_argument("handle", help="Handle")

    args = parser.parse_args()
    if not args.command:
        parser.print_help()
        sys.exit(1)

    handlers = {
        "login": cmd_login,
        "whoami": cmd_whoami,
        "post": cmd_post,
        "create-thread": cmd_create_thread,
        "ct": cmd_create_thread,
        "reply": cmd_reply,
        "quote": cmd_quote,
        "like": cmd_like,
        "repost": cmd_repost,
        "boost": cmd_repost,
        "rt": cmd_repost,
        "follow": cmd_follow,
        "unfollow": cmd_unfollow,
        "block": cmd_block,
        "mute": cmd_mute,
        "search": cmd_search,
        "timeline": cmd_timeline,
        "tl": cmd_timeline,
        "notifications": cmd_notifications,
        "n": cmd_notifications,
        "delete": cmd_delete,
        "thread": cmd_thread,
        "profile": cmd_profile,
    }

    try:
        handlers[args.command](args)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
