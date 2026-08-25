#!/usr/bin/env bash

set -euo pipefail

API_VERSION="${AZURE_SEARCH_API_VERSION:-2025-11-01-preview}"
OUTPUT_DIR="${1:-./exports/azure-search}"
BATCH_SIZE="${AZURE_SEARCH_EXPORT_BATCH_SIZE:-1000}"

load_env_value() {
  local key="$1"
  local file="$2"
  local value

  value="$(grep -m1 -E "^${key}=" "${file}" | sed -E "s/^${key}=//")" || true
  value="${value%$'\r'}"

  # Remove a single layer of surrounding quotes if present.
  if [[ "${value}" =~ ^\".*\"$ ]]; then
    value="${value:1:${#value}-2}"
  elif [[ "${value}" =~ ^\'.*\'$ ]]; then
    value="${value:1:${#value}-2}"
  fi

  printf '%s' "${value}"
}

if [[ -f ".env" ]]; then
  if [[ -z "${AZURE_SEARCH_BASEURL:-}" ]]; then
    AZURE_SEARCH_BASEURL="$(load_env_value "AZURE_SEARCH_BASEURL" ".env")"
  fi
  if [[ -z "${AZURE_SEARCH_APIKEY:-}" ]]; then
    AZURE_SEARCH_APIKEY="$(load_env_value "AZURE_SEARCH_APIKEY" ".env")"
  fi
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "Error: curl is required." >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "Error: jq is required." >&2
  exit 1
fi

if [[ -z "${AZURE_SEARCH_BASEURL:-}" ]]; then
  echo "Error: AZURE_SEARCH_BASEURL is not set." >&2
  exit 1
fi

if [[ -z "${AZURE_SEARCH_APIKEY:-}" ]]; then
  echo "Error: AZURE_SEARCH_APIKEY is not set." >&2
  exit 1
fi

if ! [[ "${BATCH_SIZE}" =~ ^[1-9][0-9]*$ ]]; then
  echo "Error: AZURE_SEARCH_EXPORT_BATCH_SIZE must be a positive integer." >&2
  exit 1
fi

BASE_URL="${AZURE_SEARCH_BASEURL%/}"
mkdir -p "${OUTPUT_DIR}"

echo "Listing indexes from ${BASE_URL} ..."
indexes_json="$(
  curl -fsS \
    -H "api-key: ${AZURE_SEARCH_APIKEY}" \
    "${BASE_URL}/indexes?api-version=${API_VERSION}&%24select=name"
)"

indexes=()
while IFS= read -r index_name; do
  [[ -n "${index_name}" ]] && indexes+=("${index_name}")
done < <(echo "${indexes_json}" | jq -r '.value[]?.name')

if [[ "${#indexes[@]}" -eq 0 ]]; then
  echo "No indexes found."
  exit 0
fi

echo "Found ${#indexes[@]} index(es)."

for index_name in "${indexes[@]}"; do
  echo "Exporting index: ${index_name}"

  safe_name="$(echo "${index_name}" | tr '/: ' '___')"
  output_file="${OUTPUT_DIR}/${safe_name}.json"
  tmp_file="$(mktemp)"
  echo '[]' > "${tmp_file}"

  skip=0
  total=0

  while true; do
    request_body="$(jq -nc \
      --argjson top "${BATCH_SIZE}" \
      --argjson skip "${skip}" \
      '{search:"*", top:$top, skip:$skip, count:true}')"

    response="$(
      curl -fsS \
        -X POST \
        -H "api-key: ${AZURE_SEARCH_APIKEY}" \
        -H "Content-Type: application/json" \
        "${BASE_URL}/indexes/${index_name}/docs/search?api-version=${API_VERSION}" \
        -d "${request_body}"
    )"

    batch_count="$(echo "${response}" | jq '.value | length')"

    if [[ "${batch_count}" -eq 0 ]]; then
      break
    fi

    next_tmp="$(mktemp)"
    jq -s '.[0] + .[1]' "${tmp_file}" <(echo "${response}" | jq '.value') > "${next_tmp}"
    mv "${next_tmp}" "${tmp_file}"

    total=$((total + batch_count))
    skip=$((skip + BATCH_SIZE))

    if [[ "${batch_count}" -lt "${BATCH_SIZE}" ]]; then
      break
    fi
  done

  exported_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  jq -n \
    --arg index "${index_name}" \
    --arg exported_at "${exported_at}" \
    --argjson count "${total}" \
    --slurpfile docs "${tmp_file}" \
    '{index:$index, exported_at:$exported_at, count:$count, documents:$docs[0]}' > "${output_file}"

  rm -f "${tmp_file}"
  echo "  -> ${output_file} (${total} documents)"
done

echo "Done. Exports saved to ${OUTPUT_DIR}"
