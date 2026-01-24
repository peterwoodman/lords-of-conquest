#!/bin/bash
# Ralph Wiggum: Stream Parser
#
# Parses cursor-agent stream-json output in real-time.
# Tracks token usage, detects failures/gutter, writes to .ralph/ logs.
#
# Usage:
#   cursor-agent -p --force --output-format stream-json "..." | ./stream-parser.sh /path/to/workspace
#
# Outputs to stdout:
#   - GUTTER when stuck pattern detected
#   - COMPLETE when agent outputs <ralph>COMPLETE</ralph>
#
# Writes to .ralph/:
#   - activity.log: all operations with token counts
#   - errors.log: failures and gutter detection
#
# Note: Token counters are RESET when a new session starts (system/init event).
# This ensures each session has fresh context tracking starting from 0.

set -euo pipefail

WORKSPACE="${1:-.}"
RALPH_DIR="$WORKSPACE/.ralph"

# Ensure .ralph directory exists
mkdir -p "$RALPH_DIR"

# Tracking state
BYTES_READ=0
BYTES_WRITTEN=0
ASSISTANT_CHARS=0
SHELL_OUTPUT_CHARS=0
PROMPT_CHARS=0
TOOL_CALLS=0

# Estimate initial prompt size (Ralph prompt is ~2KB + file references)
PROMPT_CHARS=3000

# Gutter detection - use temp files instead of associative arrays (macOS bash 3.x compat)
FAILURES_FILE=$(mktemp)
WRITES_FILE=$(mktemp)
trap "rm -f $FAILURES_FILE $WRITES_FILE" EXIT

calc_tokens() {
  local total_bytes=$((PROMPT_CHARS + BYTES_READ + BYTES_WRITTEN + ASSISTANT_CHARS + SHELL_OUTPUT_CHARS))
  echo $((total_bytes / 4))
}

# Log to activity.log
log_activity() {
  local message="$1"
  local timestamp=$(date '+%H:%M:%S')

  echo "[$timestamp] $message" >> "$RALPH_DIR/activity.log"
}

# Log to errors.log
log_error() {
  local message="$1"
  local timestamp=$(date '+%H:%M:%S')

  echo "[$timestamp] $message" >> "$RALPH_DIR/errors.log"
}

# Check and log token status
log_token_status() {
  local tokens=$(calc_tokens)
  local timestamp=$(date '+%H:%M:%S')

  local breakdown="[read:$((BYTES_READ/1024))KB write:$((BYTES_WRITTEN/1024))KB assist:$((ASSISTANT_CHARS/1024))KB shell:$((SHELL_OUTPUT_CHARS/1024))KB]"
  echo "[$timestamp] TOKENS: ~$tokens $breakdown" >> "$RALPH_DIR/activity.log"
}


# Track shell command failure
track_shell_failure() {
  local cmd="$1"
  local exit_code="$2"

  if [[ $exit_code -ne 0 ]]; then
    # Count failures for this command (grep -cFx for fixed string, whole line match)
    local count
    count=$(grep -cFx "$cmd" "$FAILURES_FILE" 2>/dev/null) || count=0
    count=$((count + 1))
    echo "$cmd" >> "$FAILURES_FILE"

    log_error "SHELL FAIL: $cmd → exit $exit_code (attempt $count)"

    if [[ $count -ge 3 ]]; then
      log_error "⚠️ GUTTER: same command failed ${count}x"
      echo "GUTTER" 2>/dev/null || true
    fi
  fi
}

# Track file writes for thrashing detection
track_file_write() {
  local path="$1"
  local now=$(date +%s)

  # Log write with timestamp
  echo "$now:$path" >> "$WRITES_FILE"

  # Count writes to this file in last 10 minutes
  local cutoff=$((now - 600))
  local count=$(awk -F: -v cutoff="$cutoff" -v path="$path" '
    $1 >= cutoff && $2 == path { count++ }
    END { print count+0 }
  ' "$WRITES_FILE")

  # Check for thrashing (5+ writes in 10 minutes)
  if [[ $count -ge 5 ]]; then
    log_error "⚠️ THRASHING: $path written ${count}x in 10 min"
    echo "GUTTER" 2>/dev/null || true
  fi
}

# Process a single JSON line from stream
process_line() {
  local line="$1"

  # Skip empty lines
  [[ -z "$line" ]] && return

  # Parse JSON type
  local type=$(echo "$line" | jq -r '.type // empty' 2>/dev/null) || return
  local subtype=$(echo "$line" | jq -r '.subtype // empty' 2>/dev/null) || true

  case "$type" in
    "system")
      if [[ "$subtype" == "init" ]]; then
        local model=$(echo "$line" | jq -r '.model // "unknown"' 2>/dev/null) || model="unknown"

        # Reset counters on session init for fresh tracking
        BYTES_READ=0
        BYTES_WRITTEN=0
        ASSISTANT_CHARS=0
        SHELL_OUTPUT_CHARS=0
        PROMPT_CHARS=3000  # Base prompt estimate
        TOOL_CALLS=0

        # Clear gutter tracking files
        > "$FAILURES_FILE"
        > "$WRITES_FILE"

        log_activity "SESSION START: model=$model"
      fi
      ;;

    "assistant")
      # Track assistant message characters
      local text=$(echo "$line" | jq -r '.message.content[0].text // empty' 2>/dev/null) || text=""
      if [[ -n "$text" ]]; then
        local chars=${#text}
        ASSISTANT_CHARS=$((ASSISTANT_CHARS + chars))

        # Check for completion sigil
        if [[ "$text" == *"<ralph>COMPLETE</ralph>"* ]]; then
          log_activity "✅ Agent signaled COMPLETE"
          echo "COMPLETE" 2>/dev/null || true
        fi

        # Check for gutter sigil
        if [[ "$text" == *"<ralph>GUTTER</ralph>"* ]]; then
          log_activity "🚨 Agent signaled GUTTER (stuck)"
          echo "GUTTER" 2>/dev/null || true
        fi

        # Check for stalled sigil (all tasks over pass threshold)
        if [[ "$text" == *"<ralph>STALLED</ralph>"* ]]; then
          log_activity "⏸️ Agent signaled STALLED (all tasks over pass limit)"
          echo "STALLED" 2>/dev/null || true
        fi

        # Check for empty sigil (no tasks in todo)
        if [[ "$text" == *"<ralph>EMPTY</ralph>"* ]]; then
          log_activity "📭 Agent signaled EMPTY (no tasks in todo)"
          echo "EMPTY" 2>/dev/null || true
        fi
      fi
      ;;

    "tool_call")
      if [[ "$subtype" == "started" ]]; then
        TOOL_CALLS=$((TOOL_CALLS + 1))

      elif [[ "$subtype" == "completed" ]]; then
        # Handle read tool completion
        if echo "$line" | jq -e '.tool_call.readToolCall.result.success' > /dev/null 2>&1; then
          local path=$(echo "$line" | jq -r '.tool_call.readToolCall.args.path // "unknown"' 2>/dev/null) || path="unknown"
          local lines=$(echo "$line" | jq -r '.tool_call.readToolCall.result.success.totalLines // 0' 2>/dev/null) || lines=0

          # Try to get actual content size, fall back to estimate
          local content_size=$(echo "$line" | jq -r '.tool_call.readToolCall.result.success.contentSize // 0' 2>/dev/null) || content_size=0

          # If no contentSize, try to measure actual content length
          if [[ $content_size -eq 0 ]]; then
            local content=$(echo "$line" | jq -r '.tool_call.readToolCall.result.success.content // ""' 2>/dev/null) || content=""
            if [[ -n "$content" ]]; then
              content_size=${#content}
            fi
          fi

          local bytes
          if [[ $content_size -gt 0 ]]; then
            bytes=$content_size
          else
            bytes=$((lines * 30))  # ~30 chars/line estimate (code averages ~25-30)
          fi
          BYTES_READ=$((BYTES_READ + bytes))

          local kb=$(echo "scale=1; $bytes / 1024" | bc 2>/dev/null || echo "$((bytes / 1024))")
          log_activity "READ $path ($lines lines, ~${kb}KB)"

        # Handle write tool completion
        elif echo "$line" | jq -e '.tool_call.writeToolCall.result.success' > /dev/null 2>&1; then
          local path=$(echo "$line" | jq -r '.tool_call.writeToolCall.args.path // "unknown"' 2>/dev/null) || path="unknown"
          local lines=$(echo "$line" | jq -r '.tool_call.writeToolCall.result.success.linesCreated // 0' 2>/dev/null) || lines=0
          local bytes=$(echo "$line" | jq -r '.tool_call.writeToolCall.result.success.fileSize // 0' 2>/dev/null) || bytes=0
          BYTES_WRITTEN=$((BYTES_WRITTEN + bytes))

          local kb=$(echo "scale=1; $bytes / 1024" | bc 2>/dev/null || echo "$((bytes / 1024))")
          log_activity "WRITE $path ($lines lines, ${kb}KB)"

          # Track for thrashing detection
          track_file_write "$path"

        # Handle shell tool completion
        elif echo "$line" | jq -e '.tool_call.shellToolCall.result' > /dev/null 2>&1; then
          local cmd=$(echo "$line" | jq -r '.tool_call.shellToolCall.args.command // "unknown"' 2>/dev/null) || cmd="unknown"
          local exit_code=$(echo "$line" | jq -r '.tool_call.shellToolCall.result.exitCode // 0' 2>/dev/null) || exit_code=0

          local stdout=$(echo "$line" | jq -r '.tool_call.shellToolCall.result.stdout // ""' 2>/dev/null) || stdout=""
          local stderr=$(echo "$line" | jq -r '.tool_call.shellToolCall.result.stderr // ""' 2>/dev/null) || stderr=""
          local output_chars=$((${#stdout} + ${#stderr}))
          SHELL_OUTPUT_CHARS=$((SHELL_OUTPUT_CHARS + output_chars))

          if [[ $exit_code -eq 0 ]]; then
            if [[ $output_chars -gt 1024 ]]; then
              log_activity "SHELL $cmd → exit 0 (${output_chars} chars output)"
            else
              log_activity "SHELL $cmd → exit 0"
            fi
          else
            log_activity "SHELL $cmd → exit $exit_code"
            track_shell_failure "$cmd" "$exit_code"
          fi
        fi

      fi
      ;;

    "result")
      local duration=$(echo "$line" | jq -r '.duration_ms // 0' 2>/dev/null) || duration=0
      local tokens=$(calc_tokens)
      log_activity "SESSION END: ${duration}ms, ~$tokens tokens used"
      ;;
  esac
}

# Main loop: read JSON lines from stdin
main() {
  # Initialize activity log for this session
  echo "" >> "$RALPH_DIR/activity.log"
  echo "═══════════════════════════════════════════════════════════════" >> "$RALPH_DIR/activity.log"
  echo "Ralph Session Started: $(date)" >> "$RALPH_DIR/activity.log"
  echo "═══════════════════════════════════════════════════════════════" >> "$RALPH_DIR/activity.log"

  # Track last token log time
  local last_token_log=$(date +%s)

  while IFS= read -r line; do
    process_line "$line"

    # Log token status every 30 seconds
    local now=$(date +%s)
    if [[ $((now - last_token_log)) -ge 30 ]]; then
      log_token_status
      last_token_log=$now
    fi
  done

  # Final token status
  log_token_status
}

main
