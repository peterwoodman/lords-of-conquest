#!/bin/bash
# Ralph Wiggum: Single Iteration (Human-in-the-Loop)
#
# Runs exactly ONE iteration of the Ralph loop, then stops.
# Useful for testing your task definition before going AFK.
#
# Sprint-style workflow:
#   ralph-todo.json     - Tasks to work on
#   ralph-complete.json - Completed tasks
#
# Usage:
#   ./ralph-once.sh                    # Run single iteration
#   ./ralph-once.sh /path/to/project   # Run in specific project
#   ./ralph-once.sh -m gpt-5.2-high    # Use specific model
#
# After running:
#   - Review the changes made
#   - Check git log for commits
#   - If satisfied, run ralph-setup.sh or ralph.sh for continuous loop
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
Ralph Wiggum: Single Iteration (Human-in-the-Loop)

Runs exactly ONE iteration, then stops for review.
This is the recommended way to test your task definition.

Usage:
  ./ralph-once.sh [options] [workspace]

Options:
  -m, --model MODEL      Model to use (default: opus-4.5-thinking)
  -p, --passes N         Max passes before task stalls (default: 3)
  -h, --help             Show this help

Sprint Files:
  ralph-todo.json        Tasks to work on (current sprint)
  ralph-complete.json    Completed tasks

Examples:
  ./ralph-once.sh                        # Run one iteration
  ./ralph-once.sh -m sonnet-4.5-thinking # Use Sonnet model
  
After reviewing the results:
  - If satisfied: run ./ralph.sh for continuous loop
  - If issues: fix them, update ralph-todo.json or guardrails, run again
EOF
}

# Parse command line arguments
WORKSPACE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -m|--model)
      MODEL="$2"
      shift 2
      ;;
    -p|--passes)
      MAX_PASSES="$2"
      shift 2
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
  echo "═══════════════════════════════════════════════════════════════════"
  echo "🐛 Ralph Wiggum: Single Iteration (Human-in-the-Loop)"
  echo "═══════════════════════════════════════════════════════════════════"
  echo ""
  echo "  This runs ONE iteration, then stops for your review."
  echo "  Use this to test your task before going AFK."
  echo ""
  echo "═══════════════════════════════════════════════════════════════════"
  echo ""
  
  # Check prerequisites
  if ! check_prerequisites "$WORKSPACE"; then
    exit 1
  fi
  
  # Initialize .ralph directory
  init_ralph_dir "$WORKSPACE"
  
  echo "Workspace: $WORKSPACE"
  echo "Model:     $MODEL"
  echo "Passes:    $MAX_PASSES (stall threshold)"
  echo ""
  
  # Show task summary
  local workable
  workable=$(show_task_summary "$WORKSPACE")
  
  # Get task counts
  local task_status=$(check_task_status "$WORKSPACE")
  
  case "$task_status" in
    WORKABLE:*)
      # Good to go
      ;;
    STALLED:*)
      echo "⏸️  All tasks are stalled (passes >= $MAX_PASSES)"
      echo "   Reset passes to 0 on a task to retry it."
      exit 0
      ;;
    EMPTY|NO_FILE)
      echo "📭 No tasks in ralph-todo.json"
      echo "   Add tasks to ralph-todo.json first."
      exit 0
      ;;
    *)
      echo "⚠️  Unknown task status: $task_status"
      exit 1
      ;;
  esac
  
  # Confirm
  read -p "Run single iteration? [Y/n] " -n 1 -r
  echo ""
  
  if [[ $REPLY =~ ^[Nn]$ ]]; then
    echo "Aborted."
    exit 0
  fi
  
  # Commit any uncommitted work first
  cd "$WORKSPACE"
  if [[ -n "$(git status --porcelain 2>/dev/null)" ]]; then
    echo "📦 Committing uncommitted changes..."
    git add -A
    git commit -m "ralph: checkpoint before single iteration" || true
  fi
  
  echo ""
  echo "🚀 Running single iteration..."
  echo ""
  
  # Run exactly one iteration
  local signal
  signal=$(run_iteration "$WORKSPACE" "1" "" "$SCRIPT_DIR")
  
  # Move any completed tasks
  move_completed_tasks "$WORKSPACE"
  
  # Check result
  local task_status
  task_status=$(check_task_status "$WORKSPACE")
  
  echo ""
  echo "═══════════════════════════════════════════════════════════════════"
  echo "📋 Single Iteration Complete"
  echo "═══════════════════════════════════════════════════════════════════"
  echo ""
  
  case "$signal" in
    "COMPLETE")
      echo "✅ Task marked complete and moved to ralph-complete.json"
      case "$task_status" in
        WORKABLE:*)
          local remaining=${task_status#WORKABLE:}
          echo "   $remaining more tasks ready to work on."
          ;;
        STALLED:*)
          echo "   Remaining tasks are stalled (passes >= $MAX_PASSES)."
          ;;
        EMPTY)
          echo "   No more tasks in ralph-todo.json!"
          ;;
      esac
      ;;
    "STALLED")
      echo "⏸️  All tasks are stalled (passes >= $MAX_PASSES)"
      echo ""
      echo "Reset passes to 0 on a task to retry it."
      ;;
    "GUTTER")
      echo "🚨 Gutter detected - agent got stuck."
      echo ""
      echo "Review .ralph/errors.log and consider:"
      echo "  1. Adding a guardrail to .ralph/guardrails.md"
      echo "  2. Simplifying the task"
      echo "  3. Fixing the blocking issue manually"
      ;;
    *)
      case "$task_status" in
        WORKABLE:*)
          local remaining=${task_status#WORKABLE:}
          echo "Agent finished. $remaining tasks remaining."
          ;;
        STALLED:*)
          echo "Agent finished. Remaining tasks are stalled."
          ;;
        EMPTY)
          echo "Agent finished. No more tasks in todo!"
          ;;
      esac
      ;;
  esac
  
  echo ""
  echo "Review the changes:"
  echo "  • git log --oneline -5     # See recent commits"
  echo "  • git diff HEAD~1          # See changes"
  echo "  • cat .ralph/progress.md   # See progress log"
  echo ""
  echo "Next steps:"
  echo "  • If satisfied: ./ralph.sh  # Run continuous loop"
  echo "  • If issues: fix, update ralph-todo.json or guardrails, ./ralph-once.sh again"
  echo ""
}

main
