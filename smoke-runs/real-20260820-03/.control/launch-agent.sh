#!/bin/sh
set +e
"$WRITE_UUTER_CODEX" -s workspace-write -a never -C "$WRITE_UUTER_RUN_DIR" exec --ephemeral --skip-git-repo-check - < "$WRITE_UUTER_PROMPT" > "$WRITE_UUTER_LOG_FILE" 2>&1
status=$?
printf '%s\n' "$status" > "$WRITE_UUTER_EXIT_FILE"
exit "$status"
