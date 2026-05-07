#!/usr/bin/env bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

failures=0

error() {
	local path="$1"
	local reason="$2"
	local suggestion="${3:-}"

	echo "docs lint: $path" >&2
	echo "  reason: $reason" >&2
	if [[ -n "$suggestion" ]]; then
		echo "  fix: $suggestion" >&2
	fi
	failures=1
}

lowercase() {
	printf '%s' "$1" | tr '[:upper:]' '[:lower:]'
}

suggest_noncanonical_translation_name() {
	local path="$1"
	local dir
	local base
	local stem
	local locale

	dir="$(dirname "$path")"
	base="$(basename "$path")"

	if [[ "$base" =~ ^(.+)_([A-Za-z]{2}(-[A-Za-z]{2})?)\.md$ ]]; then
		stem="${BASH_REMATCH[1]}"
		locale="$(lowercase "${BASH_REMATCH[2]}")"
		printf '%s/%s.%s.md' "$dir" "$stem" "$locale"
		return
	fi

	if [[ "$base" =~ ^(.+)\.([A-Za-z]{2}(-[A-Za-z]{2})?)\.md$ ]]; then
		stem="${BASH_REMATCH[1]}"
		locale="$(lowercase "${BASH_REMATCH[2]}")"
		printf '%s/%s.%s.md' "$dir" "$stem" "$locale"
		return
	fi

	printf 'rename it to use a lowercase .<locale>.md suffix beside the English source'
}

suggest_docs_language_bucket_target() {
	local path="$1"
	local locale
	local file
	local name
	local -a matches

	if [[ "$path" =~ ^docs/([A-Za-z]{2}(-[A-Za-z]{2})?)/.+\.md$ ]]; then
		locale="$(lowercase "${BASH_REMATCH[1]}")"
		file="$(basename "$path")"
		name="${file%.md}"
		mapfile -t matches < <(find docs/project docs/guides docs/reference docs/operations docs/security docs/architecture docs/channels docs/design docs/migration -type f -name "${name}.md" 2>/dev/null | sort)
		if [[ "${#matches[@]}" -eq 1 ]]; then
			printf '%s' "${matches[0]%.md}.${locale}.md"
			return
		fi
	fi

	printf 'move it to a typed docs directory and rename it to <name>.<locale>.md beside the English source'
}

suggest_nested_locale_bucket_target() {
	local path="$1"
	local prefix
	local locale
	local rest

	if [[ "$path" =~ ^(docs/(project|guides|reference|operations|security|architecture|design|migration))/([A-Za-z]{2}(-[A-Za-z]{2})?)/(.*)\.md$ ]]; then
		prefix="${BASH_REMATCH[1]}"
		locale="$(lowercase "${BASH_REMATCH[3]}")"
		rest="${BASH_REMATCH[5]}"
		printf '%s/%s.%s.md' "$prefix" "$rest" "$locale"
		return
	fi

	if [[ "$path" =~ ^(docs/channels/[^/]+)/([A-Za-z]{2}(-[A-Za-z]{2})?)/(.*)\.md$ ]]; then
		prefix="${BASH_REMATCH[1]}"
		locale="$(lowercase "${BASH_REMATCH[2]}")"
		rest="${BASH_REMATCH[4]}"
		printf '%s/%s.%s.md' "$prefix" "$rest" "$locale"
		return
	fi

	printf 'move the file beside its English source and rename it to <name>.<locale>.md'
}

is_noncanonical_translation_name() {
	local path="$1"
	local base

	base="$(basename "$path")"

	[[ "$base" =~ ^.+_[A-Za-z]{2}(-[A-Za-z]{2})?\.md$ ]] && return 0
	[[ "$base" =~ ^.+\.[A-Z]{2}(-[A-Z]{2})?\.md$ ]] && return 0
	[[ "$base" =~ ^.+\.[a-z]{2}-[A-Z]{2}\.md$ ]] && return 0
	[[ "$base" =~ ^.+\.[A-Z]{2}-[a-z]{2}\.md$ ]] && return 0

	return 1
}

is_noncanonical_locale_bucket() {
	local path="$1"

	[[ "$path" =~ ^docs/(project|guides|reference|operations|security|architecture|design|migration)/[A-Za-z]{2}(-[A-Za-z]{2})?/ ]] && return 0
	[[ "$path" =~ ^docs/channels/[^/]+/[A-Za-z]{2}(-[A-Za-z]{2})?/ ]] && return 0
	return 1
}

is_root_docs_language_bucket() {
	local path="$1"
	[[ "$path" =~ ^docs/[A-Za-z]{2}(-[A-Za-z]{2})?/ ]]
}

is_translation_file() {
	local path="$1"
	[[ "$path" =~ ^(.+)\.([a-z]{2})(-[a-z]{2})?\.md$ ]]
}

translation_base() {
	local path="$1"
	local locale="$2"

	if [[ "$path" == docs/project/* ]]; then
		local rel="${path#docs/project/}"
		echo "${rel%.$locale.md}.md"
		return
	fi

	echo "${path%.$locale.md}.md"
}

while IFS= read -r path; do
	[[ -f "$path" ]] || continue

	case "$path" in
		README.*.md)
			error \
				"$path" \
				"translated project entry docs must live under docs/project/" \
				"move it to docs/project/$(basename "$path")"
			;;
		CONTRIBUTING.*.md)
			error \
				"$path" \
				"translated project entry docs must live under docs/project/" \
				"move it to docs/project/$(basename "$path")"
			;;
	esac

	if [[ "$path" =~ (^|/)README_[A-Za-z0-9-]+\.md$ ]]; then
		error \
			"$path" \
			"legacy README translation names are not allowed" \
			"rename it to use README.<locale>.md, for example $(suggest_noncanonical_translation_name "$path")"
	fi

	if is_noncanonical_translation_name "$path"; then
		error \
			"$path" \
			"translation files must use lowercase .<locale>.md suffixes and no underscore variants" \
			"rename it to $(suggest_noncanonical_translation_name "$path")"
	fi

	if is_root_docs_language_bucket "$path"; then
		error \
			"$path" \
			"language bucket directories under docs/ are not allowed" \
			"move it to $(suggest_docs_language_bucket_target "$path")"
	fi

	if is_noncanonical_locale_bucket "$path"; then
		error \
			"$path" \
			"translations must live beside the English source, not under locale-named subdirectories" \
			"move it to $(suggest_nested_locale_bucket_target "$path")"
	fi

	if [[ "$path" =~ ^docs/[^/]+\.md$ && "$path" != "docs/README.md" ]]; then
		error \
			"$path" \
			"top-level docs Markdown files must move into a typed docs/ subdirectory" \
			"move it into one of docs/project/, docs/guides/, docs/reference/, docs/operations/, docs/security/, docs/architecture/, docs/channels/, docs/design/, or docs/migration/"
	fi

	if is_translation_file "$path"; then
		locale="${BASH_REMATCH[2]}${BASH_REMATCH[3]}"

		if [[ "$path" == docs/design/* ]]; then
			continue
		fi

		base="$(translation_base "$path" "$locale")"
		if [[ ! -f "$base" ]]; then
			error \
				"$path" \
				"missing English source document '$base'" \
				"add the English source document at '$base' or move this translation beside the correct English source"
		fi
	fi
done < <(git ls-files --cached --others --exclude-standard -- '*.md')

if [[ "$failures" -ne 0 ]]; then
	echo "docs lint: failed" >&2
	exit 1
fi

echo "docs lint: OK"
