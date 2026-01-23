#!/bin/bash
# Ralph Wiggum: Continuous Development Loop
#
# Runs cursor-agent in a continuous loop, processing tasks from ralph-todo.json.
# Moves completed tasks to ralph-complete.json.
#
# Sprint-style workflow:
#   ralph-backlog.json  - Items not ready for work (future)
#   ralph-todo.json     - Items ready to be worked on (current sprint)
#   ralph-complete.json - Completed items
#
# Usage:
#   ./ralph.sh                              # Start from current directory
#   ./ralph.sh /path/to/project             # Start from specific project
#   ./ralph.sh -n 50 -m gpt-5.2-high        # Custom iterations and model
#   ./ralph.sh --branch feature/foo --pr   # Create branch and PR
#   ./ralph.sh -y                           # Skip confirmation (for scripting)
#
# Flags:
#   -n, --iterations N     Max iterations per task (default: 20)
#   -p, --passes N         Max passes before task is stalled (default: 3)
#   -m, --model MODEL      Model to use (default: opus-4.5-thinking)
#   --branch NAME          Create and work on a new branch
#   --pr                   Open PR when complete (requires --branch)
#   -y, --yes              Skip confirmation prompt
#   -h, --help             Show this help
#
# The loop runs continuously until stopped (Ctrl+C).
# When ralph-todo.json is empty or all tasks are stalled, it polls for new work.
#
# Requirements:
#   - Git repository
#   - cursor-agent CLI installed
#   - jq installed

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common functions
source "$SCRIPT_DIR/ralph-common.sh"

# =============================================================================
# FLAG PARSING
# =============================================================================

show_help() {
  cat << 'EOF'
Ralph Wiggum: Continuous Development Loop

Usage:
  ./ralph.sh [options] [workspace]

Options:
  -n, --iterations N     Max iterations per task (default: 20)
  -p, --passes N         Max passes before task stalls (default: 3)
  -m, --model MODEL      Model to use (default: opus-4.5-thinking)
  --branch NAME          Create and work on a new branch
  --pr                   Open PR when complete (requires --branch)
  -y, --yes              Skip confirmation prompt
  -h, --help             Show this help

Sprint Files:
  ralph-todo.json        Tasks to work on (current sprint)
  ralph-complete.json    Completed tasks
  ralph-backlog.json     Future tasks (not processed)

Examples:
  ./ralph.sh                                    # Start continuous loop
  ./ralph.sh -n 50 -p 5                         # 50 iterations/task, 5 passes before stall
  ./ralph.sh -m gpt-5.2-high                    # Use GPT model
  ./ralph.sh --branch feature/api --pr -y      # Scripted PR workflow
  
Environment:
  RALPH_MODEL            Override default model (same as -m flag)
  MAX_PASSES             Override pass threshold (same as -p flag)
  POLL_INTERVAL          Seconds to wait when no work (default: 30)

The loop runs continuously until stopped with Ctrl+C.
When no tasks are available, it polls for new work every POLL_INTERVAL seconds.
EOF
}

# Parse command line arguments
WORKSPACE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -n|--iterations)
      MAX_ITERATIONS="$2"
      shift 2
      ;;
    -p|--passes)
      MAX_PASSES="$2"
      shift 2
      ;;
    -m|--model)
      MODEL="$2"
      shift 2
      ;;
    --branch)
      USE_BRANCH="$2"
      shift 2
      ;;
    --pr)
      OPEN_PR=true
      shift
      ;;
    -y|--yes)
      SKIP_CONFIRM=true
      shift
      ;;
    -h|--help)
      show_help
      exit 0
      ;;
    -*)
      echo "Unknown option: $1"
      echo "Use -h for help."
      exit 1
      ;;
    *)
      # Positional argument = workspace
      WORKSPACE="$1"
      shift
      ;;
  esac
done

# =============================================================================
# MAIN
# =============================================================================

main() {
  # Resolve workspace
  if [[ -z "$WORKSPACE" ]]; then
    WORKSPACE="$(pwd)"
  elif [[ "$WORKSPACE" == "." ]]; then
    WORKSPACE="$(pwd)"
  else
    WORKSPACE="$(cd "$WORKSPACE" && pwd)"
  fi
  
  # Show banner
  show_banner
  
  # Check prerequisites
  if ! check_prerequisites "$WORKSPACE"; then
    exit 1
  fi
  
  # Validate: PR requires branch
  if [[ "$OPEN_PR" == "true" ]] && [[ -z "$USE_BRANCH" ]]; then
    echo "❌ --pr requires --branch"
    echo "   Example: ./ralph.sh --branch feature/foo --pr"
    exit 1
  fi
  
  # Initialize .ralph directory
  init_ralph_dir "$WORKSPACE"
  
  echo "Workspace: $WORKSPACE"
  echo ""
  
  # Show task summary
  local workable
  workable=$(show_task_summary "$WORKSPACE")
  
  echo "Max iter/task: $MAX_ITERATIONS"
  echo "Max passes:    $MAX_PASSES"
  echo "Poll interval: ${POLL_INTERVAL}s"
  [[ -n "$USE_BRANCH" ]] && echo "Branch:        $USE_BRANCH"
  [[ "$OPEN_PR" == "true" ]] && echo "Open PR:       Yes"
  echo ""
  
  # Note: Don't exit if no tasks - the loop will poll for new work
  
  # Confirm before starting (unless -y flag)
  if [[ "$SKIP_CONFIRM" != "true" ]]; then
    echo "This will run cursor-agent in a continuous loop."
    echo "Tasks from ralph-todo.json will be processed and moved to ralph-complete.json."
    echo "When no work is available, Ralph will poll every ${POLL_INTERVAL}s for new tasks."
    echo ""
    echo "Press Ctrl+C at any time to stop."
    echo ""
    echo "Tip: Use -y flag to skip this prompt."
    echo ""
    read -p "Start Ralph loop? [y/N] " -n 1 -r
    echo ""
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      echo "Aborted."
      exit 0
    fi
  fi
  
  # Run the continuous loop
  run_ralph_loop "$WORKSPACE" "$SCRIPT_DIR"
  exit $?
}

main
