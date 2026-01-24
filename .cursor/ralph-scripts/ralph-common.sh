#!/bin/bash
# Ralph Wiggum: Common utilities and loop logic
#
# Shared functions for ralph.sh and ralph-setup.sh
# All state lives in .ralph/ within the project.
#
# Sprint-style workflow:
#   ralph-backlog.json  - Items not ready for work (future)
#   ralph-todo.json     - Items ready to be worked on (current sprint)
#   ralph-complete.json - Completed items

# =============================================================================
# CONFIGURATION (can be overridden before sourcing)
# =============================================================================

# Iteration limits (per task, not total)
MAX_ITERATIONS="${MAX_ITERATIONS:-20}"

# Pass threshold - skip tasks that have been attempted this many times
MAX_PASSES="${MAX_PASSES:-3}"

# Poll interval when no work available (seconds)
POLL_INTERVAL="${POLL_INTERVAL:-30}"

# Model selection
DEFAULT_MODEL="opus-4.5-thinking"
MODEL="${RALPH_MODEL:-$DEFAULT_MODEL}"

# Feature flags (set by caller)
USE_BRANCH="${USE_BRANCH:-}"
OPEN_PR="${OPEN_PR:-false}"
SKIP_CONFIRM="${SKIP_CONFIRM:-false}"

# =============================================================================
# BASIC HELPERS
# =============================================================================

# Cross-platform sed -i
sedi() {
  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "$@"
  else
    sed -i "$@"
  fi
}

# Get the .ralph directory for a workspace
get_ralph_dir() {
  local workspace="${1:-.}"
  echo "$workspace/.ralph"
}

# Get current iteration from .ralph/.iteration
get_iteration() {
  local workspace="${1:-.}"
  local state_file="$workspace/.ralph/.iteration"

  if [[ -f "$state_file" ]]; then
    cat "$state_file"
  else
    echo "0"
  fi
}

# Set iteration number
set_iteration() {
  local workspace="${1:-.}"
  local iteration="$2"
  local ralph_dir="$workspace/.ralph"

  mkdir -p "$ralph_dir"
  echo "$iteration" > "$ralph_dir/.iteration"
}

# Increment iteration and return new value
increment_iteration() {
  local workspace="${1:-.}"
  local current=$(get_iteration "$workspace")
  local next=$((current + 1))
  set_iteration "$workspace" "$next"
  echo "$next"
}


# =============================================================================
# LOGGING
# =============================================================================

# Log a message to activity.log
log_activity() {
  local workspace="${1:-.}"
  local message="$2"
  local ralph_dir="$workspace/.ralph"
  local timestamp=$(date '+%H:%M:%S')

  mkdir -p "$ralph_dir"
  echo "[$timestamp] $message" >> "$ralph_dir/activity.log"
}

# Log an error to errors.log
log_error() {
  local workspace="${1:-.}"
  local message="$2"
  local ralph_dir="$workspace/.ralph"
  local timestamp=$(date '+%H:%M:%S')

  mkdir -p "$ralph_dir"
  echo "[$timestamp] $message" >> "$ralph_dir/errors.log"
}

# Log to progress.md (called by the loop, not the agent)
log_progress() {
  local workspace="$1"
  local message="$2"
  local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
  local progress_file="$workspace/.ralph/progress.md"

  echo "" >> "$progress_file"
  echo "### $timestamp" >> "$progress_file"
  echo "$message" >> "$progress_file"
}

# =============================================================================
# INITIALIZATION
# =============================================================================

# Initialize .ralph directory with default files
init_ralph_dir() {
  local workspace="$1"
  local ralph_dir="$workspace/.ralph"

  mkdir -p "$ralph_dir"

  # Initialize progress.md if it doesn't exist
  if [[ ! -f "$ralph_dir/progress.md" ]]; then
    cat > "$ralph_dir/progress.md" << 'EOF'
# Progress Log

> Updated by the agent after significant work.

---

## Session History

EOF
  fi

  # Initialize guardrails.md if it doesn't exist
  if [[ ! -f "$ralph_dir/guardrails.md" ]]; then
    cat > "$ralph_dir/guardrails.md" << 'EOF'
# Ralph Guardrails (Signs)

> Lessons learned from past failures. READ THESE BEFORE ACTING.

## Core Signs

### Sign: Read Before Writing
- **Trigger**: Before modifying any file
- **Instruction**: Always read the existing file first
- **Added after**: Core principle

### Sign: Test After Changes
- **Trigger**: After any code change
- **Instruction**: Run tests to verify nothing broke
- **Added after**: Core principle

### Sign: Commit Checkpoints
- **Trigger**: Before risky changes
- **Instruction**: Commit current working state first
- **Added after**: Core principle

---

## Learned Signs

EOF
  fi

  # Initialize errors.log if it doesn't exist
  if [[ ! -f "$ralph_dir/errors.log" ]]; then
    cat > "$ralph_dir/errors.log" << 'EOF'
# Error Log

> Failures detected by stream-parser. Use to update guardrails.

EOF
  fi

  # Initialize activity.log if it doesn't exist
  if [[ ! -f "$ralph_dir/activity.log" ]]; then
    cat > "$ralph_dir/activity.log" << 'EOF'
# Activity Log

> Real-time tool call logging from stream-parser.

EOF
  fi
}

# =============================================================================
# SPRINT FILE MANAGEMENT
# =============================================================================

# Get file paths for sprint workflow
get_todo_file() {
  local workspace="${1:-.}"
  echo "$workspace/ralph-todo.json"
}

get_complete_file() {
  local workspace="${1:-.}"
  echo "$workspace/ralph-complete.json"
}

get_backlog_file() {
  local workspace="${1:-.}"
  echo "$workspace/ralph-backlog.json"
}

# Initialize sprint files if they don't exist
init_sprint_files() {
  local workspace="${1:-.}"
  local todo_file=$(get_todo_file "$workspace")
  local complete_file=$(get_complete_file "$workspace")
  local backlog_file=$(get_backlog_file "$workspace")

  # Create empty arrays if files don't exist
  [[ ! -f "$todo_file" ]] && echo "[]" > "$todo_file"
  [[ ! -f "$complete_file" ]] && echo "[]" > "$complete_file"
  [[ ! -f "$backlog_file" ]] && echo "[]" > "$backlog_file"
}

# =============================================================================
# TASK MANAGEMENT (SPRINT WORKFLOW)
# =============================================================================

# Check if there are any workable tasks in ralph-todo.json
# (pending/null status AND passes < MAX_PASSES)
has_workable_tasks() {
  local workspace="${1:-.}"
  local todo_file=$(get_todo_file "$workspace")

  if [[ ! -f "$todo_file" ]]; then
    echo "false"
    return
  fi

  local workable
  workable=$(jq --argjson max "$MAX_PASSES" '
    [.[] | select(
      (.status == "pending" or .status == null or .status == "in_progress") and
      ((.passes // 0) < $max)
    )] | length
  ' "$todo_file" 2>/dev/null) || workable=0

  if [[ "$workable" -gt 0 ]]; then
    echo "true"
  else
    echo "false"
  fi
}

# Check if there are any tasks at all in todo (including stalled ones)
has_any_todo_tasks() {
  local workspace="${1:-.}"
  local todo_file=$(get_todo_file "$workspace")

  if [[ ! -f "$todo_file" ]]; then
    echo "false"
    return
  fi

  local total
  total=$(jq 'length' "$todo_file" 2>/dev/null) || total=0

  if [[ "$total" -gt 0 ]]; then
    echo "true"
  else
    echo "false"
  fi
}

# Count tasks in todo file
# Returns: workable:stalled:total
count_todo_tasks() {
  local workspace="${1:-.}"
  local todo_file=$(get_todo_file "$workspace")

  if [[ ! -f "$todo_file" ]]; then
    echo "0:0:0"
    return
  fi

  local total workable stalled
  total=$(jq 'length' "$todo_file" 2>/dev/null) || total=0
  workable=$(jq --argjson max "$MAX_PASSES" '
    [.[] | select(
      (.status == "pending" or .status == null or .status == "in_progress") and
      ((.passes // 0) < $max)
    )] | length
  ' "$todo_file" 2>/dev/null) || workable=0
  stalled=$((total - workable))

  echo "$workable:$stalled:$total"
}

# Count completed tasks
count_complete_tasks() {
  local workspace="${1:-.}"
  local complete_file=$(get_complete_file "$workspace")

  if [[ ! -f "$complete_file" ]]; then
    echo "0"
    return
  fi

  jq 'length' "$complete_file" 2>/dev/null || echo "0"
}

# Move a completed task from todo to complete file
# Args: workspace, task_description (used to identify the task)
move_task_to_complete() {
  local workspace="$1"
  local description="$2"
  local todo_file=$(get_todo_file "$workspace")
  local complete_file=$(get_complete_file "$workspace")

  if [[ ! -f "$todo_file" ]] || [[ -z "$description" ]]; then
    return 1
  fi

  # Find the task by description
  local task
  task=$(jq --arg desc "$description" '
    .[] | select(.description == $desc)
  ' "$todo_file" 2>/dev/null)

  if [[ -z "$task" ]]; then
    log_error "$workspace" "Could not find task to move: $description"
    return 1
  fi

  # Add completion timestamp to the task
  local completed_task
  completed_task=$(echo "$task" | jq '. + {status: "completed", completed_at: now | todate}')

  # Remove from todo
  local new_todo
  new_todo=$(jq --arg desc "$description" '
    [.[] | select(.description != $desc)]
  ' "$todo_file")
  echo "$new_todo" > "$todo_file"

  # Append to complete
  local new_complete
  if [[ -f "$complete_file" ]]; then
    new_complete=$(jq --argjson task "$completed_task" '. + [$task]' "$complete_file")
  else
    new_complete=$(echo "[$completed_task]")
  fi
  echo "$new_complete" > "$complete_file"

  log_activity "$workspace" "Moved completed task to ralph-complete.json: $description"
  return 0
}

# Check task status - returns WORKABLE, STALLED, EMPTY, or NO_FILE
check_task_status() {
  local workspace="$1"
  local todo_file=$(get_todo_file "$workspace")

  if [[ ! -f "$todo_file" ]]; then
    echo "NO_FILE"
    return
  fi

  local counts=$(count_todo_tasks "$workspace")
  local workable=${counts%%:*}
  local rest=${counts#*:}
  local stalled=${rest%%:*}
  local total=${rest#*:}

  if [[ "$total" -eq 0 ]]; then
    echo "EMPTY"
  elif [[ "$workable" -gt 0 ]]; then
    echo "WORKABLE:$workable"
  else
    echo "STALLED:$stalled"
  fi
}

# Legacy aliases for compatibility
has_pending_tasks() {
  has_workable_tasks "$@"
}

check_task_complete() {
  local workspace="$1"
  local status=$(check_task_status "$workspace")

  case "$status" in
    WORKABLE:*) echo "INCOMPLETE:${status#WORKABLE:}" ;;
    STALLED:*) echo "STALLED:${status#STALLED:}" ;;
    EMPTY) echo "COMPLETE" ;;
    NO_FILE) echo "NO_FILE" ;;
    *) echo "UNKNOWN" ;;
  esac
}

count_criteria() {
  local workspace="${1:-.}"
  local counts=$(count_todo_tasks "$workspace")
  local workable=${counts%%:*}
  local total=${counts##*:}
  echo "$workable:$total"
}

# =============================================================================
# PROMPT BUILDING
# =============================================================================

# Build the Ralph prompt for an iteration (Sprint mode)
build_sprint_prompt() {
  local workspace="$1"
  local iteration="$2"

  cat << EOF
# Ralph Iteration $iteration (Sprint Mode)

You are an autonomous development agent using the Ralph methodology.
You are working from a sprint-style task workflow.

## Sprint Files

- \`ralph-todo.json\` - Tasks ready to work on (your focus)
- \`ralph-complete.json\` - Completed tasks (for reference)
- \`ralph-backlog.json\` - Future tasks (ignore for now)

## FIRST: Read & Analyze

Before doing anything:
1. Read \`ralph-todo.json\` - the current sprint tasks
2. Read \`.ralph/guardrails.md\` - lessons from past failures (FOLLOW THESE)
3. Read \`.ralph/progress.md\` - what's been accomplished
4. Read \`.ralph/errors.log\` - recent failures to avoid
5. **Explore the codebase** to understand current state

## YOUR PRIMARY DECISION: Choose the Most Important Task

After reading ralph-todo.json and exploring the codebase, YOU decide which task to work on.

**IMPORTANT: Skip tasks with passes >= $MAX_PASSES** - these are stalled and waiting for human review.

**Selection criteria** (in order of importance):
1. Tasks with passes < $MAX_PASSES (skip stalled tasks!)
2. What would provide the most value RIGHT NOW given the current code state?
3. What is blocking other important work?
4. What is partially implemented and close to completion?
5. What has the highest priority that can actually be started?

**You are NOT required to work in order.** Pick from ANYWHERE in the list based on what makes the most sense given the actual state of the project.

Once you've chosen a task:
1. Update ralph-todo.json: change that task's \`"status"\` to \`"in_progress"\`
2. Increment \`"passes"\` by 1 (or set to 1 if null)
3. Announce which task you're working on and WHY you chose it

## Working Directory (Critical)

You are already in a git repository. Work HERE, not in a subdirectory:

- Do NOT run \`git init\` - the repo already exists
- Do NOT run scaffolding commands that create nested directories (\`npx create-*\`, \`npm init\`, etc.)
- If you need to scaffold, use flags like \`--no-git\` or scaffold into the current directory (\`.\`)
- All code should live at the repo root or in subdirectories you create manually

## Git Protocol (Critical)

The orchestrator handles commits - one commit per completed task. Do NOT commit yourself.

- Do NOT run \`git commit\` - the orchestrator commits when you signal COMPLETE
- Do NOT push - the human will push when ready
- If you get rotated, the orchestrator commits your progress before the next agent starts

Your state is preserved through the orchestrator's commits and the progress.md file.

## Task Execution Protocol

1. **Choose a workable task** (passes < $MAX_PASSES, see selection criteria above)
2. Mark it \`"in_progress"\` in ralph-todo.json, increment \`"passes"\`
3. Complete all the steps listed for that task
4. Run tests after changes to verify nothing broke
5. **REVIEW your changes** before marking complete:
   - Re-read the code you wrote/modified
   - Look for bugs, edge cases, error handling gaps
   - Check for typos, off-by-one errors, null/undefined handling
   - Verify the code actually fulfills the task requirements
   - Fix any issues found before proceeding
6. When task is complete AND reviewed, **update ralph-todo.json**:
   - Change its \`"status"\` to \`"completed"\`
   - The orchestrator will move it to ralph-complete.json
7. Update \`.ralph/progress.md\` with what you accomplished
8. **Output \`<ralph>COMPLETE</ralph>\` immediately after completing ONE task**
   - This triggers the orchestrator to commit and move the task
   - Do NOT commit - the orchestrator handles the single commit per task
   - Do NOT try to do multiple tasks in one session
9. If stuck 3+ times on same issue: output \`<ralph>GUTTER</ralph>\`

## Task File Format

The ralph-todo.json file is an array of task objects:
\`\`\`json
[
  {
    "category": "backend",
    "description": "Task description here",
    "status": "pending",      // pending | in_progress | completed
    "priority": "high",       // high | medium | low (advisory)
    "steps": ["Step 1", "Step 2"],
    "dependencies": ["Other task description"],
    "passes": 0               // SKIP if >= $MAX_PASSES (stalled)
  }
]
\`\`\`

**Your job**:
1. Choose the most impactful WORKABLE task (passes < $MAX_PASSES)
2. Complete it and mark it \`"completed"\` in ralph-todo.json
3. If ALL tasks are stalled (passes >= $MAX_PASSES), output \`<ralph>STALLED</ralph>\`

## Learning from Failures

When something fails:
1. Check \`.ralph/errors.log\` for failure history
2. Figure out the root cause
3. Add a Sign to \`.ralph/guardrails.md\` using this format:

\`\`\`
### Sign: [Descriptive Name]
- **Trigger**: When this situation occurs
- **Instruction**: What to do instead
- **Added after**: Iteration $iteration - what happened
\`\`\`

Begin by reading the state files, exploring the code, then choose and execute the most important task.
EOF
}

# Build the Ralph prompt for an iteration
build_prompt() {
  local workspace="$1"
  local iteration="$2"

  # Check task status
  local status=$(check_task_status "$workspace")

  case "$status" in
    WORKABLE:*)
      # Has workable tasks - normal prompt
      build_sprint_prompt "$workspace" "$iteration"
      ;;
    STALLED:*)
      # All tasks are stalled (passes >= MAX_PASSES)
      local stalled_count=${status#STALLED:}
      cat << EOF
# Ralph Iteration $iteration

All $stalled_count tasks in ralph-todo.json are STALLED (passes >= $MAX_PASSES).

These tasks have been attempted too many times without success.
A human needs to review them - either:
- Reset \`passes\` to 0 to retry
- Move them to ralph-backlog.json
- Fix the underlying issue manually

Output: \`<ralph>STALLED</ralph>\`
EOF
      ;;
    EMPTY|NO_FILE)
      # No tasks in todo
      cat << EOF
# Ralph Iteration $iteration

No tasks in ralph-todo.json!

The sprint todo list is empty. Either:
- All tasks have been completed and moved to ralph-complete.json
- No tasks have been added to the sprint yet

Waiting for new tasks to be added to ralph-todo.json...

Output: \`<ralph>EMPTY</ralph>\`
EOF
      ;;
    *)
      # Unknown state
      cat << EOF
# Ralph Iteration $iteration

Unknown task state: $status

Check the sprint files:
- ralph-todo.json
- ralph-complete.json
- ralph-backlog.json

Output: \`<ralph>GUTTER</ralph>\`
EOF
      ;;
  esac
}

# =============================================================================
# SPINNER & WAITING
# =============================================================================

# Ralph Wiggum quotes for idle entertainment
RALPH_QUOTES=(
  "I'm learnding!"
  "Me fail English? That's unpossible!"
  "Hi, Super Nintendo Chalmers!"
  "I bent my wookiee."
  "My cat's breath smells like cat food."
  "I found a moonrock in my nose!"
  "That's where I saw the leprechaun."
  "I eated the purple berries."
  "Tastes like burning."
  "My daddy shoots people!"
  "I'm a brick!"
  "When I grow up, I want to be a principal or a caterpillar."
)

# Get a random Ralph quote
random_quote() {
  local idx=$((RANDOM % ${#RALPH_QUOTES[@]}))
  echo "${RALPH_QUOTES[$idx]}"
}

# Spinner to show the loop is alive (not frozen)
# Outputs to stderr so it's not captured by $()
spinner() {
  local workspace="$1"
  local spin='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
  local i=0
  while true; do
    printf "\r  🐛 Agent working... %s  (watch: tail -f %s/.ralph/activity.log)" "${spin:i++%${#spin}:1}" "$workspace" >&2
    sleep 0.1
  done
}

# Track if we've shown the waiting message (to avoid repeating)
WAITING_MSG_SHOWN=""

# Animated wait with countdown and quotes
# Usage: wait_with_countdown <seconds> <emoji> <message>
# Only shows the header message once; subsequent calls just do the countdown
wait_with_countdown() {
  local seconds="$1"
  local emoji="$2"
  local message="$3"
  local msg_key="${emoji}${message}"
  local spin='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
  local i=0

  # Only show the header if it's a new message or first time
  if [[ "$WAITING_MSG_SHOWN" != "$msg_key" ]]; then
    local quote=$(random_quote)
    echo ""
    echo "$emoji $message"
    echo "   💬 \"$quote\" - Ralph"
    echo ""
    WAITING_MSG_SHOWN="$msg_key"
  fi

  while [[ $seconds -gt 0 ]]; do
    printf "\r   %s Next check in %2ds... (Ctrl+C to stop, add tasks to ralph-todo.json to continue)" "${spin:i++%${#spin}:1}" "$seconds"
    sleep 1
    ((seconds--))
  done
  # Don't clear the line - let it show the countdown restarting
}

# Reset waiting state (call when work starts)
reset_waiting_state() {
  WAITING_MSG_SHOWN=""
}

# =============================================================================
# ITERATION RUNNER
# =============================================================================

# Run a single agent iteration
# Returns: signal (COMPLETE, GUTTER, STALLED, EMPTY, or empty)
run_iteration() {
  local workspace="$1"
  local iteration="$2"
  local session_id="${3:-}"
  local script_dir="${4:-$(dirname "${BASH_SOURCE[0]}")}"

  # Write prompt to temp file (command-line args can't handle multi-line prompts)
  local prompt_file="/tmp/ralph_prompt_$$.md"
  build_prompt "$workspace" "$iteration" > "$prompt_file"

  # Use /tmp for FIFO since WSL can't create pipes on Windows filesystems
  local fifo="/tmp/ralph_parser_fifo_$$"

  # Create named pipe for parser signals
  rm -f "$fifo"
  mkfifo "$fifo"

  # Use stderr for display (stdout is captured for signal)
  echo "" >&2
  echo "═══════════════════════════════════════════════════════════════════" >&2
  echo "🐛 Ralph Iteration $iteration" >&2
  echo "═══════════════════════════════════════════════════════════════════" >&2
  echo "" >&2
  echo "Workspace: $workspace" >&2
  echo "Model:     $MODEL" >&2
  echo "Monitor:   tail -f $workspace/.ralph/activity.log" >&2
  echo "" >&2

  # Log session start to progress.md
  log_progress "$workspace" "**Session $iteration started** (model: $MODEL)"

  # Build cursor-agent arguments
  local -a agent_args=(-p --force --output-format stream-json --model "$MODEL")

  if [[ -n "$session_id" ]]; then
    echo "Resuming session: $session_id" >&2
    agent_args+=(--resume "$session_id")
  fi

  # Change to workspace
  cd "$workspace"

  # Start spinner to show we're alive
  spinner "$workspace" &
  local spinner_pid=$!

  # Read prompt content
  local prompt_content
  prompt_content=$(cat "$prompt_file")

  # Start parser in background, reading from cursor-agent
  # Parser outputs to fifo, we read signals from fifo
  (
    cursor-agent "${agent_args[@]}" "$prompt_content" 2>&1 | "$script_dir/stream-parser.sh" "$workspace" > "$fifo"
    rm -f "$prompt_file"
  ) &
  local agent_pid=$!

  # Read signals from parser
  local signal=""
  while IFS= read -r line; do
    case "$line" in
      "GUTTER")
        printf "\r\033[K" >&2  # Clear spinner line
        echo "🚨 Gutter detected - agent may be stuck..." >&2
        signal="GUTTER"
        # Don't kill yet, let agent try to recover
        ;;
      "COMPLETE")
        printf "\r\033[K" >&2  # Clear spinner line
        echo "✅ Agent signaled completion!" >&2
        signal="COMPLETE"
        # Let agent finish gracefully
        ;;
      "STALLED")
        printf "\r\033[K" >&2  # Clear spinner line
        echo "⏸️ All tasks stalled (over pass limit)..." >&2
        signal="STALLED"
        ;;
      "EMPTY")
        printf "\r\033[K" >&2  # Clear spinner line
        echo "📭 No tasks in todo..." >&2
        signal="EMPTY"
        ;;
    esac
  done < "$fifo"

  # Wait for agent to finish
  wait $agent_pid 2>/dev/null || true

  # Stop spinner and clear line
  kill $spinner_pid 2>/dev/null || true
  wait $spinner_pid 2>/dev/null || true
  printf "\r\033[K" >&2  # Clear spinner line

  # Cleanup
  rm -f "$fifo"

  echo "$signal"
}

# =============================================================================
# MAIN LOOP
# =============================================================================

# Find and move completed tasks from todo to complete
# Scans ralph-todo.json for tasks with status "completed" and moves them
move_completed_tasks() {
  local workspace="$1"
  local todo_file=$(get_todo_file "$workspace")
  local complete_file=$(get_complete_file "$workspace")

  if [[ ! -f "$todo_file" ]]; then
    return
  fi

  # Find completed tasks
  local completed_tasks
  completed_tasks=$(jq '[.[] | select(.status == "completed")]' "$todo_file" 2>/dev/null)
  local count=$(echo "$completed_tasks" | jq 'length' 2>/dev/null) || count=0

  if [[ "$count" -eq 0 ]]; then
    return
  fi

  # Add completion timestamp to each
  completed_tasks=$(echo "$completed_tasks" | jq '[.[] | . + {completed_at: (now | todate)}]')

  # Build new todo (without completed tasks) but don't write yet
  local new_todo
  new_todo=$(jq '[.[] | select(.status != "completed")]' "$todo_file")

  # Append to complete file FIRST (if this fails, todo remains intact)
  if [[ -f "$complete_file" ]]; then
    local new_complete
    new_complete=$(jq --argjson tasks "$completed_tasks" '. + $tasks' "$complete_file")
    echo "$new_complete" > "$complete_file"
  else
    echo "$completed_tasks" > "$complete_file"
  fi

  # Only now remove from todo (complete file write succeeded)
  echo "$new_todo" > "$todo_file"

  log_activity "$workspace" "Moved $count completed task(s) to ralph-complete.json"
  echo "📦 Moved $count completed task(s) to ralph-complete.json"
}

# Run the main Ralph loop (continuous mode)
# Args: workspace
# Uses global: MAX_ITERATIONS, MODEL, USE_BRANCH, OPEN_PR, POLL_INTERVAL
run_ralph_loop() {
  local workspace="$1"
  local script_dir="${2:-$(dirname "${BASH_SOURCE[0]}")}"

  # Commit any uncommitted work first
  cd "$workspace"
  if [[ -n "$(git status --porcelain 2>/dev/null)" ]]; then
    echo "📦 Committing uncommitted changes..."
    git add -A
    git commit -m "ralph: initial commit before loop" || true
  fi

  # Create branch if requested
  if [[ -n "$USE_BRANCH" ]]; then
    echo "🌿 Creating branch: $USE_BRANCH"
    git checkout -b "$USE_BRANCH" 2>/dev/null || git checkout "$USE_BRANCH"
  fi

  echo ""
  echo "🚀 Starting Ralph loop (continuous mode)..."
  echo "   Press Ctrl+C to stop"
  echo ""

  # Main loop - runs continuously until stopped
  local iteration=1
  local session_id=""
  local task_iterations=0  # iterations spent on current task

  while true; do
    # Check task status before running
    local task_status=$(check_task_status "$workspace")

    case "$task_status" in
      WORKABLE:*)
        # Has work to do - reset waiting state and continue
        reset_waiting_state
        ;;
      STALLED:*)
        local stalled_count=${task_status#STALLED:}
        wait_with_countdown "$POLL_INTERVAL" "⏸️" "All $stalled_count tasks stalled (passes >= $MAX_PASSES). Waiting for human review..."
        continue
        ;;
      EMPTY|NO_FILE)
        wait_with_countdown "$POLL_INTERVAL" "📭" "No tasks in ralph-todo.json. Waiting for new tasks..."
        continue
        ;;
      *)
        echo "⚠️  Unknown task status: $task_status"
        sleep "$POLL_INTERVAL"
        continue
        ;;
    esac

    # Check iteration limit for current task
    if [[ $task_iterations -ge $MAX_ITERATIONS ]]; then
      log_progress "$workspace" "**Task iteration limit** - ⚠️ Max iterations ($MAX_ITERATIONS) for task"
      echo ""
      echo "⚠️  Max iterations ($MAX_ITERATIONS) reached for current task."
      echo "   Task passes will be incremented. Moving to next task..."
      task_iterations=0
      session_id=""
      continue
    fi

    # Run iteration
    local signal
    signal=$(run_iteration "$workspace" "$iteration" "$session_id" "$script_dir")
    task_iterations=$((task_iterations + 1))

    # Move any completed tasks to complete file
    move_completed_tasks "$workspace"

    # Commit all changes (code + JSON) in one commit per task
    if [[ -n "$(git -C "$workspace" status --porcelain 2>/dev/null)" ]]; then
      # Get the task description for the commit message
      local task_desc
      task_desc=$(jq -r '.[-1].description // "task"' "$workspace/ralph-complete.json" 2>/dev/null | tail -1) || task_desc="task"
      # Truncate to reasonable length for commit message
      task_desc="${task_desc:0:60}"

      git -C "$workspace" add -A
      git -C "$workspace" commit -m "ralph: $task_desc" 2>/dev/null || true
    fi

    # Re-check task status after iteration
    task_status=$(check_task_status "$workspace")

    # Handle signals
    case "$signal" in
      "COMPLETE")
        # Agent signaled completion of a task
        log_progress "$workspace" "**Session $iteration ended** - ✅ Task complete"
        echo ""
        echo "✅ Task completed!"

        # Reset task iteration counter for next task
        task_iterations=0
        session_id=""

        # Check if more work available
        if [[ "$task_status" == WORKABLE:* ]]; then
          local remaining=${task_status#WORKABLE:}
          echo "   $remaining more tasks to work on."
          echo "   Starting fresh context for next task..."
        else
          echo "   No more workable tasks. Waiting for new work..."
        fi
        ;;
      "STALLED")
        # Agent detected all tasks are stalled
        log_progress "$workspace" "**Session $iteration ended** - ⏸️ All tasks stalled"
        echo ""
        echo "⏸️  All tasks stalled. Waiting for human review..."
        task_iterations=0
        session_id=""
        ;;
      "EMPTY")
        # Agent detected no tasks
        log_progress "$workspace" "**Session $iteration ended** - 📭 No tasks"
        echo ""
        echo "📭 No tasks in todo. Waiting for new work..."
        task_iterations=0
        session_id=""
        ;;
      "GUTTER")
        log_progress "$workspace" "**Session $iteration ended** - 🚨 GUTTER (agent stuck)"
        echo ""
        echo "🚨 Gutter detected. Check .ralph/errors.log for details."
        echo "   Task passes will be incremented. Continuing..."
        # Commit any partial work before moving on
        if [[ -n "$(git -C "$workspace" status --porcelain 2>/dev/null)" ]]; then
          git -C "$workspace" add -A
          git -C "$workspace" commit -m "ralph: checkpoint before gutter recovery" 2>/dev/null || true
        fi
        task_iterations=0
        session_id=""
        ;;
      *)
        # Agent finished naturally
        if [[ "$task_status" == WORKABLE:* ]]; then
          local remaining=${task_status#WORKABLE:}
          log_progress "$workspace" "**Session $iteration ended** - Agent finished ($remaining remaining)"
          echo ""
          echo "📋 Agent finished. $remaining tasks remaining."
        fi
        ;;
    esac

    iteration=$((iteration + 1))

    # Brief pause between iterations
    sleep 2
  done
}

# =============================================================================
# PREREQUISITE CHECKS
# =============================================================================

# Check all prerequisites, exit with error message if any fail
check_prerequisites() {
  local workspace="$1"
  local todo_file=$(get_todo_file "$workspace")

  # Check for jq (required for JSON parsing)
  if ! command -v jq &> /dev/null; then
    echo "❌ jq not found (required for JSON parsing)"
    echo ""
    echo "Install via:"
    echo "  macOS:  brew install jq"
    echo "  Ubuntu: apt install jq"
    echo "  Windows: choco install jq"
    return 1
  fi

  # Check for cursor-agent CLI
  if ! command -v cursor-agent &> /dev/null; then
    echo "❌ cursor-agent CLI not found"
    echo ""
    echo "Install via:"
    echo "  curl https://cursor.com/install -fsS | bash"
    return 1
  fi

  # Check for git repo
  if ! git -C "$workspace" rev-parse --git-dir > /dev/null 2>&1; then
    echo "❌ Not a git repository"
    echo "   Ralph requires git for state persistence."
    return 1
  fi

  # Initialize sprint files if needed
  init_sprint_files "$workspace"

  # Validate ralph-todo.json structure (if it has content)
  if [[ -f "$todo_file" ]]; then
    if ! jq empty "$todo_file" 2>/dev/null; then
      echo "❌ ralph-todo.json is not valid JSON"
      echo ""
      echo "Check syntax at: https://jsonlint.com/"
      return 1
    fi

    # Check it's an array
    local is_array=$(jq 'type == "array"' "$todo_file" 2>/dev/null)
    if [[ "$is_array" != "true" ]]; then
      echo "❌ ralph-todo.json must be a JSON array of tasks"
      return 1
    fi
  fi

  # Validate ralph-complete.json structure (if it exists)
  local complete_file=$(get_complete_file "$workspace")
  if [[ -f "$complete_file" ]]; then
    if ! jq empty "$complete_file" 2>/dev/null; then
      echo "❌ ralph-complete.json is not valid JSON"
      echo ""
      echo "Check syntax at: https://jsonlint.com/"
      return 1
    fi

    # Check it's an array
    local is_array=$(jq 'type == "array"' "$complete_file" 2>/dev/null)
    if [[ "$is_array" != "true" ]]; then
      echo "❌ ralph-complete.json must be a JSON array of tasks"
      return 1
    fi
  fi

  # Show status
  local counts=$(count_todo_tasks "$workspace")
  local workable=${counts%%:*}
  local rest=${counts#*:}
  local stalled=${rest%%:*}
  local total=${rest#*:}
  local completed=$(count_complete_tasks "$workspace")

  echo "✓ Sprint files ready"
  echo "  Todo:     $total tasks ($workable workable, $stalled stalled)"
  echo "  Complete: $completed tasks"
  echo "  Pass threshold: $MAX_PASSES"

  return 0
}

# =============================================================================
# DISPLAY HELPERS
# =============================================================================

# Show sprint task summary
# Display output goes to stderr, workable count goes to stdout for capture
show_sprint_summary() {
  local workspace="$1"
  local todo_file=$(get_todo_file "$workspace")
  local complete_file=$(get_complete_file "$workspace")

  echo "📋 Sprint Todo List:" >&2
  echo "─────────────────────────────────────────────────────────────────" >&2

  if [[ -f "$todo_file" ]]; then
    # Show task summary using jq (suppress jq errors, output to stderr for display)
    jq -r --argjson max "$MAX_PASSES" '
      to_entries | .[] |
      "\(.key + 1). [\(
        if .value.status == "completed" then "✓"
        elif .value.status == "in_progress" then "→"
        elif (.value.passes // 0) >= $max then "⏸"
        else " "
        end
      )] \(.value.description) (\(.value.priority // "medium"))\(
        if (.value.passes // 0) > 0 then " [passes: \(.value.passes // 0)]" else "" end
      )\(
        if (.value.passes // 0) >= $max then " STALLED" else "" end
      )"
    ' "$todo_file" 2>/dev/null >&2 || echo "(error reading ralph-todo.json)" >&2
  else
    echo "(no tasks in todo)" >&2
  fi

  echo "─────────────────────────────────────────────────────────────────" >&2
  echo "" >&2

  # Count tasks
  local counts=$(count_todo_tasks "$workspace")
  local workable=${counts%%:*}
  local rest=${counts#*:}
  local stalled=${rest%%:*}
  local total=${rest#*:}
  local completed=$(count_complete_tasks "$workspace")

  echo "Todo:        $total tasks ($workable workable, $stalled stalled)" >&2
  echo "Completed:   $completed tasks (in ralph-complete.json)" >&2
  echo "Pass limit:  $MAX_PASSES (tasks stall after this many attempts)" >&2
  echo "Model:       $MODEL" >&2
  echo "" >&2

  # Return workable count on stdout for caller to capture
  echo "$workable"
}

# Show task summary (alias for sprint summary)
show_task_summary() {
  local workspace="$1"
  show_sprint_summary "$workspace"
}

# Show Ralph banner
show_banner() {
  echo "═══════════════════════════════════════════════════════════════════"
  echo "🐛 Ralph Wiggum: Autonomous Development Loop"
  echo "═══════════════════════════════════════════════════════════════════"
  echo ""
  echo "  \"That's the beauty of Ralph - the technique is deterministically"
  echo "   bad in an undeterministic world.\""
  echo ""
  echo "═══════════════════════════════════════════════════════════════════"
  echo ""
}
