#!/bin/bash

# === Environment Configuration ===

# 判斷運行環境：本地開發 vs Kubernetes 部署
if [ -f ".env" ]; then
    echo "📋 Loading local development environment from .env"
    # 安全載入 .env 文件，使用 source 避免複雜值解析問題
    set -a  # 自動 export 所有變數
    source ".env"
    set +a  # 關閉自動 export
    DEPLOY_ENV="local"
else
    echo "📋 Using deployment environment (ConfigMap/Secrets)"
    DEPLOY_ENV="deployment"
fi

# === Config ===
ATLASGO="atlas"

# 從環境變數讀取 MySQL 配置，提供默認值
MYSQL_HOST="${DB_HOST:-localhost}"
MYSQL_PORT="${DB_PORT:-3306}"
MYSQL_ROOT_USER="${DB_USER:-root}"
MYSQL_ROOT_PASSWORD="${DB_PASSWORD:-password}"
MYSQL_DATABASE="${DB_NAME:-fat_notification_cat}"

# Atlas 環境配置：local 或 deployment
ATLAS_ENV="${ATLAS_ENV:-${DEPLOY_ENV}}"

# 顯示當前配置信息（隱藏密碼）
echo "🔧 Configuration:"
echo "   Environment: ${DEPLOY_ENV}"
echo "   Atlas Environment: ${ATLAS_ENV}"
echo "   MySQL Host: ${MYSQL_HOST}"
echo "   MySQL Port: ${MYSQL_PORT}"
echo "   MySQL User: ${MYSQL_ROOT_USER}"
echo "   MySQL Database: ${MYSQL_DATABASE}"
echo "   MySQL Password: [HIDDEN]"
echo ""

# === Functions ===

# Helper function to build Atlas command with variables
build_atlas_cmd() {
  local base_cmd="$1"
  echo "$base_cmd --var db_host=\"$MYSQL_HOST\" --var db_user=\"$MYSQL_ROOT_USER\" --var db_password=\"$MYSQL_ROOT_PASSWORD\" --var db_name=\"$MYSQL_DATABASE\" --var db_port=\"$MYSQL_PORT\" --var atlas_dev_user=\"${ATLAS_DEV_USER:-dev_user}\" --var atlas_dev_password=\"${ATLAS_DEV_PASSWORD:-dev_password}\""
}


init_migration() {
  echo "📂 \033[1;36mInitialize migrations folder and SQL file...\033[0m"
  CMD=$(build_atlas_cmd "$ATLASGO migrate diff init --env \"$ATLAS_ENV\"")
  echo -e "👉 \033[1;33mExecuting:\033[0m $CMD"
  eval $CMD
  echo -e "✅ Initialize successfully\n"
}

apply_migration() {
  echo -e "💻 \033[1;36mApplying migrations to remote DB...\033[0m"
  CMD=$(build_atlas_cmd "$ATLASGO migrate apply --env \"$ATLAS_ENV\"")
  echo -e "👉 \033[1;33mExecuting:\033[0m $CMD"
  eval $CMD
  echo -e "✅ Migration applied successfully\n"
}

dry_run() {
  echo -e "💻 \033[1;36mPreviewing migration changes on remote DB...\033[0m"
  CMD=$(build_atlas_cmd "$ATLASGO schema apply --dry-run --env \"$ATLAS_ENV\"")
  echo -e "👉 \033[1;33mExecuting:\033[0m $CMD"
  eval $CMD
  echo -e "✅ Dry run completed\n"
}

generate_diff() {
  if [ -z "$1" ]; then
    echo -e "\033[1;31mMissing argument. Usage: ./migrate.sh gen <files prefix name>\033[0m"
    exit 1
  fi

  PREFIX_NAME="$1"
  echo -e "💻 \033[1;36mGenerating migration files...\033[0m"

  CMD=$(build_atlas_cmd "$ATLASGO migrate diff \"$PREFIX_NAME\" --env \"$ATLAS_ENV\"")
  echo -e "👉 \033[1;33mExecuting:\033[0m $CMD"
  eval $CMD
}

schema_inspect() {
  CMD=$(build_atlas_cmd "$ATLASGO schema inspect --env \"$ATLAS_ENV\"")
  echo -e "👉 \033[1;33mExecuting:\033[0m $CMD"
  eval $CMD
}

migrate_status() {
  CMD=$(build_atlas_cmd "$ATLASGO migrate status --env \"$ATLAS_ENV\"")
  echo -e "👉 \033[1;33mExecuting:\033[0m $CMD"
  eval $CMD
}

migrate_hash() {
  CMD=$(build_atlas_cmd "$ATLASGO migrate hash --env \"$ATLAS_ENV\"")
  echo -e "👉 \033[1;33mExecuting:\033[0m $CMD"
  eval $CMD
}

compare_diff() {
  echo -e "💻 \033[1;36mCompare migration files changes between local and remote...\033[0m"

  CMD="$ATLASGO schema diff \
    --from \"mysql://${MYSQL_ROOT_USER}:${MYSQL_ROOT_PASSWORD}@${MYSQL_HOST}:${MYSQL_PORT}/${MYSQL_DATABASE}?tls=true\" \
    --to \"file://migrations\" \
    --env \"$ATLAS_ENV\""

  echo -e "👉 \033[1;33mExecuting:\033[0m $CMD"

  # 執行並根據關鍵字著色輸出
  eval $CMD | while IFS= read -r line; do
    if [[ "$line" == *"DROP TABLE"* ]]; then
      echo -e "\033[1;31m$line\033[0m"
    elif [[ "$line" == *"CREATE TABLE"* ]]; then
      echo -e "\033[1;32m$line\033[0m"
    elif [[ "$line" == *"ALTER TABLE"* ]]; then
      echo -e "\033[1;35m$line\033[0m"
    else
      echo "$line"
    fi
  done
}

migrate_down() {
    local version="$1"
    local cmd=""

    if [[ -z "$version" ]]; then
      cmd=$(build_atlas_cmd "atlas migrate down --env \"$ATLAS_ENV\"")
    else
      cmd=$(build_atlas_cmd "atlas migrate down --env \"$ATLAS_ENV\" --to-version $version")
    fi

    echo -e "⚠️ \033[1;31mYou are about to execute a rollback:\033[0m"
    echo "$cmd"
    read -p "Are you sure you want to proceed? [Y/N] " confirm

    case "$confirm" in
      [Yy]* )
        echo "✅ Executing..."
        eval "$cmd"
        ;;
      * )
        echo "❌ Cancelled."
        exit 0
        ;;
    esac
}

# === Command Dispatcher ===

case "$1" in
  init)
    init_migration
    ;;
  apply)
    apply_migration
    ;;
  dry-run)
    dry_run
    ;;
  gen)
    generate_diff "$2"
    ;;
  inspect)
    schema_inspect
    ;;
  status)
    migrate_status
    ;;
  diff)
    compare_diff
    ;;
  rollback)
    migrate_down "$2"
    ;;
  hash)
    migrate_hash
    ;;
  *)
    echo "❗ Usage: bash $0 [init|apply|dry-run|gen|inspect|status|diff|rollback|hash]"
    exit 1
    ;;
esac